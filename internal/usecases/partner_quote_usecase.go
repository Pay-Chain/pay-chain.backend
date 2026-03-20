package usecases

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	domainentities "payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/pkg/utils"
)

const (
	partnerQuoteTTL      = 5 * time.Minute
	partnerQuoteSlippage = 100
)

type CreatePartnerQuoteInput struct {
	MerchantID      uuid.UUID
	InvoiceCurrency string
	InvoiceAmount   string
	SelectedChain   string
	SelectedToken   string
	DestWallet      string
}

type CreatePartnerQuoteOutput struct {
	QuoteID             string    `json:"quote_id"`
	InvoiceCurrency     string    `json:"invoice_currency"`
	InvoiceAmount       string    `json:"invoice_amount"`
	SelectedChain       string    `json:"selected_chain"`
	SelectedToken       string    `json:"selected_token"`
	SelectedTokenSymbol string    `json:"selected_token_symbol"`
	QuotedAmount        string    `json:"quoted_amount"`
	QuoteDecimals       int       `json:"quote_decimals"`
	QuoteRate           string    `json:"quote_rate"`
	PriceSource         string    `json:"price_source"`
	Route               string    `json:"route"`
	SlippageBps         int       `json:"slippage_bps"`
	RateTimestamp       time.Time `json:"rate_timestamp"`
	QuoteExpiresAt      time.Time `json:"quote_expires_at"`
}

type PartnerQuoteUsecase struct {
	quoteRepo       repositories.PaymentQuoteRepository
	tokenRepo       repositories.TokenRepository
	chainRepo       repositories.ChainRepository
	chainResolver   *ChainResolver
	routeSupportFn  func(context.Context, uuid.UUID, string, string) (*TokenRouteSupportStatus, error)
	swapQuoteFn     func(context.Context, uuid.UUID, string, string, *big.Int) (*big.Int, error)
	accurateQuoteFn func(context.Context, uuid.UUID, string, string, *big.Int) (*AccurateSwapQuoteResult, error)
}

func NewPartnerQuoteUsecase(
	quoteRepo repositories.PaymentQuoteRepository,
	tokenRepo repositories.TokenRepository,
	chainRepo repositories.ChainRepository,
	paymentUsecase *PaymentUsecase,
) *PartnerQuoteUsecase {
	uc := &PartnerQuoteUsecase{
		quoteRepo:     quoteRepo,
		tokenRepo:     tokenRepo,
		chainRepo:     chainRepo,
		chainResolver: NewChainResolver(chainRepo),
	}
	if paymentUsecase != nil {
		uc.routeSupportFn = paymentUsecase.CheckRouteSupportDetailed
		uc.swapQuoteFn = paymentUsecase.getSwapQuote
		uc.accurateQuoteFn = paymentUsecase.getAccuratePartnerQuote
	}
	return uc
}

func (u *PartnerQuoteUsecase) RouteSupportFnForTest(fn func(context.Context, uuid.UUID, string, string) (*TokenRouteSupportStatus, error)) {
	u.routeSupportFn = fn
}

func (u *PartnerQuoteUsecase) SwapQuoteFnForTest(fn func(context.Context, uuid.UUID, string, string, *big.Int) (*big.Int, error)) {
	u.swapQuoteFn = fn
}

func (u *PartnerQuoteUsecase) CreateQuote(ctx context.Context, input *CreatePartnerQuoteInput) (*CreatePartnerQuoteOutput, error) {
	if input == nil {
		return nil, domainerrors.BadRequest("input is required")
	}
	if input.MerchantID == uuid.Nil {
		return nil, domainerrors.Forbidden("merchant context required")
	}
	if strings.TrimSpace(input.InvoiceCurrency) == "" {
		return nil, domainerrors.BadRequest("invoice_currency is required")
	}
	if strings.TrimSpace(input.InvoiceAmount) == "" {
		return nil, domainerrors.BadRequest("invoice_amount is required")
	}
	if strings.TrimSpace(input.SelectedChain) == "" {
		return nil, domainerrors.BadRequest("selected_chain is required")
	}
	if strings.TrimSpace(input.SelectedToken) == "" {
		return nil, domainerrors.BadRequest("selected_token is required")
	}
	if strings.TrimSpace(input.DestWallet) == "" {
		return nil, domainerrors.BadRequest("dest_wallet is required")
	}
	if u.routeSupportFn == nil || u.swapQuoteFn == nil {
		return nil, domainerrors.InternalServerError("partner quote engine is not configured")
	}

	chainID, chainCAIP2, err := u.chainResolver.ResolveFromAny(ctx, input.SelectedChain)
	if err != nil {
		return nil, domainerrors.BadRequest(fmt.Sprintf("invalid selected_chain: %v", err))
	}
	chain, err := u.chainRepo.GetByID(ctx, chainID)
	if err != nil {
		return nil, domainerrors.BadRequest("selected_chain not found")
	}

	selectedToken, err := u.tokenRepo.GetByAddress(ctx, strings.TrimSpace(input.SelectedToken), chainID)
	if err != nil || selectedToken == nil || !selectedToken.IsActive {
		return nil, domainerrors.BadRequest("selected_token not supported on selected_chain")
	}

	invoiceToken, err := u.tokenRepo.GetBySymbol(ctx, strings.TrimSpace(input.InvoiceCurrency), chainID)
	if err != nil || invoiceToken == nil || !invoiceToken.IsActive {
		return nil, domainerrors.BadRequest("invoice_currency not supported on selected_chain")
	}

	amountIn := new(big.Int)
	if _, ok := amountIn.SetString(strings.TrimSpace(input.InvoiceAmount), 10); !ok || amountIn.Sign() <= 0 {
		return nil, domainerrors.BadRequest("invoice_amount must be a positive integer string")
	}

	routeStatus, err := u.routeSupportFn(ctx, chainID, invoiceToken.ContractAddress, selectedToken.ContractAddress)
	if err != nil {
		return nil, domainerrors.InternalServerError(fmt.Sprintf("failed to resolve route support: %v", err))
	}
	if routeStatus == nil || !routeStatus.Exists || !routeStatus.Executable {
		return nil, domainerrors.BadRequest("selected token pair is not supported on selected_chain")
	}

	var quotedAmount *big.Int
	priceSourceOverride := ""
	if u.accurateQuoteFn != nil {
		accurateQuote, quoteErr := u.accurateQuoteFn(ctx, chainID, invoiceToken.ContractAddress, selectedToken.ContractAddress, amountIn)
		if quoteErr != nil {
			return nil, domainerrors.BadRequest(fmt.Sprintf("accurate quote unavailable for selected pair: %v", quoteErr))
		}
		if accurateQuote != nil {
			quotedAmount = accurateQuote.AmountOut
			priceSourceOverride = strings.TrimSpace(accurateQuote.PriceSource)
		}
	} else {
		quotedAmount, err = u.swapQuoteFn(ctx, chainID, invoiceToken.ContractAddress, selectedToken.ContractAddress, amountIn)
		if err != nil {
			return nil, domainerrors.InternalServerError(fmt.Sprintf("failed to fetch on-chain quote: %v", err))
		}
	}
	if quotedAmount == nil || quotedAmount.Sign() <= 0 {
		return nil, domainerrors.InternalServerError("invalid on-chain quote amount")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(partnerQuoteTTL)
	routeSummary := u.summarizeRoute(ctx, chainID, invoiceToken, selectedToken, routeStatus)
	priceSource := buildPartnerPriceSource(chain, selectedToken.Symbol, invoiceToken.Symbol, routeStatus)
	if priceSourceOverride != "" {
		priceSource = priceSourceOverride
	}
	quoteRate := formatNormalizedTokenRatio(quotedAmount, selectedToken.Decimals, amountIn, invoiceToken.Decimals, 18)

	quote := &domainentities.PaymentQuote{
		ID:                    utils.GenerateUUIDv7(),
		MerchantID:            input.MerchantID,
		InvoiceCurrency:       strings.TrimSpace(input.InvoiceCurrency),
		InvoiceAmount:         amountIn.String(),
		SelectedChainID:       chainCAIP2,
		SelectedTokenAddress:  selectedToken.ContractAddress,
		SelectedTokenSymbol:   selectedToken.Symbol,
		SelectedTokenDecimals: selectedToken.Decimals,
		QuotedAmount:          quotedAmount.String(),
		QuoteRate:             quoteRate,
		PriceSource:           priceSource,
		Route:                 routeSummary,
		SlippageBps:           partnerQuoteSlippage,
		RateTimestamp:         now,
		ExpiresAt:             expiresAt,
		Status:                domainentities.PaymentQuoteStatusActive,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := u.quoteRepo.Create(ctx, quote); err != nil {
		return nil, domainerrors.InternalServerError(fmt.Sprintf("failed to persist quote: %v", err))
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

func (u *PartnerQuoteUsecase) summarizeRoute(
	ctx context.Context,
	chainID uuid.UUID,
	invoiceToken *domainentities.Token,
	selectedToken *domainentities.Token,
	status *TokenRouteSupportStatus,
) string {
	if status == nil || len(status.Path) == 0 {
		return fmt.Sprintf("%s->%s", invoiceToken.Symbol, selectedToken.Symbol)
	}

	parts := make([]string, 0, len(status.Path))
	for _, addr := range status.Path {
		if strings.EqualFold(addr, invoiceToken.ContractAddress) {
			parts = append(parts, invoiceToken.Symbol)
			continue
		}
		if strings.EqualFold(addr, selectedToken.ContractAddress) {
			parts = append(parts, selectedToken.Symbol)
			continue
		}

		token, err := u.tokenRepo.GetByAddress(ctx, addr, chainID)
		if err == nil && token != nil && strings.TrimSpace(token.Symbol) != "" {
			parts = append(parts, token.Symbol)
			continue
		}
		parts = append(parts, addr)
	}
	return strings.Join(parts, "->")
}

func buildPartnerPriceSource(chain *domainentities.Chain, selectedTokenSymbol string, invoiceTokenSymbol string, status *TokenRouteSupportStatus) string {
	prefix := "token-swapper"
	if status != nil {
		if strings.TrimSpace(status.UniversalV4) != "" && status.UniversalV4 != "0x0000000000000000000000000000000000000000" {
			prefix = "uniswap-v4"
		} else if strings.TrimSpace(status.SwapRouterV3) != "" && status.SwapRouterV3 != "0x0000000000000000000000000000000000000000" {
			prefix = "uniswap-v3"
		}
	}
	chainName := normalizePriceSourcePart(chain.Name)
	if chainName == "" {
		chainName = normalizePriceSourcePart(chain.GetCAIP2ID())
	}
	return fmt.Sprintf("%s-%s-%s-%s", prefix, chainName, normalizePriceSourcePart(selectedTokenSymbol), normalizePriceSourcePart(invoiceTokenSymbol))
}

func normalizePriceSourcePart(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "-")
	v = strings.ReplaceAll(v, "_", "-")
	return v
}

func formatDecimalRatio(numerator *big.Int, denominator *big.Int, precision int) string {
	if denominator == nil || denominator.Sign() == 0 {
		return "0"
	}
	if numerator == nil {
		return "0"
	}
	ratio := new(big.Rat).SetFrac(numerator, denominator)
	return trimTrailingZeros(ratio.FloatString(precision))
}

func formatNormalizedTokenRatio(outputAmount *big.Int, outputDecimals int, inputAmount *big.Int, inputDecimals int, precision int) string {
	if inputAmount == nil || inputAmount.Sign() == 0 {
		return "0"
	}
	if outputAmount == nil {
		return "0"
	}

	numerator := new(big.Rat).SetInt(outputAmount)
	denominator := new(big.Rat).SetInt(inputAmount)

	if inputDecimals > 0 {
		numerator.Mul(numerator, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(inputDecimals)), nil)))
	}
	if outputDecimals > 0 {
		denominator.Mul(denominator, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(outputDecimals)), nil)))
	}

	return trimTrailingZeros(new(big.Rat).Quo(numerator, denominator).FloatString(precision))
}

func trimTrailingZeros(v string) string {
	if !strings.Contains(v, ".") {
		return v
	}
	v = strings.TrimRight(v, "0")
	v = strings.TrimRight(v, ".")
	if v == "" {
		return "0"
	}
	return v
}
