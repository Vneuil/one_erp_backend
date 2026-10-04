package application

import (
	"context"
	"sort"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeInfra "github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
)

// Gross profit margin analysis (Analisa Gross Profit Margin)

type LedgerMargin struct {
	Sales          float64 `json:"sales"`
	Discounts      float64 `json:"discounts"`
	OtherCharges   float64 `json:"additionalCharges"`
	NetSales       float64 `json:"netSales"`
	CostOfGoodsSld float64 `json:"costOfGoodsSold"`
	GrossProfit    float64 `json:"grossProfit"`
	MarginPct      float64 `json:"marginPct"`
}

type CategoryMargin struct {
	Category    string  `json:"category"`
	Quantity    float64 `json:"quantity"`
	Revenue     float64 `json:"revenue"`
	Cost        float64 `json:"cost"`
	GrossProfit float64 `json:"grossProfit"`
	MarginPct   float64 `json:"marginPct"`
}

type GrossMarginDTO struct {
	From string `json:"from"`
	To   string `json:"to"`
	// Ledger is what was actually posted: sales (4000) less discounts (4100)
	// plus additional charges (4200), less cost of goods sold (5200).
	Ledger LedgerMargin `json:"ledger"`
	// Items is the estimate from order lines (see Note); ByCategory and
	// ByProduct break it down.
	Items      CategoryMargin    `json:"items"`
	ByCategory []CategoryMargin  `json:"byCategory"`
	ByProduct  []ProductSalesRow `json:"byProduct"`
	Note       string            `json:"note"`
}

func marginByCategory(lines []salesLine) ([]CategoryMargin, CategoryMargin) {
	byCat := map[string]*CategoryMargin{}
	var total CategoryMargin
	for _, l := range lines {
		cat := l.Category
		if cat == "" {
			cat = "(tanpa kategori)"
		}
		c := byCat[cat]
		if c == nil {
			c = &CategoryMargin{Category: cat}
			byCat[cat] = c
		}
		c.Quantity += l.Quantity
		c.Revenue += l.Subtotal
		c.Cost += l.cost()
		total.Quantity += l.Quantity
		total.Revenue += l.Subtotal
		total.Cost += l.cost()
	}
	fin := func(c *CategoryMargin) {
		c.Quantity, c.Revenue, c.Cost = round2(c.Quantity), round2(c.Revenue), round2(c.Cost)
		c.GrossProfit = round2(c.Revenue - c.Cost)
		c.MarginPct = pct(c.GrossProfit, c.Revenue)
	}
	out := make([]CategoryMargin, 0, len(byCat))
	for _, c := range byCat {
		fin(c)
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GrossProfit > out[j].GrossProfit })
	total.Category = "Total"
	fin(&total)
	return out, total
}

func (q *ReportQuery) GrossMargin(ctx context.Context, from, to string) (*GrossMarginDTO, error) {
	from, to, err := resolveRange(from, to)
	if err != nil {
		return nil, err
	}
	lines, err := q.salesLines(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load sales lines")
	}
	out := &GrossMarginDTO{From: from, To: to, ByProduct: aggregateByProduct(lines), Note: costNote}
	out.ByCategory, out.Items = marginByCategory(lines)

	repo := financeInfra.NewFinanceRepository(q.db)
	accounts, err := repo.ListAllAccounts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list accounts")
	}
	sums, err := repo.SumPostedByAccount(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to compute posted totals")
	}
	code := map[string]string{}
	for _, a := range accounts {
		code[a.ID.String()] = a.Code
	}
	m := &out.Ledger
	for _, s := range sums {
		switch code[s.AccountID.String()] {
		case financeApp.AccountRevenue:
			m.Sales += s.TotalCredit - s.TotalDebit
		case financeApp.AccountSalesDiscount:
			m.Discounts += s.TotalDebit - s.TotalCredit
		case financeApp.AccountAdditionalCharges:
			m.OtherCharges += s.TotalCredit - s.TotalDebit
		case financeApp.AccountCOGS:
			m.CostOfGoodsSld += s.TotalDebit - s.TotalCredit
		}
	}
	m.Sales, m.Discounts, m.OtherCharges, m.CostOfGoodsSld = round2(m.Sales), round2(m.Discounts), round2(m.OtherCharges), round2(m.CostOfGoodsSld)
	m.NetSales = round2(m.Sales - m.Discounts + m.OtherCharges)
	m.GrossProfit = round2(m.NetSales - m.CostOfGoodsSld)
	m.MarginPct = pct(m.GrossProfit, m.NetSales)
	return out, nil
}
