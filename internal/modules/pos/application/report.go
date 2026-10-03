package application

import (
	"sort"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/pos/domain"
)

var zone = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}()

// DailyRow is one day of sales.
type DailyRow struct {
	Date         string  `json:"date"`
	Transactions int     `json:"transactions"`
	ItemsSold    float64 `json:"itemsSold"`
	GrossSales   float64 `json:"grossSales"`
	Discounts    float64 `json:"discounts"`
	Tax          float64 `json:"tax"`
	Refunds      float64 `json:"refunds"`
	NetSales     float64 `json:"netSales"`
	Voided       int     `json:"voided"`
}

type BreakdownRow struct {
	Name         string  `json:"name"`
	Transactions int     `json:"transactions"`
	Amount       float64 `json:"amount"`
}

type ProductRow struct {
	Name     string  `json:"name"`
	SKU      string  `json:"sku"`
	Quantity float64 `json:"quantity"`
	Revenue  float64 `json:"revenue"`
}

// SalesReport is the POS sales summary for a date range.
type SalesReport struct {
	From        string         `json:"from"`
	To          string         `json:"to"`
	Days        []DailyRow     `json:"days"`
	Totals      DailyRow       `json:"totals"`
	ByPayment   []BreakdownRow `json:"byPayment"`
	ByCashier   []BreakdownRow `json:"byCashier"`
	TopProducts []ProductRow   `json:"topProducts"`
}

func day(t time.Time) string { return t.In(zone).Format("2006-01-02") }

func sortedRows(m map[string]*BreakdownRow) []BreakdownRow {
	out := make([]BreakdownRow, 0, len(m))
	for _, r := range m {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Amount != out[j].Amount {
			return out[i].Amount > out[j].Amount
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// BuildSalesReport summarises sales and refunds. Voided sales are counted apart
// and never in the sales figures. A refund lowers the day it was made, not the
// day of the original sale, so a closed day's figures stay as reported.
func BuildSalesReport(from, to string, txs []domain.POSTransaction, refunds []domain.POSRefund) SalesReport {
	rep := SalesReport{From: from, To: to, Days: []DailyRow{}, ByPayment: []BreakdownRow{}, ByCashier: []BreakdownRow{}, TopProducts: []ProductRow{}}
	days := map[string]*DailyRow{}
	get := func(d string) *DailyRow {
		if days[d] == nil {
			days[d] = &DailyRow{Date: d}
		}
		return days[d]
	}
	pay, cashier := map[string]*BreakdownRow{}, map[string]*BreakdownRow{}
	products := map[string]*ProductRow{}
	bump := func(m map[string]*BreakdownRow, name string, amt float64, n int) {
		if strings.TrimSpace(name) == "" {
			name = "-"
		}
		if m[name] == nil {
			m[name] = &BreakdownRow{Name: name}
		}
		m[name].Amount += amt
		m[name].Transactions += n
	}
	voided := map[string]bool{}
	for _, tx := range txs {
		d := get(day(tx.CreatedAt))
		if tx.Status == "Voided" {
			d.Voided++
			voided[tx.ID.String()] = true
			continue
		}
		d.Transactions++
		d.GrossSales += tx.TotalAmount
		d.Discounts += tx.DiscountAmount
		d.Tax += tx.TaxAmount
		bump(pay, tx.PaymentMethod, tx.TotalAmount, 1)
		bump(cashier, tx.Cashier, tx.TotalAmount, 1)
		if len(tx.Lines) == 0 {
			d.ItemsSold += float64(tx.TotalItems)
		}
		for _, l := range tx.Lines {
			d.ItemsSold += l.Quantity
			key := l.ProductID.String()
			if products[key] == nil {
				products[key] = &ProductRow{Name: l.Name, SKU: l.SKU}
			}
			netQty := l.Quantity - l.RefundedQty
			products[key].Quantity += netQty
			products[key].Revenue += l.UnitPrice * netQty
		}
	}
	for _, rf := range refunds {
		if rf.Kind != "refund" || voided[rf.TransactionID.String()] {
			continue
		}
		get(day(rf.CreatedAt)).Refunds += rf.Amount
	}
	keys := make([]string, 0, len(days))
	for k := range days {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		d := days[k]
		d.GrossSales, d.Discounts, d.Tax, d.Refunds = round2(d.GrossSales), round2(d.Discounts), round2(d.Tax), round2(d.Refunds)
		d.NetSales = round2(d.GrossSales - d.Refunds)
		rep.Days = append(rep.Days, *d)
		rep.Totals.Transactions += d.Transactions
		rep.Totals.ItemsSold += d.ItemsSold
		rep.Totals.GrossSales += d.GrossSales
		rep.Totals.Discounts += d.Discounts
		rep.Totals.Tax += d.Tax
		rep.Totals.Refunds += d.Refunds
		rep.Totals.Voided += d.Voided
	}
	rep.Totals.Date = "total"
	rep.Totals.GrossSales, rep.Totals.Discounts, rep.Totals.Tax, rep.Totals.Refunds = round2(rep.Totals.GrossSales), round2(rep.Totals.Discounts), round2(rep.Totals.Tax), round2(rep.Totals.Refunds)
	rep.Totals.NetSales = round2(rep.Totals.GrossSales - rep.Totals.Refunds)
	rep.ByPayment, rep.ByCashier = sortedRows(pay), sortedRows(cashier)
	for _, p := range products {
		if p.Quantity > 0 {
			p.Revenue = round2(p.Revenue)
			rep.TopProducts = append(rep.TopProducts, *p)
		}
	}
	sort.Slice(rep.TopProducts, func(i, j int) bool {
		if rep.TopProducts[i].Quantity != rep.TopProducts[j].Quantity {
			return rep.TopProducts[i].Quantity > rep.TopProducts[j].Quantity
		}
		return rep.TopProducts[i].Name < rep.TopProducts[j].Name
	})
	if len(rep.TopProducts) > 10 {
		rep.TopProducts = rep.TopProducts[:10]
	}
	return rep
}
