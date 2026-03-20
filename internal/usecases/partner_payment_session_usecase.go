package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	domainentities "payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	domainrepos "payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/domain/services"
	"payment-kita.backend/pkg/utils"
)

const defaultPartnerCheckoutBaseURL = "https://pay.paymentkita.com/checkout"

type CreatePartnerPaymentSessionInput struct {
	MerchantID        uuid.UUID
	QuoteID           uuid.UUID
	DestWallet        string
	DestChainOverride string
	DestTokenOverride string
}

type CreatePartnerPaymentSessionOutput struct {
	PaymentID          string    `json:"payment_id"`
	MerchantID         string    `json:"merchant_id"`
	InvoiceCurrency    string    `json:"invoice_currency"`
	InvoiceAmount      string    `json:"invoice_amount"`
	Amount             string    `json:"amount"`
	AmountDecimals     int       `json:"amount_decimals"`
	DestChain          string    `json:"dest_chain"`
	DestToken          string    `json:"dest_token"`
	DestWallet         string    `json:"dest_wallet"`
	ExpireTime         time.Time `json:"expire_time"`
	PaymentURL         string    `json:"payment_url"`
	PaymentCode        string    `json:"payment_code"`
	PaymentInstruction struct {
		ChainID    string `json:"chain_id"`
		To         string `json:"to,omitempty"`
		Value      string `json:"value,omitempty"`
		Data       string `json:"data,omitempty"`
		ProgramID  string `json:"program_id,omitempty"`
		DataBase58 string `json:"data_base58,omitempty"`
		DataBase64 string `json:"data_base64,omitempty"`
	} `json:"payment_instruction"`
	Quote struct {
		QuoteID        string    `json:"quote_id"`
		PriceSource    string    `json:"price_source"`
		QuoteRate      string    `json:"quote_rate"`
		QuoteExpiresAt time.Time `json:"quote_expires_at"`
	} `json:"quote"`
	Status string `json:"status"`
}

type GetPartnerPaymentSessionOutput struct {
	PaymentID          string    `json:"payment_id"`
	Status             string    `json:"status"`
	Amount             string    `json:"amount"`
	AmountDecimals     int       `json:"amount_decimals"`
	DestChain          string    `json:"dest_chain"`
	DestToken          string    `json:"dest_token"`
	DestWallet         string    `json:"dest_wallet"`
	ExpiresAt          time.Time `json:"expires_at"`
	PaymentURL         string    `json:"payment_url"`
	PaymentCode        string    `json:"payment_code"`
	PaymentInstruction struct {
		ChainID    string `json:"chain_id"`
		To         string `json:"to,omitempty"`
		Value      string `json:"value,omitempty"`
		Data       string `json:"data,omitempty"`
		ProgramID  string `json:"program_id,omitempty"`
		DataBase58 string `json:"data_base58,omitempty"`
		DataBase64 string `json:"data_base64,omitempty"`
	} `json:"payment_instruction"`
}

type ResolvePartnerPaymentCodeInput struct {
	PaymentCode string
	PayerWallet string
}

type ResolvePartnerPaymentCodeOutput struct {
	PaymentID          string    `json:"payment_id"`
	MerchantID         string    `json:"merchant_id"`
	Amount             string    `json:"amount"`
	AmountDecimals     int       `json:"amount_decimals"`
	DestChain          string    `json:"dest_chain"`
	DestToken          string    `json:"dest_token"`
	DestWallet         string    `json:"dest_wallet"`
	ExpiresAt          time.Time `json:"expires_at"`
	PaymentInstruction struct {
		ChainID    string `json:"chain_id"`
		To         string `json:"to,omitempty"`
		Value      string `json:"value,omitempty"`
		Data       string `json:"data,omitempty"`
		ProgramID  string `json:"program_id,omitempty"`
		DataBase58 string `json:"data_base58,omitempty"`
		DataBase64 string `json:"data_base64,omitempty"`
	} `json:"payment_instruction"`
}

type PartnerPaymentSessionUsecase struct {
	quoteRepo           domainrepos.PaymentQuoteRepository
	sessionRepo         domainrepos.PartnerPaymentSessionRepository
	paymentRequestRepo  domainrepos.PaymentRequestRepository
	contractRepo        domainrepos.SmartContractRepository
	tokenRepo           domainrepos.TokenRepository
	chainRepo           domainrepos.ChainRepository
	uow                 domainrepos.UnitOfWork
	jweService          services.JWEService
	paymentRequestLogic *PaymentRequestUsecase
	chainResolver       *ChainResolver
	checkoutBaseURL     string
}

func NewPartnerPaymentSessionUsecase(
	quoteRepo domainrepos.PaymentQuoteRepository,
	sessionRepo domainrepos.PartnerPaymentSessionRepository,
	paymentRequestRepo domainrepos.PaymentRequestRepository,
	contractRepo domainrepos.SmartContractRepository,
	tokenRepo domainrepos.TokenRepository,
	chainRepo domainrepos.ChainRepository,
	uow domainrepos.UnitOfWork,
	jweService services.JWEService,
	paymentRequestLogic *PaymentRequestUsecase,
	checkoutBaseURL string,
) *PartnerPaymentSessionUsecase {
	baseURL := strings.TrimSpace(checkoutBaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("PARTNER_CHECKOUT_BASE_URL"))
	}
	if baseURL == "" {
		baseURL = defaultPartnerCheckoutBaseURL
	}
	return &PartnerPaymentSessionUsecase{
		quoteRepo:           quoteRepo,
		sessionRepo:         sessionRepo,
		paymentRequestRepo:  paymentRequestRepo,
		contractRepo:        contractRepo,
		tokenRepo:           tokenRepo,
		chainRepo:           chainRepo,
		uow:                 uow,
		jweService:          jweService,
		paymentRequestLogic: paymentRequestLogic,
		chainResolver:       NewChainResolver(chainRepo),
		checkoutBaseURL:     strings.TrimRight(baseURL, "/"),
	}
}

func (u *PartnerPaymentSessionUsecase) CreateSession(ctx context.Context, input *CreatePartnerPaymentSessionInput) (*CreatePartnerPaymentSessionOutput, error) {
	if input == nil {
		return nil, domainerrors.BadRequest("input is required")
	}
	if input.MerchantID == uuid.Nil {
		return nil, domainerrors.Forbidden("merchant context required")
	}
	if input.QuoteID == uuid.Nil {
		return nil, domainerrors.BadRequest("quote_id is required")
	}
	if strings.TrimSpace(input.DestWallet) == "" {
		return nil, domainerrors.BadRequest("dest_wallet is required")
	}
	if u.uow == nil {
		return nil, domainerrors.InternalServerError("unit of work is not configured")
	}
	if u.jweService == nil {
		return nil, domainerrors.InternalServerError("jwe service is not configured")
	}
	if u.paymentRequestLogic == nil {
		return nil, domainerrors.InternalServerError("payment request helper is not configured")
	}

	var output *CreatePartnerPaymentSessionOutput
	err := u.uow.Do(u.uow.WithLock(ctx), func(txCtx context.Context) error {
		quote, err := u.quoteRepo.GetByID(txCtx, input.QuoteID)
		if err != nil {
			return domainerrors.NotFound("quote not found")
		}
		if quote.MerchantID != input.MerchantID {
			return domainerrors.Forbidden("quote does not belong to merchant")
		}
		if quote.Status != domainentities.PaymentQuoteStatusActive {
			return domainerrors.BadRequest("quote is no longer active")
		}
		now := time.Now().UTC()
		if now.After(quote.ExpiresAt) {
			return domainerrors.BadRequest("quote has expired")
		}

		selectedChainID, selectedChainCAIP2, err := u.chainResolver.ResolveFromAny(txCtx, quote.SelectedChainID)
		if err != nil {
			return domainerrors.BadRequest(fmt.Sprintf("invalid selected chain on quote: %v", err))
		}
		selectedToken, err := u.tokenRepo.GetByAddress(txCtx, quote.SelectedTokenAddress, selectedChainID)
		if err != nil || selectedToken == nil {
			return domainerrors.BadRequest("quoted token no longer supported")
		}

		paymentRequest := &domainentities.PaymentRequest{
			ID:            utils.GenerateUUIDv7(),
			MerchantID:    input.MerchantID,
			ChainID:       selectedChainID,
			NetworkID:     selectedChainCAIP2,
			TokenID:       selectedToken.ID,
			TokenAddress:  selectedToken.ContractAddress,
			WalletAddress: strings.TrimSpace(input.DestWallet),
			Amount:        quote.QuotedAmount,
			Decimals:      quote.SelectedTokenDecimals,
			Status:        domainentities.PaymentRequestStatusPending,
			ExpiresAt:     quote.ExpiresAt,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := u.paymentRequestRepo.Create(txCtx, paymentRequest); err != nil {
			return domainerrors.InternalServerError(fmt.Sprintf("failed to create payment request primitive: %v", err))
		}

		contract, _ := u.contractRepo.GetActiveContract(txCtx, selectedChainID, domainentities.ContractTypeGateway)
		txData := u.paymentRequestLogic.buildTransactionData(paymentRequest, contract)
		if txData == nil {
			return domainerrors.InternalServerError("failed to build payment instruction")
		}

		session := &domainentities.PartnerPaymentSession{
			ID:                    utils.GenerateUUIDv7(),
			MerchantID:            input.MerchantID,
			QuoteID:               &quote.ID,
			PaymentRequestID:      &paymentRequest.ID,
			InvoiceCurrency:       quote.InvoiceCurrency,
			InvoiceAmount:         quote.InvoiceAmount,
			SelectedChainID:       quote.SelectedChainID,
			SelectedTokenAddress:  quote.SelectedTokenAddress,
			SelectedTokenSymbol:   quote.SelectedTokenSymbol,
			SelectedTokenDecimals: quote.SelectedTokenDecimals,
			DestChain:             coalesceString(strings.TrimSpace(input.DestChainOverride), quote.SelectedChainID),
			DestToken:             coalesceString(strings.TrimSpace(input.DestTokenOverride), quote.SelectedTokenAddress),
			DestWallet:            strings.TrimSpace(input.DestWallet),
			PaymentAmount:         quote.QuotedAmount,
			PaymentAmountDecimals: quote.SelectedTokenDecimals,
			Status:                domainentities.PartnerPaymentSessionStatusPending,
			PaymentURL:            u.checkoutBaseURL + "/" + paymentRequest.ID.String(),
			InstructionTo:         txData.To,
			InstructionValue:      "0",
			InstructionDataHex:    txData.Hex,
			InstructionDataBase58: txData.Base58,
			InstructionDataBase64: txData.Base64,
			QuoteRate:             stringPtr(quote.QuoteRate),
			QuoteSource:           stringPtr(quote.PriceSource),
			QuoteRoute:            stringPtr(quote.Route),
			QuoteExpiresAt:        &quote.ExpiresAt,
			QuoteSnapshotJSON:     mustJSON(map[string]interface{}{"quote_id": quote.ID.String(), "price_source": quote.PriceSource, "quote_rate": quote.QuoteRate, "route": quote.Route, "quote_expires_at": quote.ExpiresAt}),
			ExpiresAt:             quote.ExpiresAt,
			CreatedAt:             now,
			UpdatedAt:             now,
		}

		session.PaymentCode, err = u.jweService.Encrypt(services.JWEPayload{
			Version:    "partner_session.v1",
			SessionID:  session.ID.String(),
			PaymentID:  session.ID.String(),
			Amount:     session.PaymentAmount,
			MerchantID: session.MerchantID.String(),
			Currency:   quote.InvoiceCurrency,
			DestChain:  session.DestChain,
			DestToken:  session.DestToken,
			DestWallet: session.DestWallet,
			Nonce:      utils.GenerateUUIDv7().String(),
			ExpiresAt:  session.ExpiresAt.Unix(),
		})
		if err != nil {
			return domainerrors.InternalServerError(fmt.Sprintf("failed to generate payment code: %v", err))
		}

		if err := u.sessionRepo.Create(txCtx, session); err != nil {
			return domainerrors.InternalServerError(fmt.Sprintf("failed to create partner payment session: %v", err))
		}
		if err := u.quoteRepo.MarkUsed(txCtx, quote.ID); err != nil {
			return domainerrors.InternalServerError(fmt.Sprintf("failed to mark quote used: %v", err))
		}

		output = &CreatePartnerPaymentSessionOutput{
			PaymentID:       session.ID.String(),
			MerchantID:      session.MerchantID.String(),
			InvoiceCurrency: session.InvoiceCurrency,
			InvoiceAmount:   session.InvoiceAmount,
			Amount:          session.PaymentAmount,
			AmountDecimals:  session.PaymentAmountDecimals,
			DestChain:       session.DestChain,
			DestToken:       session.DestToken,
			DestWallet:      session.DestWallet,
			ExpireTime:      session.ExpiresAt,
			PaymentURL:      session.PaymentURL,
			PaymentCode:     session.PaymentCode,
			Status:          string(session.Status),
		}
		output.PaymentInstruction.ChainID = session.SelectedChainID
		output.PaymentInstruction.To = txData.To
		output.PaymentInstruction.Value = "0"
		output.PaymentInstruction.Data = txData.Hex
		output.PaymentInstruction.ProgramID = txData.ProgramID
		output.PaymentInstruction.DataBase58 = txData.Base58
		output.PaymentInstruction.DataBase64 = txData.Base64
		output.Quote.QuoteID = quote.ID.String()
		output.Quote.PriceSource = quote.PriceSource
		output.Quote.QuoteRate = quote.QuoteRate
		output.Quote.QuoteExpiresAt = quote.ExpiresAt

		return nil
	})
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (u *PartnerPaymentSessionUsecase) GetSession(ctx context.Context, sessionID uuid.UUID) (*GetPartnerPaymentSessionOutput, error) {
	if sessionID == uuid.Nil {
		return nil, domainerrors.BadRequest("payment session id is required")
	}
	session, err := u.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, domainerrors.NotFound("payment session not found")
	}
	return buildPartnerPaymentSessionReadModel(session), nil
}

func (u *PartnerPaymentSessionUsecase) ResolvePaymentCode(ctx context.Context, input *ResolvePartnerPaymentCodeInput) (*ResolvePartnerPaymentCodeOutput, error) {
	if input == nil || strings.TrimSpace(input.PaymentCode) == "" {
		return nil, domainerrors.BadRequest("payment_code is required")
	}
	payload, err := u.jweService.Decrypt(strings.TrimSpace(input.PaymentCode))
	if err != nil {
		return nil, domainerrors.Unauthorized("invalid or tampered payment code")
	}
	if payload.ExpiresAt > 0 && time.Now().Unix() > payload.ExpiresAt {
		return nil, domainerrors.NewAppError(410, domainerrors.CodeBadRequest, "payment invitation has expired", nil)
	}
	if payload.Version != "" && payload.Version != "partner_session.v1" {
		return nil, domainerrors.BadRequest("unsupported payment code version")
	}

	sessionIDRaw := strings.TrimSpace(payload.SessionID)
	if sessionIDRaw == "" {
		sessionIDRaw = strings.TrimSpace(payload.PaymentID)
	}
	sessionID, err := uuid.Parse(sessionIDRaw)
	if err != nil {
		return nil, domainerrors.BadRequest("invalid session id in payment code")
	}

	session, err := u.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, domainerrors.NotFound("payment session not found")
	}
	if session.Status != domainentities.PartnerPaymentSessionStatusPending {
		return nil, domainerrors.BadRequest("payment session is not payable")
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		return nil, domainerrors.NewAppError(410, domainerrors.CodeBadRequest, "payment invitation has expired", nil)
	}

	out := &ResolvePartnerPaymentCodeOutput{
		PaymentID:      session.ID.String(),
		MerchantID:     session.MerchantID.String(),
		Amount:         session.PaymentAmount,
		AmountDecimals: session.PaymentAmountDecimals,
		DestChain:      session.DestChain,
		DestToken:      session.DestToken,
		DestWallet:     session.DestWallet,
		ExpiresAt:      session.ExpiresAt,
	}
	out.PaymentInstruction.ChainID = session.SelectedChainID
	out.PaymentInstruction.To = instructionToAddress(session)
	out.PaymentInstruction.Value = coalesceString(session.InstructionValue, "0")
	out.PaymentInstruction.Data = session.InstructionDataHex
	out.PaymentInstruction.ProgramID = instructionProgramID(session)
	out.PaymentInstruction.DataBase58 = session.InstructionDataBase58
	out.PaymentInstruction.DataBase64 = session.InstructionDataBase64
	return out, nil
}

func buildPartnerPaymentSessionReadModel(session *domainentities.PartnerPaymentSession) *GetPartnerPaymentSessionOutput {
	out := &GetPartnerPaymentSessionOutput{
		PaymentID:      session.ID.String(),
		Status:         string(session.Status),
		Amount:         session.PaymentAmount,
		AmountDecimals: session.PaymentAmountDecimals,
		DestChain:      session.DestChain,
		DestToken:      session.DestToken,
		DestWallet:     session.DestWallet,
		ExpiresAt:      session.ExpiresAt,
		PaymentURL:     session.PaymentURL,
		PaymentCode:    session.PaymentCode,
	}
	out.PaymentInstruction.ChainID = session.SelectedChainID
	out.PaymentInstruction.To = instructionToAddress(session)
	out.PaymentInstruction.Value = coalesceString(session.InstructionValue, "0")
	out.PaymentInstruction.Data = session.InstructionDataHex
	out.PaymentInstruction.ProgramID = instructionProgramID(session)
	out.PaymentInstruction.DataBase58 = session.InstructionDataBase58
	out.PaymentInstruction.DataBase64 = session.InstructionDataBase64
	return out
}

func instructionToAddress(session *domainentities.PartnerPaymentSession) string {
	if strings.TrimSpace(session.InstructionDataHex) != "" {
		return session.InstructionTo
	}
	return ""
}

func instructionProgramID(session *domainentities.PartnerPaymentSession) string {
	if strings.TrimSpace(session.InstructionDataHex) == "" {
		return session.InstructionTo
	}
	return ""
}

func coalesceString(v string, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func stringPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	out := v
	return &out
}

func mustJSON(v interface{}) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
