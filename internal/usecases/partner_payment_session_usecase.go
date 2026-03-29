package usecases

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	domainentities "payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	domainrepos "payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/domain/services"
	"payment-kita.backend/pkg/utils"
)

const defaultFrontendBaseURL = "https://payment-kita.excitech.id"

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
		ChainID     string `json:"chain_id"`
		To          string `json:"to,omitempty"`
		Value       string `json:"value,omitempty"`
		Data        string `json:"data,omitempty"`
		ProgramID   string `json:"program_id,omitempty"`
		DataBase58  string `json:"data_base58,omitempty"`
		DataBase64  string `json:"data_base64,omitempty"`
		ApprovalTo  string `json:"approval_to,omitempty"`
		ApprovalHex string `json:"approval_hex_data,omitempty"`
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
		ChainID     string `json:"chain_id"`
		To          string `json:"to,omitempty"`
		Value       string `json:"value,omitempty"`
		Data        string `json:"data,omitempty"`
		ProgramID   string `json:"program_id,omitempty"`
		DataBase58  string `json:"data_base58,omitempty"`
		DataBase64  string `json:"data_base64,omitempty"`
		ApprovalTo  string `json:"approval_to,omitempty"`
		ApprovalHex string `json:"approval_hex_data,omitempty"`
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
		ChainID     string `json:"chain_id"`
		To          string `json:"to,omitempty"`
		Value       string `json:"value,omitempty"`
		Data        string `json:"data,omitempty"`
		ProgramID   string `json:"program_id,omitempty"`
		DataBase58  string `json:"data_base58,omitempty"`
		DataBase64  string `json:"data_base64,omitempty"`
		ApprovalTo  string `json:"approval_to,omitempty"`
		ApprovalHex string `json:"approval_hex_data,omitempty"`
	} `json:"payment_instruction"`
}

type PartnerPaymentSessionUsecase struct {
	quoteRepo           domainrepos.PaymentQuoteRepository
	sessionRepo         domainrepos.PartnerPaymentSessionRepository
	paymentRequestRepo  domainrepos.PaymentRequestRepository
	contractRepo        domainrepos.SmartContractRepository
	tokenRepo           domainrepos.TokenRepository
	chainRepo           domainrepos.ChainRepository
	merchantRepo        domainrepos.MerchantRepository
	uow                 domainrepos.UnitOfWork
	jweService          services.JWEService
	paymentRequestLogic *PaymentRequestUsecase
	paymentUC           *PaymentUsecase
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
	merchantRepo domainrepos.MerchantRepository,
	uow domainrepos.UnitOfWork,
	jweService services.JWEService,
	paymentRequestLogic *PaymentRequestUsecase,
	paymentUC *PaymentUsecase,
	checkoutBaseURL string,
) *PartnerPaymentSessionUsecase {
	baseURL := resolvePartnerPayBaseURL(checkoutBaseURL)

	return &PartnerPaymentSessionUsecase{
		quoteRepo:           quoteRepo,
		sessionRepo:         sessionRepo,
		paymentRequestRepo:  paymentRequestRepo,
		contractRepo:        contractRepo,
		tokenRepo:           tokenRepo,
		chainRepo:           chainRepo,
		merchantRepo:        merchantRepo,
		uow:                 uow,
		jweService:          jweService,
		paymentRequestLogic: paymentRequestLogic,
		paymentUC:           paymentUC,
		chainResolver:       NewChainResolver(chainRepo),
		checkoutBaseURL:     strings.TrimRight(baseURL, "/"),
	}
}

func resolvePartnerPayBaseURL(explicitBaseURL string) string {
	baseURL := strings.TrimSpace(explicitBaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("PARTNER_CHECKOUT_BASE_URL"))
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("FRONTEND_BASE_URL"))
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("NEXT_PUBLIC_APP_URL"))
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("NEXT_PUBLIC_FRONTEND_URL"))
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("NEXT_PUBLIC_SITE_URL"))
	}
	if baseURL == "" {
		baseURL = defaultFrontendBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	lower := strings.ToLower(baseURL)
	switch {
	case strings.HasSuffix(lower, "/pay"):
		return baseURL
	case strings.HasSuffix(lower, "/checkout"):
		return strings.TrimSuffix(baseURL, "/checkout") + "/pay"
	default:
		return baseURL + "/pay"
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
		if !isUnlimitedExpiryTime(quote.ExpiresAt) && now.After(quote.ExpiresAt) {
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

		destChainID, destChainCAIP2, err := u.chainResolver.ResolveFromAny(txCtx, coalesceString(strings.TrimSpace(input.DestChainOverride), quote.SelectedChainID))
		if err != nil {
			return domainerrors.BadRequest(fmt.Sprintf("invalid destination chain: %v", err))
		}
		destTokenAddress := coalesceString(strings.TrimSpace(input.DestTokenOverride), quote.SelectedTokenAddress)

		// V2 Logic: Calculate native bridge fee if necessary
		bridgeFeeNative := big.NewInt(0)
		if u.paymentUC != nil && contract != nil {
			tempPayment := &domainentities.Payment{
				SourceChainID:      selectedChainID,
				DestChainID:        destChainID,
				SourceTokenAddress: quote.SelectedTokenAddress,
				DestTokenAddress:   destTokenAddress,
				SourceAmount:       quote.QuotedAmount,
				ReceiverAddress:    strings.TrimSpace(input.DestWallet),
			}
			onchainCost, err := u.paymentUC.quoteGatewayPaymentCost(txCtx, tempPayment, contract.ContractAddress, nil)
			if err == nil && onchainCost != nil {
				if f, ok := new(big.Int).SetString(onchainCost.BridgeFeeNative, 10); ok {
					bridgeFeeNative = f
				}
			}
		}

		// V2 Logic: Build transaction data
		var txData *domainentities.PaymentRequestTxData
		amountInSource := new(big.Int)
		amountInSource.SetString(paymentRequest.Amount, 10)

		if contract != nil {
			addrType, _ := abi.NewType("address", "", nil)
			receiverPacked, _ := abi.Arguments{{Type: addrType}}.Pack(common.HexToAddress(normalizeEvmAddress(paymentRequest.WalletAddress)))

			v2Args := PaymentRequestV2Args{
				DestChainIDBytes:   []byte(destChainCAIP2),
				ReceiverBytes:      receiverPacked,
				SourceToken:        common.HexToAddress(normalizeEvmAddress(quote.SelectedTokenAddress)),
				BridgeTokenSource:  common.Address{}, // Default
				DestToken:          common.HexToAddress(normalizeEvmAddress(destTokenAddress)),
				AmountInSource:     amountInSource,
				MinBridgeAmountOut: big.NewInt(0),
				MinDestAmountOut:   big.NewInt(0),
				Mode:               0, // Standard
				BridgeOption:       1, // Default Bridge
			}

			dataHex, err := packCreatePaymentDefaultBridgeV2Calldata(v2Args)
			if err == nil {
				dataBytes, _ := hex.DecodeString(strings.TrimPrefix(dataHex, "0x"))
				txData = &domainentities.PaymentRequestTxData{
					To:     contract.ContractAddress,
					Hex:    dataHex,
					Base64: base64.StdEncoding.EncodeToString(dataBytes),
					Base58: base58Encode(dataBytes),
				}
			}
		}

		// Fallback to legacy if V2 fails or contract missing
		if txData == nil {
			txData = u.paymentRequestLogic.buildTransactionData(paymentRequest, contract)
		}

		if txData == nil {
			return domainerrors.InternalServerError("failed to build payment instruction")
		}

		sessionID := utils.GenerateUUIDv7()
		session := &domainentities.PartnerPaymentSession{
			ID:                    sessionID,
			MerchantID:            input.MerchantID,
			QuoteID:               &quote.ID,
			PaymentRequestID:      &paymentRequest.ID,
			InvoiceCurrency:       quote.InvoiceCurrency,
			InvoiceAmount:         quote.InvoiceAmount,
			SelectedChainID:       quote.SelectedChainID,
			SelectedTokenAddress:  quote.SelectedTokenAddress,
			SelectedTokenSymbol:   quote.SelectedTokenSymbol,
			SelectedTokenDecimals: quote.SelectedTokenDecimals,
			DestChain:             destChainCAIP2,
			DestToken:             destTokenAddress,
			DestWallet:            strings.TrimSpace(input.DestWallet),
			PaymentAmount:         quote.QuotedAmount,
			PaymentAmountDecimals: quote.SelectedTokenDecimals,
			Status:                domainentities.PartnerPaymentSessionStatusPending,
			PaymentURL:            buildSessionPaymentURL(u.checkoutBaseURL, sessionID),
			InstructionTo:         txData.To,
			InstructionValue:      bridgeFeeNative.String(),
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

		// ERC20 Approval logic for EVM
		if strings.HasPrefix(selectedChainCAIP2, "eip155:") && contract != nil {
			normalizedToken := normalizeEvmAddress(quote.SelectedTokenAddress)
			if normalizedToken != "0x0000000000000000000000000000000000000000" {
				session.InstructionApprovalTo = normalizedToken
				if u.paymentUC != nil {
					// Resolve chain IDs to internal UUIDs for CalculateOnchainApprovalAmount
					sourceChain, _ := u.chainRepo.GetByCAIP2(txCtx, quote.SelectedChainID)
					destChain, _ := u.chainRepo.GetByCAIP2(txCtx, destChainCAIP2)

					// Create a temporary payment object to satisfy CalculateOnchainApprovalAmount
					// TotalCharged should be at least source amount; calculation logic will handle fees.
					tempPayment := &domainentities.Payment{
						SourceTokenAddress: quote.SelectedTokenAddress,
						SourceAmount:       quote.QuotedAmount,
						TotalCharged:       quote.QuotedAmount,
						DestTokenAddress:   destTokenAddress,
						ReceiverAddress:    session.DestWallet,
					}
					if sourceChain != nil {
						tempPayment.SourceChainID = sourceChain.ID
					}
					if destChain != nil {
						tempPayment.DestChainID = destChain.ID
					}

					vaultAddress := contract.ContractAddress
					if sourceChain != nil {
						resolvedVault := u.paymentUC.ResolveVaultAddressForApproval(sourceChain.ID, contract.ContractAddress)
						if resolvedVault != "" {
							vaultAddress = resolvedVault
						}
					}

					approvalAmount, err := u.paymentUC.CalculateOnchainApprovalAmount(tempPayment, contract.ContractAddress)
					if err != nil {
						// Fallback to base amount if on-chain calculation fails
						approvalAmount = quote.QuotedAmount
					}

					session.InstructionApprovalDataHex = u.paymentUC.buildErc20ApproveHex(vaultAddress, approvalAmount)
				}
			}
		}

		expiresAtUnix := session.ExpiresAt.Unix()
		if isUnlimitedExpiryTime(session.ExpiresAt) {
			expiresAtUnix = 0
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
			ExpiresAt:  expiresAtUnix,
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
		if txData != nil {
			output.PaymentInstruction.To = txData.To
			output.PaymentInstruction.Value = "0"
			output.PaymentInstruction.Data = txData.Hex
			output.PaymentInstruction.ProgramID = txData.ProgramID
			output.PaymentInstruction.DataBase58 = txData.Base58
			output.PaymentInstruction.DataBase64 = txData.Base64
		}
		output.PaymentInstruction.ApprovalTo = session.InstructionApprovalTo
		output.PaymentInstruction.ApprovalHex = session.InstructionApprovalDataHex
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
	if !isUnlimitedExpiryTime(session.ExpiresAt) && time.Now().UTC().After(session.ExpiresAt) {
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
	out.PaymentInstruction.ApprovalTo = session.InstructionApprovalTo
	out.PaymentInstruction.ApprovalHex = session.InstructionApprovalDataHex
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
		PaymentURL:     normalizePaymentURLWithSessionID(session.PaymentURL, session.ID),
		PaymentCode:    session.PaymentCode,
	}
	out.PaymentInstruction.ChainID = session.SelectedChainID
	out.PaymentInstruction.To = instructionToAddress(session)
	out.PaymentInstruction.Value = coalesceString(session.InstructionValue, "0")
	out.PaymentInstruction.Data = session.InstructionDataHex
	out.PaymentInstruction.ProgramID = instructionProgramID(session)
	out.PaymentInstruction.DataBase58 = session.InstructionDataBase58
	out.PaymentInstruction.DataBase64 = session.InstructionDataBase64
	out.PaymentInstruction.ApprovalTo = session.InstructionApprovalTo
	out.PaymentInstruction.ApprovalHex = session.InstructionApprovalDataHex
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

func buildSessionPaymentURL(baseURL string, sessionID uuid.UUID) string {
	return strings.TrimRight(baseURL, "/") + "/" + sessionID.String()
}

func normalizePaymentURLWithSessionID(storedURL string, sessionID uuid.UUID) string {
	raw := strings.TrimSpace(storedURL)
	if raw == "" {
		return raw
	}

	trimmed := strings.TrimRight(raw, "/")
	lastSlash := strings.LastIndex(trimmed, "/")
	if lastSlash < 0 {
		return raw
	}

	return trimmed[:lastSlash+1] + sessionID.String()
}

func mustJSON(v interface{}) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
