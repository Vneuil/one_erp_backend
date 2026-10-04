package application

import (
	"context"
	"sort"
	"strings"

	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// salesLine is one sales-order line joined with its order and product. POS
// sales are sales orders too (channel POS), so this covers every channel once.
type salesLine struct {
	OrderID      uuid.UUID
	OrderDate    string
	CustomerName string
	Channel      string
	ProductID    uuid.UUID
	SKU          string
	Name         string
	Category     string
	Quantity     float64
	Subtotal     float64
	CostPrice    float64
	// POSUnitCost is the unit cost snapshot taken at the till, when the sale was a POS sale.
	POSUnitCost *float64
}

// unitCost prefers the cost snapshot of the sale, falling back to the product's current cost price.
func (l salesLine) unitCost() float64 {
	if l.POSUnitCost != nil && *l.POSUnitCost > 0 {
		return *l.POSUnitCost
	}
	return l.CostPrice
}

func (l salesLine) cost() float64 { return l.Quantity * l.unitCost() }

func (q *ReportQuery) salesLines(ctx context.Context, from, to string) ([]salesLine, error) {
	var rows []salesLine
	db := scope(ctx, q.db.WithContext(ctx).Table("sales_order_lines l").
		Joins("JOIN sales_orders o ON o.id = l.sales_order_id AND o.deleted_at IS NULL").
		Joins("LEFT JOIN products p ON p.id = l.product_id").
		Where("l.deleted_at IS NULL").
		Where("LOWER(o.status) NOT IN ?", notCountedOrder).
		Where("o.order_date >= ? AND o.order_date <= ?", from, to), "o")
	err := db.Select(`o.id AS order_id, o.order_date, o.customer_name, o.channel, l.product_id,
		COALESCE(p.sku,'') AS sku, COALESCE(p.name,'(produk dihapus)') AS name, COALESCE(p.category,'') AS category,
		l.quantity, l.subtotal, COALESCE(p.cost_price,0) AS cost_price,
		(SELECT NULLIF(MAX(pl.unit_cost),0) FROM pos_transaction_lines pl JOIN pos_transactions t ON t.id = pl.transaction_id
		  WHERE t.sales_order_id = o.id AND pl.product_id = l.product_id AND pl.deleted_at IS NULL) AS pos_unit_cost`).
		Order("o.order_date asc, o.created_at asc").Scan(&rows).Error
	return rows, err
}

// Sales by product (Laporan Penjualan Per Produk / Peringkat Produk)

type ProductSalesRow struct {
	Rank        int       `json:"rank"`
	ProductID   uuid.UUID `json:"productId"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Quantity    float64   `json:"quantity"`
	Revenue     float64   `json:"revenue"`
	AvgPrice    float64   `json:"avgPrice"`
	Cost        float64   `json:"cost"`
	GrossProfit float64   `json:"grossProfit"`
	MarginPct   float64   `json:"marginPct"`
}

type SalesByProductDTO struct {
	From        string            `json:"from"`
	To          string            `json:"to"`
	Rows        []ProductSalesRow `json:"rows"`
	Quantity    float64           `json:"quantity"`
	Revenue     float64           `json:"revenue"`
	Cost        float64           `json:"cost"`
	GrossProfit float64           `json:"grossProfit"`
	MarginPct   float64           `json:"marginPct"`
	// Note explains the cost basis, which is an estimate for non-POS sales.
	Note string `json:"note"`
}

const costNote = "Revenue is the sum of order line amounts before order-level discount and tax. Cost uses the unit cost captured at the till for POS sales and the product's current cost price otherwise, so margins on other channels are estimates."

func aggregateByProduct(lines []salesLine) []ProductSalesRow {
	byProduct := map[uuid.UUID]*ProductSalesRow{}
	for _, l := range lines {
		r := byProduct[l.ProductID]
		if r == nil {
			r = &ProductSalesRow{ProductID: l.ProductID, SKU: l.SKU, Name: l.Name, Category: l.Category}
			byProduct[l.ProductID] = r
		}
		r.Quantity += l.Quantity
		r.Revenue += l.Subtotal
		r.Cost += l.cost()
	}
	rows := make([]ProductSalesRow, 0, len(byProduct))
	for _, r := range byProduct {
		r.Quantity, r.Revenue, r.Cost = round2(r.Quantity), round2(r.Revenue), round2(r.Cost)
		if r.Quantity > 0 {
			r.AvgPrice = round2(r.Revenue / r.Quantity)
		}
		r.GrossProfit = round2(r.Revenue - r.Cost)
		r.MarginPct = pct(r.GrossProfit, r.Revenue)
		rows = append(rows, *r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Revenue != rows[j].Revenue {
			return rows[i].Revenue > rows[j].Revenue
		}
		return rows[i].Name < rows[j].Name
	})
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows
}

func (q *ReportQuery) SalesByProduct(ctx context.Context, from, to string) (*SalesByProductDTO, error) {
	from, to, err := resolveRange(from, to)
	if err != nil {
		return nil, err
	}
	lines, err := q.salesLines(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load sales lines")
	}
	out := &SalesByProductDTO{From: from, To: to, Rows: aggregateByProduct(lines), Note: costNote}
	for _, r := range out.Rows {
		out.Quantity, out.Revenue, out.Cost = out.Quantity+r.Quantity, out.Revenue+r.Revenue, out.Cost+r.Cost
	}
	out.Quantity, out.Revenue, out.Cost = round2(out.Quantity), round2(out.Revenue), round2(out.Cost)
	out.GrossProfit = round2(out.Revenue - out.Cost)
	out.MarginPct = pct(out.GrossProfit, out.Revenue)
	return out, nil
}

// Daily product sales (Laporan Produk Harian)

type DailyProductRow struct {
	Date      string    `json:"date"`
	ProductID uuid.UUID `json:"productId"`
	SKU       string    `json:"sku"`
	Name      string    `json:"name"`
	Quantity  float64   `json:"quantity"`
	Revenue   float64   `json:"revenue"`
}

type DailyProductSalesDTO struct {
	From string            `json:"from"`
	To   string            `json:"to"`
	Rows []DailyProductRow `json:"rows"`
}

func aggregateDailyProduct(lines []salesLine) []DailyProductRow {
	type key struct {
		date string
		id   uuid.UUID
	}
	agg := map[key]*DailyProductRow{}
	for _, l := range lines {
		k := key{l.OrderDate, l.ProductID}
		r := agg[k]
		if r == nil {
			r = &DailyProductRow{Date: l.OrderDate, ProductID: l.ProductID, SKU: l.SKU, Name: l.Name}
			agg[k] = r
		}
		r.Quantity += l.Quantity
		r.Revenue += l.Subtotal
	}
	rows := make([]DailyProductRow, 0, len(agg))
	for _, r := range agg {
		r.Quantity, r.Revenue = round2(r.Quantity), round2(r.Revenue)
		rows = append(rows, *r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Date != rows[j].Date {
			return rows[i].Date < rows[j].Date
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

func (q *ReportQuery) DailyProductSales(ctx context.Context, from, to string) (*DailyProductSalesDTO, error) {
	from, to, err := resolveRange(from, to)
	if err != nil {
		return nil, err
	}
	lines, err := q.salesLines(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load sales lines")
	}
	return &DailyProductSalesDTO{From: from, To: to, Rows: aggregateDailyProduct(lines)}, nil
}

// Sales by customer (Laporan Penjualan Per Customer / Peringkat Customer)

type orderRow struct {
	OrderDate    string
	CustomerName string
	TotalAmount  float64
}

type CustomerSalesRow struct {
	Rank      int     `json:"rank"`
	Customer  string  `json:"customer"`
	Orders    int     `json:"orders"`
	Revenue   float64 `json:"revenue"`
	AvgOrder  float64 `json:"avgOrder"`
	FirstDate string  `json:"firstDate"`
	LastDate  string  `json:"lastDate"`
	// ItemsRevenue and Cost cover only orders that have product lines.
	ItemsRevenue float64 `json:"itemsRevenue"`
	Cost         float64 `json:"cost"`
	GrossProfit  float64 `json:"grossProfit"`
	MarginPct    float64 `json:"marginPct"`
}

type SalesByCustomerDTO struct {
	From    string             `json:"from"`
	To      string             `json:"to"`
	Rows    []CustomerSalesRow `json:"rows"`
	Orders  int                `json:"orders"`
	Revenue float64            `json:"revenue"`
	Note    string             `json:"note"`
}

func aggregateByCustomer(orders []orderRow, lines []salesLine) []CustomerSalesRow {
	byName := map[string]*CustomerSalesRow{}
	get := func(name string) *CustomerSalesRow {
		k := strings.ToLower(strings.TrimSpace(name))
		r := byName[k]
		if r == nil {
			r = &CustomerSalesRow{Customer: strings.TrimSpace(name)}
			byName[k] = r
		}
		return r
	}
	for _, o := range orders {
		r := get(o.CustomerName)
		r.Orders++
		r.Revenue += o.TotalAmount
		if r.FirstDate == "" || o.OrderDate < r.FirstDate {
			r.FirstDate = o.OrderDate
		}
		if o.OrderDate > r.LastDate {
			r.LastDate = o.OrderDate
		}
	}
	for _, l := range lines {
		r := get(l.CustomerName)
		r.ItemsRevenue += l.Subtotal
		r.Cost += l.cost()
	}
	rows := make([]CustomerSalesRow, 0, len(byName))
	for _, r := range byName {
		if r.Orders == 0 {
			continue
		}
		r.Revenue, r.ItemsRevenue, r.Cost = round2(r.Revenue), round2(r.ItemsRevenue), round2(r.Cost)
		r.AvgOrder = round2(r.Revenue / float64(r.Orders))
		r.GrossProfit = round2(r.ItemsRevenue - r.Cost)
		r.MarginPct = pct(r.GrossProfit, r.ItemsRevenue)
		rows = append(rows, *r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Revenue != rows[j].Revenue {
			return rows[i].Revenue > rows[j].Revenue
		}
		return rows[i].Customer < rows[j].Customer
	})
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows
}

func (q *ReportQuery) salesOrders(ctx context.Context, from, to string) ([]orderRow, error) {
	var rows []orderRow
	err := scope(ctx, q.db.WithContext(ctx).Table("sales_orders o").
		Where("o.deleted_at IS NULL").
		Where("LOWER(o.status) NOT IN ?", notCountedOrder).
		Where("o.order_date >= ? AND o.order_date <= ?", from, to), "o").
		Select("o.order_date, o.customer_name, o.total_amount").Order("o.order_date asc").Scan(&rows).Error
	return rows, err
}

func (q *ReportQuery) SalesByCustomer(ctx context.Context, from, to string) (*SalesByCustomerDTO, error) {
	from, to, err := resolveRange(from, to)
	if err != nil {
		return nil, err
	}
	orders, err := q.salesOrders(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load sales orders")
	}
	lines, err := q.salesLines(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load sales lines")
	}
	out := &SalesByCustomerDTO{From: from, To: to, Rows: aggregateByCustomer(orders, lines),
		Note: "Revenue is the order total (after discount and tax). Gross profit covers only the part of each order that has product lines. " + costNote}
	for _, r := range out.Rows {
		out.Orders += r.Orders
		out.Revenue += r.Revenue
	}
	out.Revenue = round2(out.Revenue)
	return out, nil
}

// Sales summary and averages (Laporan Rata-rata Penjualan)

type SalesPeriodRow struct {
	Period     string  `json:"period"`
	Orders     int     `json:"orders"`
	Revenue    float64 `json:"revenue"`
	ActiveDays int     `json:"activeDays"`
	AvgPerDay  float64 `json:"avgPerDay"`
}

type SalesDayRow struct {
	Date    string  `json:"date"`
	Orders  int     `json:"orders"`
	Revenue float64 `json:"revenue"`
}

type SalesSummaryDTO struct {
	From            string           `json:"from"`
	To              string           `json:"to"`
	Orders          int              `json:"orders"`
	Revenue         float64          `json:"revenue"`
	AvgOrderValue   float64          `json:"avgOrderValue"`
	Days            int              `json:"days"`
	ActiveDays      int              `json:"activeDays"`
	AvgPerDay       float64          `json:"avgPerDay"`
	AvgPerActiveDay float64          `json:"avgPerActiveDay"`
	AvgOrdersPerDay float64          `json:"avgOrdersPerDay"`
	Daily           []SalesDayRow    `json:"daily"`
	Monthly         []SalesPeriodRow `json:"monthly"`
}

func summarizeSales(from, to string, orders []orderRow) *SalesSummaryDTO {
	out := &SalesSummaryDTO{From: from, To: to, Days: daysInclusive(from, to), Daily: []SalesDayRow{}, Monthly: []SalesPeriodRow{}}
	daily := map[string]*SalesDayRow{}
	for _, o := range orders {
		d := daily[o.OrderDate]
		if d == nil {
			d = &SalesDayRow{Date: o.OrderDate}
			daily[o.OrderDate] = d
		}
		d.Orders++
		d.Revenue += o.TotalAmount
		out.Orders++
		out.Revenue += o.TotalAmount
	}
	monthly := map[string]*SalesPeriodRow{}
	for _, date := range sortedKeys(daily) {
		d := daily[date]
		d.Revenue = round2(d.Revenue)
		out.Daily = append(out.Daily, *d)
		if len(date) >= 7 {
			m := monthly[date[:7]]
			if m == nil {
				m = &SalesPeriodRow{Period: date[:7]}
				monthly[date[:7]] = m
			}
			m.Orders += d.Orders
			m.Revenue += d.Revenue
			m.ActiveDays++
		}
	}
	for _, p := range sortedKeys(monthly) {
		m := monthly[p]
		m.Revenue = round2(m.Revenue)
		if m.ActiveDays > 0 {
			m.AvgPerDay = round2(m.Revenue / float64(m.ActiveDays))
		}
		out.Monthly = append(out.Monthly, *m)
	}
	out.Revenue = round2(out.Revenue)
	out.ActiveDays = len(daily)
	if out.Orders > 0 {
		out.AvgOrderValue = round2(out.Revenue / float64(out.Orders))
	}
	if out.Days > 0 {
		out.AvgPerDay = round2(out.Revenue / float64(out.Days))
		out.AvgOrdersPerDay = round2(float64(out.Orders) / float64(out.Days))
	}
	if out.ActiveDays > 0 {
		out.AvgPerActiveDay = round2(out.Revenue / float64(out.ActiveDays))
	}
	return out
}

func (q *ReportQuery) SalesSummary(ctx context.Context, from, to string) (*SalesSummaryDTO, error) {
	from, to, err := resolveRange(from, to)
	if err != nil {
		return nil, err
	}
	orders, err := q.salesOrders(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load sales orders")
	}
	return summarizeSales(from, to, orders), nil
}
