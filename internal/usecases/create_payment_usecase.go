package usecases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	domainrepos "payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/pkg/utils"
)

type CreatePaymentPricingType string

const (
	CreatePaymentPricingTypeInvoiceCurrency   CreatePaymentPricingType = "invoice_currency"
	CreatePaymentPricingTypePaymentTokenFixed CreatePaymentPricingType = "payment_token_fixed"
	CreatePaymentPricingTypePaymentTokenDyn   CreatePaymentPricingType = "payment_token_dynamic"
)

type CreatePaymentInput struct {
	MerchantContextID uuid.UUID
	MerchantID        uuid.UUID
	ChainID           string
	SelectedToken     string
	PricingType       string
	RequestedAmount   string
}

type CreatePaymentOutput struct {
	PaymentID                string    `json:"payment_id"`
	MerchantID               string    `json:"merchant_id"`
	Amount                   string    `json:"amount"`
	InvoiceCurrency          string    `json:"invoice_currency"`
	InvoiceAmount            string    `json:"invoice_amount"`
	PayerSelectedChain       string    `json:"payer_selected_chain"`
	PayerSelectedToken       string    `json:"payer_selected_token"`
	PayerSelectedTokenSymbol string    `json:"payer_selected_token_symbol"`
	QuotedTokenSymbol        string    `json:"quoted_token_symbol"`
	QuotedTokenAmount        string    `json:"quoted_token_amount"`
	QuotedTokenAmountAtomic  string    `json:"quoted_token_amount_atomic"`
	QuotedTokenDecimals      int       `json:"quoted_token_decimals"`
	QuoteRate                string    `json:"quote_rate"`
	QuoteSource              string    `json:"quote_source"`
	QuoteExpiresAt           time.Time `json:"quote_expires_at"`
	DestChain                string    `json:"dest_chain"`
	DestToken                string    `json:"dest_token"`
	DestWallet               string    `json:"dest_wallet"`
	SettlementDestChain      string    `json:"settlement_dest_chain"`
	SettlementDestToken      string    `json:"settlement_dest_token"`
	SettlementDestWallet     string    `json:"settlement_dest_wallet"`
	ExpireTime               time.Time `json:"expire_time"`
	PaymentURL               string    `json:"payment_url"`
	PaymentCode              string    `json:"payment_code"`
	PaymentInstruction       struct {
		ChainID    string `json:"chain_id"`
		To         string `json:"to,omitempty"`
		Value      string `json:"value,omitempty"`
		Data       string `json:"data,omitempty"`
		ProgramID  string `json:"program_id,omitempty"`
		DataBase58 string `json:"data_base58,omitempty"`
		DataBase64 string `json:"data_base64,omitempty"`
	} `json:"payment_instruction"`
}

type createPaymentQuoteEngine interface {
	CreateQuote(ctx context.Context, input *CreatePartnerQuoteInput) (*CreatePartnerQuoteOutput, error)
}

type createPaymentSessionEngine interface {
	CreateSession(ctx context.Context, input *CreatePartnerPaymentSessionInput) (*CreatePartnerPaymentSessionOutput, error)
}

type CreatePaymentUsecase struct {
	merchantRepo   domainrepos.MerchantRepository
	settlementRepo domainrepos.MerchantSettlementProfileRepository
	walletRepo     domainrepos.WalletRepository
	tokenRepo      domainrepos.TokenRepository
	chainRepo      domainrepos.ChainRepository
	quoteRepo      domainrepos.PaymentQuoteRepository
	quoteUC        createPaymentQuoteEngine
	sessionUC      createPaymentSessionEngine
	chainResolver  *ChainResolver
}

type merchantCreatePaymentConfig struct {
	DestChain         string
	DestToken         string
	DestWallet        string
	BridgeTokenSymbol string
}

type resolvedMerchantSettlementConfig struct {
	InvoiceToken      *entities.Token
	DestChainID       uuid.UUID
	DestChainCAIP2    string
	DestToken         *entities.Token
	DestWallet        string
	BridgeTokenSymbol string
	DestBridgeToken   *entities.Token
}

func NewCreatePaymentUsecase(
	merchantRepo domainrepos.MerchantRepository,
	settlementRepo domainrepos.MerchantSettlementProfileRepository,
	walletRepo domainrepos.WalletRepository,
	tokenRepo domainrepos.TokenRepository,
	chainRepo domainrepos.ChainRepository,
	quoteRepo domainrepos.PaymentQuoteRepository,
	quoteUC createPaymentQuoteEngine,
	sessionUC createPaymentSessionEngine,
) *CreatePaymentUsecase {
	return &CreatePaymentUsecase{
		merchantRepo:   merchantRepo,
		settlementRepo: settlementRepo,
		walletRepo:     walletRepo,
		tokenRepo:      tokenRepo,
		chainRepo:      chainRepo,
		quoteRepo:      quoteRepo,
		quoteUC:        quoteUC,
		sessionUC:      sessionUC,
		chainResolver:  NewChainResolver(chainRepo),
	}
}

func (u *CreatePaymentUsecase) CreatePayment(ctx context.Context, input *CreatePaymentInput) (*CreatePaymentOutput, error) {
	if input == nil {
		return nil, domainerrors.BadRequest("input is required")
	}
	merchantID, err := u.resolveMerchantID(input)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.ChainID) == "" {
		return nil, domainerrors.BadRequest("chain_id is required")
	}
	if strings.TrimSpace(input.SelectedToken) == "" {
		return nil, domainerrors.BadRequest("selected_token is required")
	}
	if strings.TrimSpace(input.RequestedAmount) == "" {
		return nil, domainerrors.BadRequest("requested_amount is required")
	}
	pricingType := CreatePaymentPricingType(strings.TrimSpace(input.PricingType))
	if pricingType == "" {
		return nil, domainerrors.BadRequest("pricing_type is required")
	}
	if pricingType != CreatePaymentPricingTypeInvoiceCurrency && pricingType != CreatePaymentPricingTypePaymentTokenFixed && pricingType != CreatePaymentPricingTypePaymentTokenDyn {
		return nil, domainerrors.BadRequest("unsupported pricing_type")
	}
	if u.quoteUC == nil || u.sessionUC == nil || u.quoteRepo == nil || u.merchantRepo == nil || u.walletRepo == nil || u.tokenRepo == nil || u.chainRepo == nil || u.settlementRepo == nil {
		return nil, domainerrors.InternalServerError("create payment orchestrator is not configured")
	}

	merchant, err := u.merchantRepo.GetByID(ctx, merchantID)
	if err != nil || merchant == nil {
		return nil, domainerrors.NotFound("merchant not found")
	}
	if merchant.Status != entities.MerchantStatusActive {
		return nil, domainerrors.BadRequest("merchant account is not active")
	}

	config := u.resolveMerchantCreatePaymentConfig(ctx, merchant)
	settlement, err := u.resolveMerchantSettlementConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	chainUUID, chainCAIP2, err := u.chainResolver.ResolveFromAny(ctx, input.ChainID)
	if err != nil {
		return nil, domainerrors.BadRequest(fmt.Sprintf("invalid chain_id: %v", err))
	}
	selectedToken, err := u.tokenRepo.GetByAddress(ctx, strings.TrimSpace(input.SelectedToken), chainUUID)
	if err != nil || selectedToken == nil || !selectedToken.IsActive {
		return nil, domainerrors.BadRequest("selected_token not supported on chain_id")
	}
	walletAddress, err := u.resolveMerchantWallet(ctx, merchant, config)
	if err != nil {
		return nil, err
	}

	var quoteOut *CreatePartnerQuoteOutput
	switch pricingType {
	case CreatePaymentPricingTypeInvoiceCurrency:
		quoteOut, err = u.createInvoiceCurrencyQuote(ctx, merchantID, chainCAIP2, selectedToken, settlement, strings.TrimSpace(input.RequestedAmount), walletAddress)
		if err != nil {
			return nil, err
		}
	default:
		quoteOut, err = u.createSyntheticSelectedTokenQuote(ctx, merchantID, chainCAIP2, selectedToken, pricingType, strings.TrimSpace(input.RequestedAmount), settlement)
		if err != nil {
			return nil, err
		}
	}

	quoteID, err := uuid.Parse(quoteOut.QuoteID)
	if err != nil {
		return nil, domainerrors.InternalServerError("invalid quote id generated")
	}
	sessionOut, err := u.sessionUC.CreateSession(ctx, &CreatePartnerPaymentSessionInput{
		MerchantID:        merchantID,
		QuoteID:           quoteID,
		DestWallet:        walletAddress,
		DestChainOverride: settlement.DestChainCAIP2,
		DestTokenOverride: settlement.DestToken.ContractAddress,
	})
	if err != nil {
		return nil, err
	}

	out := &CreatePaymentOutput{
		PaymentID:                sessionOut.PaymentID,
		MerchantID:               sessionOut.MerchantID,
		Amount:                   sessionOut.Amount,
		InvoiceCurrency:          settlement.InvoiceToken.Symbol,
		InvoiceAmount:            strings.TrimSpace(input.RequestedAmount),
		PayerSelectedChain:       chainCAIP2,
		PayerSelectedToken:       selectedToken.ContractAddress,
		PayerSelectedTokenSymbol: selectedToken.Symbol,
		QuotedTokenSymbol:        quoteOut.SelectedTokenSymbol,
		QuotedTokenAmount:        smallestUnitToDecimalString(quoteOut.QuotedAmount, quoteOut.QuoteDecimals),
		QuotedTokenAmountAtomic:  quoteOut.QuotedAmount,
		QuotedTokenDecimals:      quoteOut.QuoteDecimals,
		QuoteRate:                quoteOut.QuoteRate,
		QuoteSource:              quoteOut.PriceSource,
		QuoteExpiresAt:           quoteOut.QuoteExpiresAt,
		DestChain:                sessionOut.DestChain,
		DestToken:                sessionOut.DestToken,
		DestWallet:               sessionOut.DestWallet,
		SettlementDestChain:      sessionOut.DestChain,
		SettlementDestToken:      sessionOut.DestToken,
		SettlementDestWallet:     sessionOut.DestWallet,
		ExpireTime:               sessionOut.ExpireTime,
		PaymentURL:               sessionOut.PaymentURL,
		PaymentCode:              sessionOut.PaymentCode,
	}
	out.PaymentInstruction.ChainID = sessionOut.PaymentInstruction.ChainID
	out.PaymentInstruction.To = sessionOut.PaymentInstruction.To
	out.PaymentInstruction.Value = sessionOut.PaymentInstruction.Value
	out.PaymentInstruction.Data = sessionOut.PaymentInstruction.Data
	out.PaymentInstruction.ProgramID = sessionOut.PaymentInstruction.ProgramID
	out.PaymentInstruction.DataBase58 = sessionOut.PaymentInstruction.DataBase58
	out.PaymentInstruction.DataBase64 = sessionOut.PaymentInstruction.DataBase64
	return out, nil
}

func (u *CreatePaymentUsecase) resolveMerchantCreatePaymentConfig(ctx context.Context, merchant *entities.Merchant) merchantCreatePaymentConfig {
	if u.settlementRepo != nil && merchant != nil {
		profile, err := u.settlementRepo.GetByMerchantID(ctx, merchant.ID)
		if err == nil && profile != nil {
			return merchantCreatePaymentConfig{
				DestChain:         strings.TrimSpace(profile.DestChain),
				DestToken:         strings.TrimSpace(profile.DestToken),
				DestWallet:        strings.TrimSpace(profile.DestWallet),
				BridgeTokenSymbol: strings.TrimSpace(profile.BridgeTokenSymbol),
			}
		}
	}
	return merchantCreatePaymentConfig{}
}

func (u *CreatePaymentUsecase) resolveMerchantID(input *CreatePaymentInput) (uuid.UUID, error) {
	if input.MerchantContextID == uuid.Nil {
		return uuid.Nil, domainerrors.Forbidden("merchant context required")
	}
	if input.MerchantID != uuid.Nil && input.MerchantID != input.MerchantContextID {
		return uuid.Nil, domainerrors.Forbidden("merchant_id override is not allowed on this route")
	}
	return input.MerchantContextID, nil
}

func (u *CreatePaymentUsecase) resolveMerchantWallet(ctx context.Context, merchant *entities.Merchant, config merchantCreatePaymentConfig) (string, error) {
	if strings.TrimSpace(config.DestWallet) != "" {
		return strings.TrimSpace(config.DestWallet), nil
	}
	wallets, err := u.walletRepo.GetByUserID(ctx, merchant.UserID)
	if err != nil || len(wallets) == 0 {
		return "", domainerrors.BadRequest("merchant destination wallet is not configured")
	}
	for _, wallet := range wallets {
		if wallet != nil && wallet.IsPrimary && strings.TrimSpace(wallet.Address) != "" {
			return strings.TrimSpace(wallet.Address), nil
		}
	}
	if wallets[0] == nil || strings.TrimSpace(wallets[0].Address) == "" {
		return "", domainerrors.BadRequest("merchant destination wallet is not configured")
	}
	return strings.TrimSpace(wallets[0].Address), nil
}

func (u *CreatePaymentUsecase) createSyntheticSelectedTokenQuote(ctx context.Context, merchantID uuid.UUID, chainCAIP2 string, selectedToken *entities.Token, pricingType CreatePaymentPricingType, requestedAmount string, settlement *resolvedMerchantSettlementConfig) (*CreatePartnerQuoteOutput, error) {
	atomicAmount, err := convertToSmallestUnit(requestedAmount, selectedToken.Decimals)
	if err != nil {
		return nil, domainerrors.BadRequest(err.Error())
	}
	now := time.Now().UTC()
	expiresAt := now.Add(partnerQuoteTTL)
	invoiceCurrency := settlement.InvoiceToken.Symbol
	priceSource := "merchant-fixed-selected-token"
	if pricingType == CreatePaymentPricingTypePaymentTokenDyn {
		priceSource = "customer-input-selected-token"
	}
	if chainCAIP2 != settlement.DestChainCAIP2 {
		if strings.EqualFold(selectedToken.Symbol, settlement.BridgeTokenSymbol) {
			priceSource = fmt.Sprintf("cross-chain-bridge-token-direct-via-%s", strings.ToLower(settlement.BridgeTokenSymbol))
		} else {
			priceSource = fmt.Sprintf("cross-chain-normalized-via-%s", strings.ToLower(settlement.BridgeTokenSymbol))
		}
	}
	quote := &entities.PaymentQuote{
		ID:                    utils.GenerateUUIDv7(),
		MerchantID:            merchantID,
		InvoiceCurrency:       invoiceCurrency,
		InvoiceAmount:         atomicAmount,
		SelectedChainID:       chainCAIP2,
		SelectedTokenAddress:  selectedToken.ContractAddress,
		SelectedTokenSymbol:   selectedToken.Symbol,
		SelectedTokenDecimals: selectedToken.Decimals,
		QuotedAmount:          atomicAmount,
		QuoteRate:             fmt.Sprintf("1 %s = 1 %s", selectedToken.Symbol, selectedToken.Symbol),
		PriceSource:           priceSource,
		Route:                 selectedToken.Symbol,
		SlippageBps:           0,
		RateTimestamp:         now,
		ExpiresAt:             expiresAt,
		Status:                entities.PaymentQuoteStatusActive,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := u.quoteRepo.Create(ctx, quote); err != nil {
		return nil, domainerrors.InternalServerError(fmt.Sprintf("failed to persist synthetic quote: %v", err))
	}
	return &CreatePartnerQuoteOutput{
		QuoteID:             quote.ID.String(),
		InvoiceCurrency:     quote.InvoiceCurrency,
		InvoiceAmount:       quote.InvoiceAmount,
		SelectedChain:       quote.SelectedChainID,
		SelectedToken:       quote.SelectedTokenAddress,
		SelectedTokenSymbol: quote.SelectedTokenSymbol,
		QuotedAmount:        quote.QuotedAmount,
		QuoteDecimals:       quote.SelectedTokenDecimals,
		QuoteRate:           quote.QuoteRate,
		PriceSource:         quote.PriceSource,
		Route:               quote.Route,
		SlippageBps:         quote.SlippageBps,
		RateTimestamp:       quote.RateTimestamp,
		QuoteExpiresAt:      quote.ExpiresAt,
	}, nil
}

func (u *CreatePaymentUsecase) createInvoiceCurrencyQuote(ctx context.Context, merchantID uuid.UUID, selectedChainCAIP2 string, selectedToken *entities.Token, settlement *resolvedMerchantSettlementConfig, requestedAmount string, destWallet string) (*CreatePartnerQuoteOutput, error) {
	invoiceAtomic, err := convertToSmallestUnit(requestedAmount, settlement.InvoiceToken.Decimals)
	if err != nil {
		return nil, domainerrors.BadRequest(err.Error())
	}
	if selectedChainCAIP2 == settlement.DestChainCAIP2 {
		return u.quoteUC.CreateQuote(ctx, &CreatePartnerQuoteInput{
			MerchantID:      merchantID,
			InvoiceCurrency: settlement.InvoiceToken.Symbol,
			InvoiceAmount:   invoiceAtomic,
			SelectedChain:   selectedChainCAIP2,
			SelectedToken:   selectedToken.ContractAddress,
			DestWallet:      destWallet,
		})
	}

	destBridgeAmount, routeSummary, _, err := u.resolveCrossChainBridgeAmount(ctx, merchantID, settlement, invoiceAtomic, destWallet)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(selectedToken.Symbol, settlement.BridgeTokenSymbol) {
		return u.createCompositeQuote(ctx, merchantID, selectedChainCAIP2, selectedToken, settlement.InvoiceToken.Symbol, invoiceAtomic, destBridgeAmount, fmt.Sprintf("cross-chain-bridge-token-direct-via-%s", strings.ToLower(settlement.BridgeTokenSymbol)), routeSummary)
	}

	sourceBridgeToken, err := u.tokenRepo.GetBySymbol(ctx, settlement.BridgeTokenSymbol, selectedToken.ChainUUID)
	if err != nil || sourceBridgeToken == nil || !sourceBridgeToken.IsActive {
		return nil, domainerrors.BadRequest("bridge token is not supported on selected source chain")
	}
	sourceLegQuote, err := u.quoteUC.CreateQuote(ctx, &CreatePartnerQuoteInput{
		MerchantID:      merchantID,
		InvoiceCurrency: sourceBridgeToken.Symbol,
		InvoiceAmount:   destBridgeAmount,
		SelectedChain:   selectedChainCAIP2,
		SelectedToken:   selectedToken.ContractAddress,
		DestWallet:      destWallet,
	})
	if err != nil {
		return nil, err
	}
	sourceLegQuoteID, parseErr := uuid.Parse(sourceLegQuote.QuoteID)
	if parseErr == nil {
		_ = u.quoteRepo.UpdateStatus(ctx, sourceLegQuoteID, entities.PaymentQuoteStatusCancelled)
	}

	return u.createCompositeQuote(
		ctx,
		merchantID,
		selectedChainCAIP2,
		selectedToken,
		settlement.InvoiceToken.Symbol,
		invoiceAtomic,
		sourceLegQuote.QuotedAmount,
		fmt.Sprintf("cross-chain-normalized-via-%s", strings.ToLower(settlement.BridgeTokenSymbol)),
		fmt.Sprintf("%s | %s", routeSummary, sourceLegQuote.Route),
	)
}

func (u *CreatePaymentUsecase) resolveCrossChainBridgeAmount(ctx context.Context, merchantID uuid.UUID, settlement *resolvedMerchantSettlementConfig, invoiceAtomic string, destWallet string) (string, string, string, error) {
	if strings.EqualFold(settlement.InvoiceToken.ContractAddress, settlement.DestBridgeToken.ContractAddress) {
		return invoiceAtomic, fmt.Sprintf("%s->%s", settlement.InvoiceToken.Symbol, settlement.DestBridgeToken.Symbol), fmt.Sprintf("cross-chain-bridge-token-direct-via-%s", strings.ToLower(settlement.BridgeTokenSymbol)), nil
	}
	destQuote, err := u.quoteUC.CreateQuote(ctx, &CreatePartnerQuoteInput{
		MerchantID:      merchantID,
		InvoiceCurrency: settlement.InvoiceToken.Symbol,
		InvoiceAmount:   invoiceAtomic,
		SelectedChain:   settlement.DestChainCAIP2,
		SelectedToken:   settlement.DestBridgeToken.ContractAddress,
		DestWallet:      destWallet,
	})
	if err != nil {
		return "", "", "", err
	}
	destQuoteID, parseErr := uuid.Parse(destQuote.QuoteID)
	if parseErr == nil {
		_ = u.quoteRepo.UpdateStatus(ctx, destQuoteID, entities.PaymentQuoteStatusCancelled)
	}
	return destQuote.QuotedAmount, destQuote.Route, destQuote.PriceSource, nil
}

func (u *CreatePaymentUsecase) createCompositeQuote(ctx context.Context, merchantID uuid.UUID, selectedChainCAIP2 string, selectedToken *entities.Token, invoiceCurrency string, invoiceAtomic string, quotedAmount string, priceSource string, route string) (*CreatePartnerQuoteOutput, error) {
	now := time.Now().UTC()
	quote := &entities.PaymentQuote{
		ID:                    utils.GenerateUUIDv7(),
		MerchantID:            merchantID,
		InvoiceCurrency:       invoiceCurrency,
		InvoiceAmount:         invoiceAtomic,
		SelectedChainID:       selectedChainCAIP2,
		SelectedTokenAddress:  selectedToken.ContractAddress,
		SelectedTokenSymbol:   selectedToken.Symbol,
		SelectedTokenDecimals: selectedToken.Decimals,
		QuotedAmount:          quotedAmount,
		QuoteRate:             fmt.Sprintf("%s normalized to %s", invoiceCurrency, selectedToken.Symbol),
		PriceSource:           priceSource,
		Route:                 route,
		SlippageBps:           partnerQuoteSlippage,
		RateTimestamp:         now,
		ExpiresAt:             now.Add(partnerQuoteTTL),
		Status:                entities.PaymentQuoteStatusActive,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := u.quoteRepo.Create(ctx, quote); err != nil {
		return nil, domainerrors.InternalServerError(fmt.Sprintf("failed to persist composite quote: %v", err))
	}
	return &CreatePartnerQuoteOutput{
		QuoteID:             quote.ID.String(),
		InvoiceCurrency:     quote.InvoiceCurrency,
		InvoiceAmount:       quote.InvoiceAmount,
		SelectedChain:       quote.SelectedChainID,
		SelectedToken:       quote.SelectedTokenAddress,
		SelectedTokenSymbol: quote.SelectedTokenSymbol,
		QuotedAmount:        quote.QuotedAmount,
		QuoteDecimals:       quote.SelectedTokenDecimals,
		QuoteRate:           quote.QuoteRate,
		PriceSource:         quote.PriceSource,
		Route:               quote.Route,
		SlippageBps:         quote.SlippageBps,
		RateTimestamp:       quote.RateTimestamp,
		QuoteExpiresAt:      quote.ExpiresAt,
	}, nil
}

func (u *CreatePaymentUsecase) resolveMerchantSettlementConfig(ctx context.Context, config merchantCreatePaymentConfig) (*resolvedMerchantSettlementConfig, error) {
	if strings.TrimSpace(config.DestChain) == "" {
		return nil, domainerrors.BadRequest("merchant dest_chain is not configured")
	}
	if strings.TrimSpace(config.DestToken) == "" {
		return nil, domainerrors.BadRequest("merchant dest_token is not configured")
	}
	destChainID, destChainCAIP2, err := u.chainResolver.ResolveFromAny(ctx, config.DestChain)
	if err != nil {
		return nil, domainerrors.BadRequest(fmt.Sprintf("invalid merchant dest_chain: %v", err))
	}
	destToken, err := u.tokenRepo.GetByAddress(ctx, config.DestToken, destChainID)
	if err != nil || destToken == nil || !destToken.IsActive {
		return nil, domainerrors.BadRequest("merchant dest_token is not supported")
	}
	invoiceToken := destToken
	bridgeSymbol := strings.TrimSpace(config.BridgeTokenSymbol)
	if bridgeSymbol == "" {
		bridgeSymbol = "USDC"
	}
	destBridgeToken, err := u.tokenRepo.GetBySymbol(ctx, bridgeSymbol, destChainID)
	if err != nil || destBridgeToken == nil || !destBridgeToken.IsActive {
		return nil, domainerrors.BadRequest("merchant bridge token is not supported on dest_chain")
	}
	return &resolvedMerchantSettlementConfig{
		InvoiceToken:      invoiceToken,
		DestChainID:       destChainID,
		DestChainCAIP2:    destChainCAIP2,
		DestToken:         destToken,
		DestWallet:        strings.TrimSpace(config.DestWallet),
		BridgeTokenSymbol: bridgeSymbol,
		DestBridgeToken:   destBridgeToken,
	}, nil
}

func smallestUnitToDecimalString(amount string, decimals int) string {
	raw := strings.TrimSpace(amount)
	if raw == "" {
		return "0"
	}
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return raw
		}
	}
	if decimals <= 0 {
		return strings.TrimLeft(raw, "0")
	}
	if len(raw) <= decimals {
		raw = strings.Repeat("0", decimals-len(raw)+1) + raw
	}
	split := len(raw) - decimals
	whole := strings.TrimLeft(raw[:split], "0")
	if whole == "" {
		whole = "0"
	}
	frac := strings.TrimRight(raw[split:], "0")
	if frac == "" {
		return whole
	}
	return whole + "." + frac
}
