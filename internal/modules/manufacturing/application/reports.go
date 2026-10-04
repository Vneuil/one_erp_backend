package application

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func pct(part, whole float64) float64 {
	if whole == 0 {
		return 0
	}
	return round2(part / whole * 100)
}

var wib = time.FixedZone("WIB", 7*3600)

func reportToday() string { return time.Now().In(wib).Format("2006-01-02") }

func validDay(s string) bool { _, err := time.Parse("2006-01-02", s); return err == nil }

// reportRange validates from/to; both empty means the current month to date.
func reportRange(from, to string) (string, string, error) {
	for _, d := range []string{from, to} {
		if d != "" && !validDay(d) {
			return "", "", apperrors.NewBadRequest("from and to must be YYYY-MM-DD")
		}
	}
	if to == "" {
		to = reportToday()
	}
	if from == "" {
		from = to[:8] + "01"
	}
	if from > to {
		return "", "", apperrors.NewBadRequest("from cannot be after to")
	}
	return from, to, nil
}

// reportOrder is a production order with the names the reports show.
type reportOrder struct {
	domain.ProductionOrder
	Number      string
	ProductID   uuid.UUID
	ProductSKU  string
	ProductName string
}

func (o reportOrder) active() bool {
	return o.Status == domain.OrderReleased || o.Status == domain.OrderInProgress || o.Status == domain.OrderPaused
}

// refDate is the date an order is reported under: its planned date, else the day it was created.
func (o reportOrder) refDate() string {
	if validDay(o.PlannedDate) {
		return o.PlannedDate
	}
	return o.CreatedAt.In(wib).Format("2006-01-02")
}

func (uc *manufacturingUseCase) loadReportOrders(ctx context.Context) ([]reportOrder, map[uuid.UUID]*domain.BillOfMaterial, error) {
	orders, err := uc.repo.ListAllOrders(ctx)
	if err != nil {
		return nil, nil, apperrors.NewInternal(err, "Failed to load production orders")
	}
	boms := map[uuid.UUID]*domain.BillOfMaterial{}
	out := make([]reportOrder, 0, len(orders))
	for i := range orders {
		o := &orders[i]
		bom, ok := boms[o.BOMID]
		if !ok {
			b, err := uc.repo.GetBOMByID(ctx, o.BOMID)
			if err != nil {
				return nil, nil, apperrors.NewInternal(err, "Failed to load a BOM")
			}
			boms[o.BOMID], bom = b, b
		}
		ro := reportOrder{ProductionOrder: *o, Number: OrderLabel(o)}
		if bom != nil {
			ro.ProductID = bom.ProductID
			ro.ProductSKU, ro.ProductName = uc.productInfo(ctx, bom.ProductID)
		}
		out = append(out, ro)
	}
	return out, boms, nil
}

// Daily report (Laporan Order Produksi per Hari)

type DailyOrderLine struct {
	OrderID   uuid.UUID `json:"orderId"`
	Number    string    `json:"orderNumber"`
	Product   string    `json:"product"`
	Planned   int       `json:"quantityToProduce"`
	Completed int       `json:"quantityCompleted"`
	Status    string    `json:"status"`
}

type DailyQuantityLine struct {
	OrderID uuid.UUID `json:"orderId"`
	Number  string    `json:"orderNumber"`
	Product string    `json:"product"`
	Step    string    `json:"step,omitempty"`
	Qty     int       `json:"quantity"`
	Notes   string    `json:"notes,omitempty"`
}

type DailyReportDay struct {
	Date          string              `json:"date"`
	PlannedOrders []DailyOrderLine    `json:"plannedOrders"`
	StepOutputs   []DailyQuantityLine `json:"stepOutputs"`
	Completions   []DailyQuantityLine `json:"completions"`
	PlannedUnits  int                 `json:"plannedUnits"`
	StepUnits     int                 `json:"stepUnits"`
	UnitsDone     int                 `json:"unitsCompleted"`
}

type DailyReportDTO struct {
	From         string           `json:"from"`
	To           string           `json:"to"`
	Days         []DailyReportDay `json:"days"`
	PlannedUnits int              `json:"plannedUnits"`
	StepUnits    int              `json:"stepUnits"`
	UnitsDone    int              `json:"unitsCompleted"`
}

func buildDailyReport(from, to string, orders []reportOrder, logs []domain.ProductionStepLog, batches []domain.ProductionBatch) *DailyReportDTO {
	byID := map[uuid.UUID]reportOrder{}
	stepName := map[uuid.UUID]string{}
	for _, o := range orders {
		byID[o.ID] = o
		for _, s := range o.Steps {
			stepName[s.ID] = s.Name
		}
	}
	days := map[string]*DailyReportDay{}
	day := func(d string) *DailyReportDay {
		if days[d] == nil {
			days[d] = &DailyReportDay{Date: d, PlannedOrders: []DailyOrderLine{}, StepOutputs: []DailyQuantityLine{}, Completions: []DailyQuantityLine{}}
		}
		return days[d]
	}
	for _, o := range orders {
		if o.Status == domain.OrderCancelled || !validDay(o.PlannedDate) || o.PlannedDate < from || o.PlannedDate > to {
			continue
		}
		d := day(o.PlannedDate)
		d.PlannedOrders = append(d.PlannedOrders, DailyOrderLine{OrderID: o.ID, Number: o.Number, Product: o.ProductName, Planned: o.QuantityToProduce, Completed: o.QuantityCompleted, Status: o.Status})
		d.PlannedUnits += o.QuantityToProduce
	}
	for _, l := range logs {
		o := byID[l.OrderID]
		d := day(l.Date)
		d.StepOutputs = append(d.StepOutputs, DailyQuantityLine{OrderID: l.OrderID, Number: o.Number, Product: o.ProductName, Step: stepName[l.StepID], Qty: l.Quantity, Notes: l.Notes})
		d.StepUnits += l.Quantity
	}
	for _, b := range batches {
		o := byID[b.ProductionOrderID]
		d := day(b.CompletionDate)
		d.Completions = append(d.Completions, DailyQuantityLine{OrderID: b.ProductionOrderID, Number: o.Number, Product: o.ProductName, Qty: b.QuantityCompleted, Notes: b.Notes})
		d.UnitsDone += b.QuantityCompleted
	}
	out := &DailyReportDTO{From: from, To: to, Days: []DailyReportDay{}}
	for _, k := range sortedKeys(days) {
		d := days[k]
		out.Days = append(out.Days, *d)
		out.PlannedUnits += d.PlannedUnits
		out.StepUnits += d.StepUnits
		out.UnitsDone += d.UnitsDone
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (uc *manufacturingUseCase) DailyReport(ctx context.Context, from, to string) (*DailyReportDTO, error) {
	from, to, err := reportRange(from, to)
	if err != nil {
		return nil, err
	}
	orders, _, err := uc.loadReportOrders(ctx)
	if err != nil {
		return nil, err
	}
	logs, err := uc.repo.ListStepLogs(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load process logs")
	}
	batches, err := uc.repo.ListBatchesBetween(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load production batches")
	}
	return buildDailyReport(from, to, orders, logs, batches), nil
}

// Order summary (Laporan Ringkasan Order Produksi)

type OrderSummaryRow struct {
	OrderID       uuid.UUID `json:"orderId"`
	Number        string    `json:"orderNumber"`
	Product       string    `json:"product"`
	Status        string    `json:"status"`
	PlannedDate   string    `json:"plannedDate"`
	Planned       int       `json:"quantityToProduce"`
	Completed     int       `json:"quantityCompleted"`
	ProgressPct   float64   `json:"progressPct"`
	CompletedDate string    `json:"actualCompletionDate,omitempty"`
	// LateDays is how many days after the planned date a completed order finished (0 = on time or early).
	LateDays int `json:"lateDays"`
}

type OrderSummaryProduct struct {
	Product     string  `json:"product"`
	SKU         string  `json:"sku"`
	Orders      int     `json:"orders"`
	Planned     int     `json:"planned"`
	Completed   int     `json:"completed"`
	CompletePct float64 `json:"completionPct"`
}

type OrderSummaryDTO struct {
	From        string                `json:"from"`
	To          string                `json:"to"`
	Orders      []OrderSummaryRow     `json:"orders"`
	ByStatus    map[string]int        `json:"byStatus"`
	ByProduct   []OrderSummaryProduct `json:"byProduct"`
	Planned     int                   `json:"plannedUnits"`
	Completed   int                   `json:"completedUnits"`
	CompletePct float64               `json:"completionPct"`
	// OnTime counts completed orders that finished on or before their planned date, out of those that had one.
	OnTime       int     `json:"onTimeOrders"`
	CompletedSet int     `json:"completedOrders"`
	OnTimePct    float64 `json:"onTimePct"`
}

func daysBetween(from, to string) int {
	f, err1 := time.Parse("2006-01-02", from)
	t, err2 := time.Parse("2006-01-02", to)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(t.Sub(f).Hours() / 24)
}

func buildOrderSummary(from, to string, orders []reportOrder) *OrderSummaryDTO {
	out := &OrderSummaryDTO{From: from, To: to, Orders: []OrderSummaryRow{}, ByStatus: map[string]int{}, ByProduct: []OrderSummaryProduct{}}
	byProduct := map[string]*OrderSummaryProduct{}
	var onTimeBase int
	for _, o := range orders {
		if d := o.refDate(); d < from || d > to {
			continue
		}
		row := OrderSummaryRow{OrderID: o.ID, Number: o.Number, Product: o.ProductName, Status: o.Status, PlannedDate: o.PlannedDate,
			Planned: o.QuantityToProduce, Completed: o.QuantityCompleted, ProgressPct: pct(float64(o.QuantityCompleted), float64(o.QuantityToProduce)), CompletedDate: o.ActualCompletionDate}
		if o.Status == domain.OrderCompleted && validDay(o.PlannedDate) && validDay(o.ActualCompletionDate) {
			onTimeBase++
			if late := daysBetween(o.PlannedDate, o.ActualCompletionDate); late > 0 {
				row.LateDays = late
			} else {
				out.OnTime++
			}
		}
		out.Orders = append(out.Orders, row)
		out.ByStatus[o.Status]++
		if o.Status == domain.OrderCompleted {
			out.CompletedSet++
		}
		if o.Status != domain.OrderCancelled {
			out.Planned += o.QuantityToProduce
			out.Completed += o.QuantityCompleted
			p := byProduct[o.ProductName+"|"+o.ProductSKU]
			if p == nil {
				p = &OrderSummaryProduct{Product: o.ProductName, SKU: o.ProductSKU}
				byProduct[o.ProductName+"|"+o.ProductSKU] = p
			}
			p.Orders++
			p.Planned += o.QuantityToProduce
			p.Completed += o.QuantityCompleted
		}
	}
	for _, p := range byProduct {
		p.CompletePct = pct(float64(p.Completed), float64(p.Planned))
		out.ByProduct = append(out.ByProduct, *p)
	}
	sort.Slice(out.ByProduct, func(i, j int) bool { return out.ByProduct[i].Planned > out.ByProduct[j].Planned })
	sort.Slice(out.Orders, func(i, j int) bool { return out.Orders[i].PlannedDate < out.Orders[j].PlannedDate })
	out.CompletePct = pct(float64(out.Completed), float64(out.Planned))
	out.OnTimePct = pct(float64(out.OnTime), float64(onTimeBase))
	return out
}

func (uc *manufacturingUseCase) OrderSummary(ctx context.Context, from, to string) (*OrderSummaryDTO, error) {
	from, to, err := reportRange(from, to)
	if err != nil {
		return nil, err
	}
	orders, _, err := uc.loadReportOrders(ctx)
	if err != nil {
		return nil, err
	}
	return buildOrderSummary(from, to, orders), nil
}

// Process summary (Laporan Ringkasan per Proses)

type ProcessSummaryRow struct {
	Process         string  `json:"process"`
	Units           int     `json:"units"`
	Orders          int     `json:"orders"`
	ActiveDays      int     `json:"activeDays"`
	AvgPerDay       float64 `json:"avgUnitsPerDay"`
	StandardMinutes float64 `json:"standardMinutes"`
	// Waiting is how many units are queued at this process right now on running orders.
	Waiting int `json:"waitingUnits"`
}

type ProcessSummaryDTO struct {
	From string              `json:"from"`
	To   string              `json:"to"`
	Rows []ProcessSummaryRow `json:"rows"`
	Note string              `json:"note"`
}

func buildProcessSummary(from, to string, orders []reportOrder, logs []domain.ProductionStepLog) *ProcessSummaryDTO {
	type stepRef struct {
		name string
		std  float64
	}
	steps := map[uuid.UUID]stepRef{}
	for _, o := range orders {
		for _, s := range o.Steps {
			steps[s.ID] = stepRef{name: s.Name, std: s.StandardMinutes}
		}
	}
	type agg struct {
		row    ProcessSummaryRow
		orders map[uuid.UUID]bool
		days   map[string]bool
	}
	byName := map[string]*agg{}
	get := func(name string) *agg {
		k := strings.ToLower(strings.TrimSpace(name))
		a := byName[k]
		if a == nil {
			a = &agg{row: ProcessSummaryRow{Process: strings.TrimSpace(name)}, orders: map[uuid.UUID]bool{}, days: map[string]bool{}}
			byName[k] = a
		}
		return a
	}
	for _, l := range logs {
		ref, ok := steps[l.StepID]
		if !ok {
			continue
		}
		a := get(ref.name)
		a.row.Units += l.Quantity
		a.row.StandardMinutes += float64(l.Quantity) * ref.std
		a.orders[l.OrderID] = true
		a.days[l.Date] = true
	}
	for _, o := range orders {
		if !o.active() {
			continue
		}
		for i, s := range o.Steps {
			prev := o.QuantityToProduce
			if i > 0 {
				prev = o.Steps[i-1].QuantityDone
			}
			if w := prev - s.QuantityDone; w > 0 {
				get(s.Name).row.Waiting += w
			}
		}
	}
	out := &ProcessSummaryDTO{From: from, To: to, Rows: []ProcessSummaryRow{},
		Note: "Units are the output logged on each process in the period. Standard minutes is units times the standard time per unit set on the BOM routing. Waiting is the current queue on released, in-progress and paused orders."}
	for _, a := range byName {
		a.row.Orders, a.row.ActiveDays = len(a.orders), len(a.days)
		a.row.StandardMinutes = round2(a.row.StandardMinutes)
		if a.row.ActiveDays > 0 {
			a.row.AvgPerDay = round2(float64(a.row.Units) / float64(a.row.ActiveDays))
		}
		out.Rows = append(out.Rows, a.row)
	}
	sort.Slice(out.Rows, func(i, j int) bool {
		if out.Rows[i].Units != out.Rows[j].Units {
			return out.Rows[i].Units > out.Rows[j].Units
		}
		return out.Rows[i].Process < out.Rows[j].Process
	})
	return out
}

func (uc *manufacturingUseCase) ProcessSummary(ctx context.Context, from, to string) (*ProcessSummaryDTO, error) {
	from, to, err := reportRange(from, to)
	if err != nil {
		return nil, err
	}
	orders, _, err := uc.loadReportOrders(ctx)
	if err != nil {
		return nil, err
	}
	logs, err := uc.repo.ListStepLogs(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load process logs")
	}
	return buildProcessSummary(from, to, orders, logs), nil
}

// Work in process (Laporan Work In Process)

type WIPStepLine struct {
	Name    string `json:"name"`
	Done    int    `json:"done"`
	Waiting int    `json:"waiting"`
}

type WIPOrderRow struct {
	OrderID      uuid.UUID     `json:"orderId"`
	Number       string        `json:"orderNumber"`
	Product      string        `json:"product"`
	Status       string        `json:"status"`
	MaterialMode string        `json:"materialMode"`
	Planned      int           `json:"quantityToProduce"`
	Completed    int           `json:"quantityCompleted"`
	Steps        []WIPStepLine `json:"steps"`
	// UnitsInWIP are units that finished the first step but are not finished goods yet.
	UnitsInWIP          int     `json:"unitsInWip"`
	MaterialCostPerUnit float64 `json:"materialCostPerUnit"`
	// IssuedValue is the material issued to the order (issued mode only).
	IssuedValue float64 `json:"issuedValue"`
	WIPValue    float64 `json:"wipValue"`
}

type WIPReportDTO struct {
	Orders     []WIPOrderRow `json:"orders"`
	TotalUnits int           `json:"totalUnitsInWip"`
	TotalValue float64       `json:"totalWipValue"`
	// LedgerBalance is the Work in Process account (1250) in the general ledger, when available.
	LedgerBalance *float64 `json:"ledgerBalance,omitempty"`
	Note          string   `json:"note"`
}

type wipInputs struct {
	costPerUnit map[uuid.UUID]float64 // by BOM
	issued      map[uuid.UUID]float64 // by order: value of materials issued
}

func buildWIPReport(orders []reportOrder, in wipInputs) *WIPReportDTO {
	out := &WIPReportDTO{Orders: []WIPOrderRow{}, Note: "Only material cost is valued; labour and overhead are not. For orders whose materials are issued beforehand the value is what was issued minus the material cost of the units already completed. For other orders it is an estimate: units past the first step times the BOM material cost per unit. Orders with no routing steps show no units."}
	for _, o := range orders {
		if !o.active() {
			continue
		}
		cost := in.costPerUnit[o.BOMID]
		row := WIPOrderRow{OrderID: o.ID, Number: o.Number, Product: o.ProductName, Status: o.Status, MaterialMode: o.MaterialMode, Planned: o.QuantityToProduce,
			Completed: o.QuantityCompleted, Steps: []WIPStepLine{}, MaterialCostPerUnit: round2(cost)}
		if row.MaterialMode == "" {
			row.MaterialMode = domain.MaterialBackflush
		}
		for i, s := range o.Steps {
			prev := o.QuantityToProduce
			if i > 0 {
				prev = o.Steps[i-1].QuantityDone
			}
			row.Steps = append(row.Steps, WIPStepLine{Name: s.Name, Done: s.QuantityDone, Waiting: max(prev-s.QuantityDone, 0)})
		}
		if len(o.Steps) > 0 {
			row.UnitsInWIP = max(o.Steps[0].QuantityDone-o.QuantityCompleted, 0)
		}
		if row.MaterialMode == domain.MaterialIssued {
			row.IssuedValue = round2(in.issued[o.ID])
			row.WIPValue = round2(math.Max(row.IssuedValue-float64(o.QuantityCompleted)*cost, 0))
		} else {
			row.WIPValue = round2(float64(row.UnitsInWIP) * cost)
		}
		if row.UnitsInWIP == 0 && row.WIPValue == 0 && row.IssuedValue == 0 {
			continue
		}
		out.Orders = append(out.Orders, row)
		out.TotalUnits += row.UnitsInWIP
		out.TotalValue += row.WIPValue
	}
	out.TotalValue = round2(out.TotalValue)
	sort.Slice(out.Orders, func(i, j int) bool { return out.Orders[i].WIPValue > out.Orders[j].WIPValue })
	return out
}

func (uc *manufacturingUseCase) WIPReport(ctx context.Context) (*WIPReportDTO, error) {
	orders, boms, err := uc.loadReportOrders(ctx)
	if err != nil {
		return nil, err
	}
	in := wipInputs{costPerUnit: map[uuid.UUID]float64{}, issued: map[uuid.UUID]float64{}}
	for id, b := range boms {
		if b != nil {
			in.costPerUnit[id] = uc.materialCostPerUnit(ctx, b)
		}
	}
	for _, o := range orders {
		if !o.active() || o.MaterialMode != domain.MaterialIssued {
			continue
		}
		issued, err := uc.inventoryRepo.SumIssuedByOrder(ctx, o.ID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to load issued materials")
		}
		for pid, qty := range issued {
			if p, err := uc.productRepo.GetByID(ctx, pid); err == nil && p != nil {
				in.issued[o.ID] += float64(qty) * p.CostPrice
			}
		}
	}
	out := buildWIPReport(orders, in)
	if uc.wip != nil {
		if bal, err := uc.wip.WIPBalance(ctx); err == nil {
			bal = round2(bal)
			out.LedgerBalance = &bal
		}
	}
	return out, nil
}
