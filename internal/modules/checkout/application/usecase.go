package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/payment"
	"github.com/divinecoid/one-backend/internal/modules/checkout/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// planPrice is the server-side source of truth for what a plan/cycle costs.
// The public checkout form only ever supplies planCode/billingCycle - never
// a price - specifically so a tampered request can't buy a real
// subscription for a manipulated amount. Keep in sync with
// one-frontend/app/checkout/page.tsx's PLAN_PRICES table.
var planPrice = map[string]map[string]float64{
	"spark": {"monthly": 999_000, "annual": 9_999_000},
	"scale": {"monthly": 2_499_000, "annual": 24_999_000},
}

var planLabel = map[string]string{
	"spark": "Spark",
	"scale": "Scale",
}

type CheckoutUseCase interface {
	CreateOrder(ctx context.Context, dto CreateCheckoutOrderDTO) (*CreateCheckoutOrderResponseDTO, error)
	GetOrderStatus(ctx context.Context, id uuid.UUID) (*CheckoutOrderStatusDTO, error)
	// HandleXenditWebhook applies a Xendit invoice status update. Token
	// verification (X-Callback-Token) happens in the HTTP handler, before
	// this is ever called - this method trusts that it already ran.
	HandleXenditWebhook(ctx context.Context, providerRef, xenditStatus string) error
}

type checkoutUseCase struct {
	repo   domain.CheckoutOrderRepository
	client payment.Client
}

func NewCheckoutUseCase(repo domain.CheckoutOrderRepository, client payment.Client) CheckoutUseCase {
	return &checkoutUseCase{repo: repo, client: client}
}

func (uc *checkoutUseCase) CreateOrder(ctx context.Context, dto CreateCheckoutOrderDTO) (*CreateCheckoutOrderResponseDTO, error) {
	planCode := strings.ToLower(strings.TrimSpace(dto.PlanCode))
	billingCycle := strings.ToLower(strings.TrimSpace(dto.BillingCycle))

	cycles, ok := planPrice[planCode]
	if !ok {
		return nil, apperrors.NewBadRequest("Unknown plan")
	}
	amount, ok := cycles[billingCycle]
	if !ok {
		return nil, apperrors.NewBadRequest("Unknown billing cycle")
	}

	if dto.CompanyName == "" || dto.ContactName == "" || dto.ContactEmail == "" {
		return nil, apperrors.NewBadRequest("Company name, contact name and contact email are required")
	}

	if !uc.client.Enabled() {
		return nil, apperrors.NewInternal(nil, "Online payment is not configured yet. Please contact support.")
	}

	order := &domain.CheckoutOrder{
		PlanCode:     planCode,
		BillingCycle: billingCycle,
		AmountIDR:    amount,
		CompanyName:  dto.CompanyName,
		ContactName:  dto.ContactName,
		ContactEmail: dto.ContactEmail,
		ContactPhone: dto.ContactPhone,
		Status:       domain.CheckoutStatusPending,
	}
	if err := uc.repo.Create(ctx, order); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create checkout order")
	}

	description := fmt.Sprintf("ONE ERP - Paket %s (%s)", planLabel[planCode], billingCycleLabel(billingCycle))
	inv, err := uc.client.CreateInvoice(ctx, order.ID.String(), amount, description, dto.ContactEmail)
	if err != nil {
		order.Status = domain.CheckoutStatusFailed
		_ = uc.repo.Update(ctx, order)
		return nil, apperrors.NewInternal(err, "Failed to create payment invoice")
	}

	order.ProviderRef = inv.ProviderRef
	order.CheckoutURL = inv.CheckoutURL
	order.Status = domain.CheckoutStatusProcessing
	if err := uc.repo.Update(ctx, order); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save checkout order")
	}

	return &CreateCheckoutOrderResponseDTO{ID: order.ID.String(), CheckoutURL: inv.CheckoutURL}, nil
}

func (uc *checkoutUseCase) GetOrderStatus(ctx context.Context, id uuid.UUID) (*CheckoutOrderStatusDTO, error) {
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get checkout order")
	}
	if order == nil {
		return nil, apperrors.NewNotFound("Checkout order not found")
	}
	return toStatusDTO(order), nil
}

// HandleXenditWebhook maps Xendit's invoice statuses (PAID, EXPIRED,
// PENDING) onto CheckoutStatus. Anything else is left as processing rather
// than guessed at, so an unrecognized future status from Xendit doesn't get
// silently mis-recorded as failed.
func (uc *checkoutUseCase) HandleXenditWebhook(ctx context.Context, providerRef, xenditStatus string) error {
	order, err := uc.repo.GetByProviderRef(ctx, providerRef)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to look up checkout order for webhook")
	}
	if order == nil {
		// Not an error from Xendit's perspective (still return 200) - just
		// nothing on our side to update, e.g. a retried webhook for an
		// order that was since deleted.
		return nil
	}

	switch strings.ToUpper(xenditStatus) {
	case "PAID", "SETTLED":
		order.Status = domain.CheckoutStatusPaid
		now := time.Now()
		order.PaidAt = &now
	case "EXPIRED":
		order.Status = domain.CheckoutStatusExpired
	case "FAILED":
		order.Status = domain.CheckoutStatusFailed
	default:
		return nil
	}
	return uc.repo.Update(ctx, order)
}

func billingCycleLabel(cycle string) string {
	if cycle == "annual" {
		return "Tahunan"
	}
	return "Bulanan"
}
