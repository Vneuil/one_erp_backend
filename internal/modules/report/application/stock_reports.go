package application

import (
	"context"
	"sort"
	"strings"
	"time"

	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// Stock card (Kartu Stock / Laporan Mutasi)

type stockMovementRow struct {
	CreatedAt     time.Time
	ProductID     uuid.UUID
	SKU           string
	ProductName   string
	CostPrice     float64
	WarehouseID   uuid.UUID
	WarehouseName string
	Type          string
	Quantity      int
	Balance       int
	Reference     string
	Reason        string
	BatchNo       string
	CreatedBy     string
}

func (m stockMovementRow) date() string { return m.CreatedAt.In(wib).Format("2006-01-02") }

type StockCardEntry struct {
	Date      string `json:"date"`
	Time      string `json:"time"`
	Warehouse string `json:"warehouse"`
	Type      string `json:"type"`
	Reference string `json:"reference"`
	Reason    string `json:"reason"`
	BatchNo   string `json:"batchNo,omitempty"`
	In        int    `json:"in"`
	Out       int    `json:"out"`
	Balance   int    `json:"balance"`
}

type StockCardDTO struct {
	ProductID   uuid.UUID        `json:"productId"`
	SKU         string           `json:"sku"`
	ProductName string           `json:"productName"`
	WarehouseID *uuid.UUID       `json:"warehouseId,omitempty"`
	From        string           `json:"from"`
	To          string           `json:"to"`
	Opening     int              `json:"openingBalance"`
	Entries     []StockCardEntry `json:"entries"`
	TotalIn     int              `json:"totalIn"`
	TotalOut    int              `json:"totalOut"`
	Closing     int              `json:"closingBalance"`
}

// buildStockCard turns a product's movements (up to the end of the range, in
// time order) into a card. The opening balance is what each warehouse held
// after its last movement before the range; with several warehouses the
// running balance is their combined stock.
func buildStockCard(movements []stockMovementRow, from, to string) *StockCardDTO {
	out := &StockCardDTO{From: from, To: to, Entries: []StockCardEntry{}}
	lastBefore := map[uuid.UUID]int{}
	for _, m := range movements {
		if m.date() < from {
			lastBefore[m.WarehouseID] = m.Balance
		}
	}
	for _, b := range lastBefore {
		out.Opening += b
	}
	running := out.Opening
	for _, m := range movements {
		d := m.date()
		if d < from || d > to {
			continue
		}
		running += m.Quantity
		e := StockCardEntry{Date: d, Time: m.CreatedAt.In(wib).Format("15:04"), Warehouse: m.WarehouseName, Type: m.Type,
			Reference: m.Reference, Reason: m.Reason, BatchNo: m.BatchNo, Balance: running}
		if m.Quantity >= 0 {
			e.In = m.Quantity
			out.TotalIn += m.Quantity
		} else {
			e.Out = -m.Quantity
			out.TotalOut += -m.Quantity
		}
		out.Entries = append(out.Entries, e)
	}
	out.Closing = running
	return out
}

func (q *ReportQuery) StockCard(ctx context.Context, productID uuid.UUID, warehouseID *uuid.UUID, from, to string) (*StockCardDTO, error) {
	from, to, err := resolveRange(from, to)
	if err != nil {
		return nil, err
	}
	var product struct {
		SKU  string
		Name string
	}
	if err := q.db.WithContext(ctx).Table("products").Select("sku, name").Where("id = ? AND deleted_at IS NULL", productID).Scan(&product).Error; err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the product")
	}
	if product.Name == "" {
		return nil, apperrors.NewNotFound("Product not found")
	}
	// Fetch from the start: movements before the range establish the opening balance.
	db := scope(ctx, q.db.WithContext(ctx).Table("inventory_stock_movements m").
		Joins("LEFT JOIN inventory_warehouses w ON w.id = m.warehouse_id").
		Where("m.deleted_at IS NULL AND m.product_id = ?", productID).
		Where("m.created_at < ?", endOfDayUTC(to)), "m")
	if warehouseID != nil {
		db = db.Where("m.warehouse_id = ?", *warehouseID)
	}
	var rows []stockMovementRow
	if err := db.Select(`m.created_at, m.product_id, m.warehouse_id, COALESCE(w.name,'') AS warehouse_name, m.type, m.quantity, m.balance,
		m.reference, m.reason, m.batch_no, m.created_by`).Order("m.created_at asc, m.id asc").Scan(&rows).Error; err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load stock movements")
	}
	card := buildStockCard(rows, from, to)
	card.ProductID, card.SKU, card.ProductName, card.WarehouseID = productID, product.SKU, product.Name, warehouseID
	return card, nil
}

// endOfDayUTC is the first instant after the given WIB date, as a UTC time.
func endOfDayUTC(date string) time.Time {
	d, err := time.ParseInLocation("2006-01-02", date, wib)
	if err != nil {
		return time.Now().Add(24 * time.Hour)
	}
	return d.AddDate(0, 0, 1).UTC()
}

func startOfDayUTC(date string) time.Time {
	d, err := time.ParseInLocation("2006-01-02", date, wib)
	if err != nil {
		return time.Time{}
	}
	return d.UTC()
}

// Stock transaction detail (Laporan Perincian Transaksi Stock)

type StockTransactionRow struct {
	Date      string  `json:"date"`
	Time      string  `json:"time"`
	SKU       string  `json:"sku"`
	Product   string  `json:"product"`
	Warehouse string  `json:"warehouse"`
	Type      string  `json:"type"`
	Reference string  `json:"reference"`
	Reason    string  `json:"reason"`
	BatchNo   string  `json:"batchNo,omitempty"`
	CreatedBy string  `json:"createdBy,omitempty"`
	Quantity  int     `json:"quantity"`
	Balance   int     `json:"balance"`
	UnitCost  float64 `json:"unitCost"`
	Value     float64 `json:"value"`
}

type StockTransactionTypeTotal struct {
	Type     string  `json:"type"`
	Count    int     `json:"count"`
	Quantity int     `json:"quantity"`
	Value    float64 `json:"value"`
}

type StockTransactionsDTO struct {
	From      string                      `json:"from"`
	To        string                      `json:"to"`
	Rows      []StockTransactionRow       `json:"rows"`
	ByType    []StockTransactionTypeTotal `json:"byType"`
	Truncated bool                        `json:"truncated"`
	Note      string                      `json:"note"`
}

const stockTransactionLimit = 5000

func buildStockTransactions(from, to string, rows []stockMovementRow, truncated bool) *StockTransactionsDTO {
	out := &StockTransactionsDTO{From: from, To: to, Rows: []StockTransactionRow{}, ByType: []StockTransactionTypeTotal{}, Truncated: truncated,
		Note: "Value is the quantity times the product's current cost price."}
	byType := map[string]*StockTransactionTypeTotal{}
	for _, m := range rows {
		value := round2(float64(m.Quantity) * m.CostPrice)
		out.Rows = append(out.Rows, StockTransactionRow{Date: m.date(), Time: m.CreatedAt.In(wib).Format("15:04"), SKU: m.SKU, Product: m.ProductName,
			Warehouse: m.WarehouseName, Type: m.Type, Reference: m.Reference, Reason: m.Reason, BatchNo: m.BatchNo, CreatedBy: m.CreatedBy,
			Quantity: m.Quantity, Balance: m.Balance, UnitCost: m.CostPrice, Value: value})
		t := byType[m.Type]
		if t == nil {
			t = &StockTransactionTypeTotal{Type: m.Type}
			byType[m.Type] = t
		}
		t.Count++
		t.Quantity += m.Quantity
		t.Value += value
	}
	for _, k := range sortedKeys(byType) {
		t := byType[k]
		t.Value = round2(t.Value)
		out.ByType = append(out.ByType, *t)
	}
	return out
}

func (q *ReportQuery) StockTransactions(ctx context.Context, from, to, movementType string, productID, warehouseID *uuid.UUID) (*StockTransactionsDTO, error) {
	from, to, err := resolveRange(from, to)
	if err != nil {
		return nil, err
	}
	db := scope(ctx, q.db.WithContext(ctx).Table("inventory_stock_movements m").
		Joins("LEFT JOIN products p ON p.id = m.product_id").
		Joins("LEFT JOIN inventory_warehouses w ON w.id = m.warehouse_id").
		Where("m.deleted_at IS NULL").
		Where("m.created_at >= ? AND m.created_at < ?", startOfDayUTC(from), endOfDayUTC(to)), "m")
	if t := strings.TrimSpace(movementType); t != "" {
		db = db.Where("m.type = ?", t)
	}
	if productID != nil {
		db = db.Where("m.product_id = ?", *productID)
	}
	if warehouseID != nil {
		db = db.Where("m.warehouse_id = ?", *warehouseID)
	}
	var rows []stockMovementRow
	if err := db.Select(`m.created_at, m.product_id, COALESCE(p.sku,'') AS sku, COALESCE(p.name,'(produk dihapus)') AS product_name, COALESCE(p.cost_price,0) AS cost_price,
		m.warehouse_id, COALESCE(w.name,'') AS warehouse_name, m.type, m.quantity, m.balance, m.reference, m.reason, m.batch_no, m.created_by`).
		Order("m.created_at asc, m.id asc").Limit(stockTransactionLimit + 1).Scan(&rows).Error; err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load stock movements")
	}
	truncated := len(rows) > stockTransactionLimit
	if truncated {
		rows = rows[:stockTransactionLimit]
	}
	return buildStockTransactions(from, to, rows, truncated), nil
}

// Stock aging (Laporan Analisa Umur Stock)

// AgingBuckets are the age ranges in days; the last is open-ended.
var agingBuckets = []struct {
	Label string
	Max   int // inclusive upper bound in days; -1 = no limit
}{{"0-30", 30}, {"31-60", 60}, {"61-90", 90}, {">90", -1}}

type stockBatchRow struct {
	ProductID     uuid.UUID
	SKU           string
	ProductName   string
	Category      string
	CostPrice     float64
	WarehouseID   uuid.UUID
	WarehouseName string
	BatchNo       string
	ReceivedAt    string
	ExpiryDate    string
	Quantity      int
}

type stockLevelRow struct {
	ProductID     uuid.UUID
	SKU           string
	ProductName   string
	Category      string
	CostPrice     float64
	WarehouseID   uuid.UUID
	WarehouseName string
	Quantity      int
}

type StockAgingRow struct {
	SKU       string `json:"sku"`
	Product   string `json:"product"`
	Category  string `json:"category"`
	Warehouse string `json:"warehouse"`
	// Buckets holds the quantity per age range, in AgingBuckets order.
	Buckets []int `json:"buckets"`
	// Untracked is stock that has no batch (age unknown).
	Untracked int     `json:"untracked"`
	Total     int     `json:"total"`
	Value     float64 `json:"value"`
	// OldestDays is the age of the oldest batch still in stock.
	OldestDays int `json:"oldestDays"`
	// Expired is the quantity past its expiry date.
	Expired int `json:"expired"`
}

type StockAgingDTO struct {
	AsOf          string          `json:"asOf"`
	BucketLabels  []string        `json:"bucketLabels"`
	Rows          []StockAgingRow `json:"rows"`
	BucketTotals  []int           `json:"bucketTotals"`
	BucketValues  []float64       `json:"bucketValues"`
	UntrackedQty  int             `json:"untrackedQty"`
	UntrackedVal  float64         `json:"untrackedValue"`
	TotalQuantity int             `json:"totalQuantity"`
	TotalValue    float64         `json:"totalValue"`
	Note          string          `json:"note"`
}

func bucketIndex(days int) int {
	for i, b := range agingBuckets {
		if b.Max < 0 || days <= b.Max {
			return i
		}
	}
	return len(agingBuckets) - 1
}

func ageDays(receivedAt, asOf string) (int, bool) {
	r, err1 := time.Parse("2006-01-02", receivedAt)
	a, err2 := time.Parse("2006-01-02", asOf)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	d := int(a.Sub(r).Hours() / 24)
	if d < 0 {
		d = 0
	}
	return d, true
}

// buildStockAging ages the batch-tracked part of each stock level by receipt
// date. Stock a level holds beyond its batches has no age and is reported as
// untracked rather than guessed.
func buildStockAging(asOf string, levels []stockLevelRow, batches []stockBatchRow) *StockAgingDTO {
	labels := make([]string, len(agingBuckets))
	for i, b := range agingBuckets {
		labels[i] = b.Label
	}
	out := &StockAgingDTO{AsOf: asOf, BucketLabels: labels, Rows: []StockAgingRow{}, BucketTotals: make([]int, len(labels)), BucketValues: make([]float64, len(labels)),
		Note: "Age is days since the batch was received. Only batch-tracked stock can be aged; the rest is shown as untracked. Value is quantity times current cost price, and quantities are current stock, not as of a past date."}

	type key struct{ product, warehouse uuid.UUID }
	rows := map[key]*StockAgingRow{}
	costs := map[key]float64{}
	batchQty := map[key]int{}
	get := func(k key, sku, name, cat, wh string, cost float64) *StockAgingRow {
		r := rows[k]
		if r == nil {
			r = &StockAgingRow{SKU: sku, Product: name, Category: cat, Warehouse: wh, Buckets: make([]int, len(labels))}
			rows[k] = r
			costs[k] = cost
		}
		return r
	}
	for _, l := range levels {
		if l.Quantity > 0 {
			get(key{l.ProductID, l.WarehouseID}, l.SKU, l.ProductName, l.Category, l.WarehouseName, l.CostPrice)
		}
	}
	for _, b := range batches {
		if b.Quantity <= 0 {
			continue
		}
		k := key{b.ProductID, b.WarehouseID}
		r := get(k, b.SKU, b.ProductName, b.Category, b.WarehouseName, b.CostPrice)
		days, ok := ageDays(b.ReceivedAt, asOf)
		if !ok {
			r.Untracked += b.Quantity
		} else {
			r.Buckets[bucketIndex(days)] += b.Quantity
			if days > r.OldestDays {
				r.OldestDays = days
			}
		}
		if b.ExpiryDate != "" && b.ExpiryDate < asOf {
			r.Expired += b.Quantity
		}
		batchQty[k] += b.Quantity
	}
	levelQty := map[key]int{}
	for _, l := range levels {
		levelQty[key{l.ProductID, l.WarehouseID}] += l.Quantity
	}
	for k, r := range rows {
		if extra := levelQty[k] - batchQty[k]; extra > 0 {
			r.Untracked += extra
		}
		for i, q := range r.Buckets {
			r.Total += q
			out.BucketTotals[i] += q
			out.BucketValues[i] += float64(q) * costs[k]
		}
		r.Total += r.Untracked
		out.UntrackedQty += r.Untracked
		out.UntrackedVal += float64(r.Untracked) * costs[k]
		r.Value = round2(float64(r.Total) * costs[k])
		out.TotalQuantity += r.Total
		out.TotalValue += r.Value
		out.Rows = append(out.Rows, *r)
	}
	for i := range out.BucketValues {
		out.BucketValues[i] = round2(out.BucketValues[i])
	}
	out.UntrackedVal, out.TotalValue = round2(out.UntrackedVal), round2(out.TotalValue)
	sort.Slice(out.Rows, func(i, j int) bool {
		if out.Rows[i].OldestDays != out.Rows[j].OldestDays {
			return out.Rows[i].OldestDays > out.Rows[j].OldestDays
		}
		return out.Rows[i].Product < out.Rows[j].Product
	})
	return out
}

func (q *ReportQuery) StockAging(ctx context.Context, asOf string, warehouseID *uuid.UUID) (*StockAgingDTO, error) {
	if asOf == "" {
		asOf = today()
	}
	if !validDate(asOf) {
		return nil, apperrors.NewBadRequest("asOf must be YYYY-MM-DD")
	}
	lvl := scope(ctx, q.db.WithContext(ctx).Table("inventory_stock_levels sl").
		Joins("JOIN products p ON p.id = sl.product_id AND p.deleted_at IS NULL").
		Joins("LEFT JOIN inventory_warehouses w ON w.id = sl.warehouse_id").
		Where("sl.deleted_at IS NULL AND sl.quantity > 0"), "sl")
	bat := scope(ctx, q.db.WithContext(ctx).Table("inventory_stock_batches b").
		Joins("JOIN products p ON p.id = b.product_id AND p.deleted_at IS NULL").
		Joins("LEFT JOIN inventory_warehouses w ON w.id = b.warehouse_id").
		Where("b.deleted_at IS NULL AND b.quantity > 0"), "b")
	if warehouseID != nil {
		lvl = lvl.Where("sl.warehouse_id = ?", *warehouseID)
		bat = bat.Where("b.warehouse_id = ?", *warehouseID)
	}
	var levels []stockLevelRow
	if err := lvl.Select(`sl.product_id, p.sku, p.name AS product_name, p.category, p.cost_price, sl.warehouse_id, COALESCE(w.name,'') AS warehouse_name, sl.quantity`).
		Scan(&levels).Error; err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load stock levels")
	}
	var batches []stockBatchRow
	if err := bat.Select(`b.product_id, p.sku, p.name AS product_name, p.category, p.cost_price, b.warehouse_id, COALESCE(w.name,'') AS warehouse_name,
		b.batch_no, b.received_at, b.expiry_date, b.quantity`).Scan(&batches).Error; err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load stock batches")
	}
	return buildStockAging(asOf, levels, batches), nil
}
