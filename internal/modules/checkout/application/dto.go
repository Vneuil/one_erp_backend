package application

import "github.com/divinecoid/one-backend/internal/modules/checkout/domain"

type CreateCheckoutOrderDTO struct {
	PlanCode     string `json:"planCode"`
	BillingCycle string `json:"billingCycle"`
	CompanyName  string `json:"companyName"`
	ContactName  string `json:"contactName"`
	ContactEmail string `json:"contactEmail"`
	ContactPhone string `json:"contactPhone"`
}

type CreateCheckoutOrderResponseDTO struct {
	ID          string `json:"id"`
	CheckoutURL string `json:"checkoutUrl"`
}

type CheckoutOrderStatusDTO struct {
	ID           string  `json:"id"`
	PlanCode     string  `json:"planCode"`
	BillingCycle string  `json:"billingCycle"`
	AmountIDR    float64 `json:"amountIdr"`
	Status       string  `json:"status"`
}

func toStatusDTO(o *domain.CheckoutOrder) *CheckoutOrderStatusDTO {
	return &CheckoutOrderStatusDTO{
		ID:           o.ID.String(),
		PlanCode:     o.PlanCode,
		BillingCycle: o.BillingCycle,
		AmountIDR:    o.AmountIDR,
		Status:       string(o.Status),
	}
}
