package application

import (
	"context"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/currency/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

type CurrencyUseCase interface {
	SeedBaseCurrency(ctx context.Context, code, name, symbol string) error

	CreateCurrency(ctx context.Context, dto CreateCurrencyDTO) (*CurrencyResponseDTO, error)
	ListCurrencies(ctx context.Context) ([]CurrencyResponseDTO, error)
	DeleteCurrency(ctx context.Context, id uuid.UUID) error

	SetExchangeRate(ctx context.Context, dto SetExchangeRateDTO) (*ExchangeRateResponseDTO, error)
	ListExchangeRates(ctx context.Context, currencyCode string) ([]ExchangeRateResponseDTO, error)

	// Convert converts amount (in fromCurrency) to the company's base
	// currency, using the latest exchange rate on or before asOfDate. If
	// fromCurrency is already the base currency, it's a 1:1 passthrough
	// (no rate lookup needed) - this is what makes multi-currency opt-in:
	// a company that only ever uses its base currency never needs a
	// single exchange rate configured.
	Convert(ctx context.Context, amount float64, fromCurrency string, asOfDate string) (*ConvertResultDTO, error)
}

type currencyUseCase struct {
	repo domain.CurrencyRepository
}

func NewCurrencyUseCase(repo domain.CurrencyRepository) CurrencyUseCase {
	return &currencyUseCase{repo: repo}
}

func (uc *currencyUseCase) SeedBaseCurrency(ctx context.Context, code, name, symbol string) error {
	existing, err := uc.repo.GetBaseCurrency(ctx)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	c := &domain.Currency{Code: strings.ToUpper(code), Name: name, Symbol: symbol, IsBase: true, IsActive: true}
	return uc.repo.CreateCurrency(ctx, c)
}

func (uc *currencyUseCase) CreateCurrency(ctx context.Context, dto CreateCurrencyDTO) (*CurrencyResponseDTO, error) {
	code := strings.ToUpper(strings.TrimSpace(dto.Code))
	name := strings.TrimSpace(dto.Name)
	if code == "" || name == "" {
		return nil, apperrors.NewBadRequest("Currency code and name are required")
	}
	existing, err := uc.repo.GetCurrencyByCode(ctx, code)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing currency")
	}
	if existing != nil {
		return nil, apperrors.NewConflict("This currency is already added")
	}
	c := &domain.Currency{Code: code, Name: name, Symbol: dto.Symbol, IsActive: true}
	if err := uc.repo.CreateCurrency(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to add currency")
	}
	res := ToCurrencyResponse(c)
	return &res, nil
}

func (uc *currencyUseCase) ListCurrencies(ctx context.Context) ([]CurrencyResponseDTO, error) {
	items, err := uc.repo.ListCurrencies(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list currencies")
	}
	return ToCurrencyResponseList(items), nil
}

func (uc *currencyUseCase) DeleteCurrency(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteCurrency(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to remove currency")
	}
	return nil
}

func (uc *currencyUseCase) SetExchangeRate(ctx context.Context, dto SetExchangeRateDTO) (*ExchangeRateResponseDTO, error) {
	code := strings.ToUpper(strings.TrimSpace(dto.CurrencyCode))
	if code == "" {
		return nil, apperrors.NewBadRequest("currencyCode is required")
	}
	if dto.RateToBase <= 0 {
		return nil, apperrors.NewBadRequest("rateToBase must be greater than 0")
	}
	rateDate := dto.RateDate
	if rateDate == "" {
		rateDate = time.Now().Format("2006-01-02")
	}
	rate := &domain.ExchangeRate{CurrencyCode: code, RateDate: rateDate, RateToBase: dto.RateToBase}
	if err := uc.repo.UpsertExchangeRate(ctx, rate); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save exchange rate")
	}
	res := ToExchangeRateResponse(rate)
	return &res, nil
}

func (uc *currencyUseCase) ListExchangeRates(ctx context.Context, currencyCode string) ([]ExchangeRateResponseDTO, error) {
	items, err := uc.repo.ListExchangeRates(ctx, strings.ToUpper(currencyCode))
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list exchange rates")
	}
	return ToExchangeRateResponseList(items), nil
}

func (uc *currencyUseCase) Convert(ctx context.Context, amount float64, fromCurrency string, asOfDate string) (*ConvertResultDTO, error) {
	fromCurrency = strings.ToUpper(strings.TrimSpace(fromCurrency))
	if asOfDate == "" {
		asOfDate = time.Now().Format("2006-01-02")
	}

	base, err := uc.repo.GetBaseCurrency(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get base currency")
	}
	baseCode := "IDR"
	if base != nil {
		baseCode = base.Code
	}

	if fromCurrency == "" || fromCurrency == baseCode {
		return &ConvertResultDTO{
			OriginalAmount: amount, OriginalCurrency: baseCode,
			BaseAmount: amount, BaseCurrency: baseCode,
			RateToBase: 1, RateDate: asOfDate,
		}, nil
	}

	rate, err := uc.repo.GetLatestRate(ctx, fromCurrency, asOfDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to look up exchange rate")
	}
	if rate == nil {
		return nil, apperrors.NewNotFound("No exchange rate configured for " + fromCurrency + " on or before " + asOfDate)
	}

	return &ConvertResultDTO{
		OriginalAmount: amount, OriginalCurrency: fromCurrency,
		BaseAmount: amount * rate.RateToBase, BaseCurrency: baseCode,
		RateToBase: rate.RateToBase, RateDate: rate.RateDate,
	}, nil
}
