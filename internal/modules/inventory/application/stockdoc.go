package application

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// docKind describes how one stock document type behaves.
type docKind struct {
	prefix string
	label  string
	// sign is +1 when the document brings stock in and -1 when it takes stock out.
	sign        int
	needsReason bool
}

var docKinds = map[string]docKind{
	domain.DocMaterialIssue: {"PBH", "Pengambilan bahan", -1, false},
	domain.DocFinishedGoods: {"PBJ", "Penerimaan barang jadi", +1, false},
	domain.DocScrap:         {"SCR", "Scrap", -1, true},
	domain.DocMemoIn:        {"MMI", "Memo stock masuk", +1, true},
	domain.DocMemoOut:       {"MMO", "Memo stock keluar", -1, true},
}

const stockDocSourcePrefix = "stock-doc:"

type StockDocumentLineInput struct {
	ProductID uuid.UUID `json:"productId"`
	Quantity  int       `json:"quantity"`
	// UnitCost overrides the product's cost price; only allowed on a finished goods receipt (production cost).
	UnitCost   *float64 `json:"unitCost,omitempty"`
	BatchNo    string   `json:"batchNo,omitempty"`
	ExpiryDate string   `json:"expiryDate,omitempty"`
}

type CreateStockDocumentDTO struct {
	Type        string    `json:"type"`
	Date        string    `json:"date"`
	WarehouseID uuid.UUID `json:"warehouseId"`
	Reference   string    `json:"reference"`
	Reason      string    `json:"reason"`
	Notes       string    `json:"notes"`
	// ProductionOrderID ties a material issue to the production order it supplies.
	ProductionOrderID *uuid.UUID               `json:"productionOrderId,omitempty"`
	Lines             []StockDocumentLineInput `json:"lines"`
}

type StockDocumentLineDTO struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"productId"`
	ProductSKU  string    `json:"productSku"`
	ProductName string    `json:"productName"`
	Quantity    int       `json:"quantity"`
	UnitCost    float64   `json:"unitCost"`
	Amount      float64   `json:"amount"`
	BatchNo     string    `json:"batchNo,omitempty"`
	ExpiryDate  string    `json:"expiryDate,omitempty"`
}

type StockDocumentResponseDTO struct {
	ID                uuid.UUID              `json:"id"`
	Number            string                 `json:"number"`
	Type              string                 `json:"type"`
	TypeLabel         string                 `json:"typeLabel"`
	Direction         string                 `json:"direction"` // in | out
	Date              string                 `json:"date"`
	WarehouseID       uuid.UUID              `json:"warehouseId"`
	WarehouseName     string                 `json:"warehouseName"`
	Reference         string                 `json:"reference"`
	ProductionOrderID *uuid.UUID             `json:"productionOrderId,omitempty"`
	Reason            string                 `json:"reason"`
	Notes             string                 `json:"notes"`
	TotalValue        float64                `json:"totalValue"`
	Posted            bool                   `json:"posted"`
	CreatedBy         string                 `json:"createdBy,omitempty"`
	Lines             []StockDocumentLineDTO `json:"lines"`
	CreatedAt         time.Time              `json:"createdAt"`
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// StockDocumentLedgerEntry is the journal of a stock document, valued at cost:
//
//	issue to production:  Dr Work in Process      / Cr Inventory
//	finished goods in:    Dr Inventory            / Cr Work in Process
//	scrap:                Dr Scrap Loss           / Cr Inventory
//	memo in:              Dr Inventory            / Cr Inventory Adjustments
//	memo out:             Dr Inventory Adjustments / Cr Inventory
func StockDocumentLedgerEntry(d *domain.StockDocument) financeApp.LedgerEntry {
	var debit, credit string
	switch d.Type {
	case domain.DocMaterialIssue:
		debit, credit = financeApp.AccountWIP, financeApp.AccountInventory
	case domain.DocFinishedGoods:
		debit, credit = financeApp.AccountInventory, financeApp.AccountWIP
	case domain.DocScrap:
		debit, credit = financeApp.AccountScrapLoss, financeApp.AccountInventory
	case domain.DocMemoIn:
		debit, credit = financeApp.AccountInventory, financeApp.AccountInventoryAdjustment
	default:
		debit, credit = financeApp.AccountInventoryAdjustment, financeApp.AccountInventory
	}
	return financeApp.LedgerEntry{
		SourceDoc: stockDocSourcePrefix + d.ID.String(),
		Memo:      docKinds[d.Type].label + " " + d.Number,
		Lines: []financeApp.LedgerLine{
			{AccountCode: debit, Debit: d.TotalValue, Description: d.Number},
			{AccountCode: credit, Credit: d.TotalValue, Description: d.Number},
		},
	}
}

func (uc *inventoryUseCase) nextDocNumber(ctx context.Context, docType, date string) (string, error) {
	prefix := fmt.Sprintf("%s-%s-", docKinds[docType].prefix, strings.ReplaceAll(date[:7], "-", ""))
	n, err := uc.repo.CountStockDocuments(ctx, docType, prefix)
	if err != nil {
		return "", apperrors.NewInternal(err, "Failed to number the document")
	}
	return fmt.Sprintf("%s%04d", prefix, n+1), nil
}

// CreateStockDocument validates, moves the stock of every line, saves the
// document and posts its value. Stock levels are checked up front for
// documents that take stock out, and if anything fails after some lines have
// moved those lines are put back, so a document is all or nothing.
func (uc *inventoryUseCase) CreateStockDocument(ctx context.Context, dto CreateStockDocumentDTO) (*StockDocumentResponseDTO, error) {
	kind, ok := docKinds[dto.Type]
	if !ok {
		return nil, apperrors.NewBadRequest("type must be one of material_issue, finished_goods_receipt, scrap, memo_in, memo_out")
	}
	if dto.ProductionOrderID != nil && dto.Type != domain.DocMaterialIssue {
		return nil, apperrors.NewBadRequest("Only a material issue can be tied to a production order")
	}
	reason := strings.TrimSpace(dto.Reason)
	if kind.needsReason && reason == "" {
		return nil, apperrors.NewBadRequest("A reason is required for " + strings.ToLower(kind.label))
	}
	if len(dto.Lines) == 0 || len(dto.Lines) > 200 {
		return nil, apperrors.NewBadRequest("A document needs between 1 and 200 lines")
	}
	date := dto.Date
	if date == "" {
		date = today()
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, apperrors.NewBadRequest("date must be YYYY-MM-DD")
	}
	if date > today() {
		return nil, apperrors.NewBadRequest("date cannot be in the future")
	}
	wh, err := uc.repo.GetWarehouseByID(ctx, dto.WarehouseID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get warehouse")
	}
	if wh == nil {
		return nil, apperrors.NewNotFound("Warehouse not found")
	}

	lines := make([]domain.StockDocumentLine, 0, len(dto.Lines))
	need := map[uuid.UUID]int{}
	var total float64
	for i, in := range dto.Lines {
		if in.ProductID == uuid.Nil || in.Quantity <= 0 {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Line %d: a product and a positive quantity are required", i+1))
		}
		p, err := uc.productRepo.GetByID(ctx, in.ProductID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get product")
		}
		if p == nil {
			return nil, apperrors.NewNotFound(fmt.Sprintf("Line %d: product not found", i+1))
		}
		cost := p.CostPrice
		if in.UnitCost != nil {
			if dto.Type != domain.DocFinishedGoods {
				return nil, apperrors.NewBadRequest(fmt.Sprintf("Line %d: a unit cost can only be given on a finished goods receipt", i+1))
			}
			if *in.UnitCost < 0 {
				return nil, apperrors.NewBadRequest(fmt.Sprintf("Line %d: the unit cost cannot be negative", i+1))
			}
			cost = *in.UnitCost
		}
		batchNo, expiry := strings.TrimSpace(in.BatchNo), strings.TrimSpace(in.ExpiryDate)
		if kind.sign < 0 && (batchNo != "" || expiry != "") {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Line %d: a batch can only be given on documents that bring stock in", i+1))
		}
		if expiry != "" {
			if _, err := time.Parse("2006-01-02", expiry); err != nil {
				return nil, apperrors.NewBadRequest(fmt.Sprintf("Line %d: expiryDate must be YYYY-MM-DD", i+1))
			}
		}
		amount := round2(cost * float64(in.Quantity))
		total += amount
		lines = append(lines, domain.StockDocumentLine{ProductID: in.ProductID, Quantity: in.Quantity, UnitCost: cost, Amount: amount, BatchNo: batchNo, ExpiryDate: expiry})
		need[in.ProductID] += in.Quantity
	}
	if kind.sign < 0 {
		for pid, n := range need {
			avail, err := uc.GetAvailability(ctx, pid, dto.WarehouseID)
			if err != nil {
				return nil, err
			}
			if n > avail {
				sku, name := uc.productInfo(ctx, pid)
				return nil, apperrors.NewBadRequest(fmt.Sprintf("Insufficient stock of %s (%s): need %d, only %d available", name, sku, n, avail))
			}
		}
	}

	number, err := uc.nextDocNumber(ctx, dto.Type, date)
	if err != nil {
		return nil, err
	}
	by := actor.EmailFrom(ctx)
	moveReason := kind.label
	if reason != "" {
		moveReason += ": " + reason
	}

	var applied []domain.StockDocumentLine
	undo := func() {
		for _, l := range applied {
			if _, err := uc.AdjustStock(ctx, AdjustStockDTO{ProductID: l.ProductID, WarehouseID: dto.WarehouseID, Quantity: -kind.sign * l.Quantity,
				Reason: "Pembatalan otomatis " + number, Reference: number, CreatedBy: by}); err != nil {
				slog.Error("inventory: failed to roll back a stock document line", "document", number, "product", l.ProductID, "error", err)
			}
		}
	}
	for _, l := range lines {
		if _, err := uc.AdjustStock(ctx, AdjustStockDTO{ProductID: l.ProductID, WarehouseID: dto.WarehouseID, Quantity: kind.sign * l.Quantity,
			Reason: moveReason, Reference: number, CreatedBy: by, BatchNo: l.BatchNo, ExpiryDate: l.ExpiryDate}); err != nil {
			undo()
			return nil, err
		}
		applied = append(applied, l)
	}

	doc := &domain.StockDocument{Number: number, Type: dto.Type, Date: date, WarehouseID: dto.WarehouseID, Reference: strings.TrimSpace(dto.Reference), ProductionOrderID: dto.ProductionOrderID,
		Reason: reason, Notes: strings.TrimSpace(dto.Notes), TotalValue: round2(total), CreatedBy: by, Lines: lines}
	doc.ID = uuid.New()
	for i := range doc.Lines {
		doc.Lines[i].ID, doc.Lines[i].DocumentID = uuid.New(), doc.ID
	}
	if err := uc.repo.CreateStockDocument(ctx, doc); err != nil {
		undo()
		return nil, apperrors.NewInternal(err, "Failed to save the stock document; the stock movements were rolled back")
	}

	var postErr error
	if doc.TotalValue <= 0 {
		doc.Posted = true // nothing to value, so nothing to post
	} else if uc.ledger == nil {
		postErr = fmt.Errorf("no ledger configured")
	} else {
		e := StockDocumentLedgerEntry(doc)
		postErr = uc.ledger.PostEntryOn(ctx, doc.Date, e.SourceDoc, e.Memo, e.Lines)
		doc.Posted = postErr == nil
	}
	if doc.Posted {
		if err := uc.repo.UpdateStockDocument(ctx, doc); err != nil {
			postErr = err
			doc.Posted = false
		}
	}
	resp := uc.toStockDocumentResponse(ctx, doc)
	if postErr != nil {
		return resp, apperrors.NewInternal(postErr, "Stock document saved and stock moved, but its value could not be posted to the ledger")
	}
	return resp, nil
}

func (uc *inventoryUseCase) toStockDocumentResponse(ctx context.Context, d *domain.StockDocument) *StockDocumentResponseDTO {
	kind := docKinds[d.Type]
	dir := "in"
	if kind.sign < 0 {
		dir = "out"
	}
	lines := make([]StockDocumentLineDTO, len(d.Lines))
	for i, l := range d.Lines {
		sku, name := uc.productInfo(ctx, l.ProductID)
		lines[i] = StockDocumentLineDTO{ID: l.ID, ProductID: l.ProductID, ProductSKU: sku, ProductName: name, Quantity: l.Quantity,
			UnitCost: l.UnitCost, Amount: l.Amount, BatchNo: l.BatchNo, ExpiryDate: l.ExpiryDate}
	}
	return &StockDocumentResponseDTO{ID: d.ID, Number: d.Number, Type: d.Type, TypeLabel: kind.label, Direction: dir, Date: d.Date,
		WarehouseID: d.WarehouseID, WarehouseName: uc.warehouseName(ctx, d.WarehouseID), Reference: d.Reference, ProductionOrderID: d.ProductionOrderID, Reason: d.Reason, Notes: d.Notes,
		TotalValue: d.TotalValue, Posted: d.Posted, CreatedBy: d.CreatedBy, Lines: lines, CreatedAt: d.CreatedAt}
}

func (uc *inventoryUseCase) GetStockDocument(ctx context.Context, id uuid.UUID) (*StockDocumentResponseDTO, error) {
	d, err := uc.repo.GetStockDocument(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get the stock document")
	}
	if d == nil {
		return nil, apperrors.NewNotFound("Stock document not found")
	}
	return uc.toStockDocumentResponse(ctx, d), nil
}

func (uc *inventoryUseCase) ListStockDocuments(ctx context.Context, docType, from, to string) ([]StockDocumentResponseDTO, error) {
	if docType != "" {
		if _, ok := docKinds[docType]; !ok {
			return nil, apperrors.NewBadRequest("Unknown document type")
		}
	}
	for _, d := range []string{from, to} {
		if d != "" {
			if _, err := time.Parse("2006-01-02", d); err != nil {
				return nil, apperrors.NewBadRequest("from and to must be YYYY-MM-DD")
			}
		}
	}
	items, err := uc.repo.ListStockDocuments(ctx, domain.StockDocumentFilter{Type: docType, From: from, To: to})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list stock documents")
	}
	out := make([]StockDocumentResponseDTO, len(items))
	for i := range items {
		out[i] = *uc.toStockDocumentResponse(ctx, &items[i])
	}
	return out, nil
}

// Stock opname: value of the variance, and import of counted quantities.

// OpnameLedgerEntry books an opname's variances at cost: a surplus is
// Dr Inventory / Cr Inventory Adjustments and a shortage the reverse.
func OpnameLedgerEntry(id uuid.UUID, auditDate string, gain, loss float64) financeApp.LedgerEntry {
	return financeApp.LedgerEntry{
		SourceDoc: "stock-opname:" + id.String(),
		Memo:      "Stock opname " + auditDate,
		Lines: []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountInventory, Debit: round2(gain), Description: "Opname surplus"},
			{AccountCode: financeApp.AccountInventoryAdjustment, Credit: round2(gain), Description: "Opname surplus"},
			{AccountCode: financeApp.AccountInventoryAdjustment, Debit: round2(loss), Description: "Opname shortage"},
			{AccountCode: financeApp.AccountInventory, Credit: round2(loss), Description: "Opname shortage"},
		},
	}
}

// postOpnameVariance values the counted variances at the product cost price
// and posts them. A failure is logged and does not undo the reconciliation.
func (uc *inventoryUseCase) postOpnameVariance(ctx context.Context, o *domain.StockOpname) {
	if uc.ledger == nil {
		return
	}
	var gain, loss float64
	for _, l := range o.Lines {
		if !l.Counted || l.Variance() == 0 {
			continue
		}
		p, err := uc.productRepo.GetByID(ctx, l.ProductID)
		if err != nil || p == nil || p.CostPrice <= 0 {
			continue
		}
		v := float64(l.Variance()) * p.CostPrice
		if v > 0 {
			gain += v
		} else {
			loss -= v
		}
	}
	if gain <= 0 && loss <= 0 {
		return
	}
	date := o.AuditDate
	if _, err := time.Parse("2006-01-02", date); err != nil || date > today() {
		date = today()
	}
	e := OpnameLedgerEntry(o.ID, o.AuditDate, gain, loss)
	if err := uc.ledger.PostEntryOn(ctx, date, e.SourceDoc, e.Memo, e.Lines); err != nil {
		slog.Error("inventory: failed to post opname variance", "opname", o.ID, "error", err)
	}
}

type OpnameCountRow struct {
	SKU        string `json:"sku"`
	CountedQty int    `json:"countedQty"`
}

type OpnameImportError struct {
	Row     int    `json:"row"`
	SKU     string `json:"sku"`
	Message string `json:"message"`
}

type OpnameImportResult struct {
	Applied int                 `json:"applied"`
	Errors  []OpnameImportError `json:"errors"`
	Opname  *OpnameResponseDTO  `json:"opname"`
}

// ImportOpnameCounts records counted quantities by SKU. Valid rows are applied
// and the rest reported with their row number; re-importing a corrected file
// simply overwrites the counts. A SKU repeated in the file is an error for the
// later rows.
func (uc *inventoryUseCase) ImportOpnameCounts(ctx context.Context, opnameID uuid.UUID, rows []OpnameCountRow) (*OpnameImportResult, error) {
	o, err := uc.repo.GetOpnameByID(ctx, opnameID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get stock opname")
	}
	if o == nil {
		return nil, apperrors.NewNotFound("Stock opname not found")
	}
	if o.Status == domain.OpnameCompleted {
		return nil, apperrors.NewConflict("Cannot record counts on a completed opname")
	}
	if len(rows) == 0 || len(rows) > 5000 {
		return nil, apperrors.NewBadRequest("Provide between 1 and 5000 rows")
	}
	bySKU := make(map[string]int, len(o.Lines))
	for i, l := range o.Lines {
		sku, _ := uc.productInfo(ctx, l.ProductID)
		bySKU[strings.ToLower(strings.TrimSpace(sku))] = i
	}
	out := &OpnameImportResult{Errors: []OpnameImportError{}}
	seen := map[string]bool{}
	for i, r := range rows {
		key := strings.ToLower(strings.TrimSpace(r.SKU))
		fail := func(msg string) {
			out.Errors = append(out.Errors, OpnameImportError{Row: i + 1, SKU: r.SKU, Message: msg})
		}
		switch idx, ok := bySKU[key]; {
		case key == "":
			fail("SKU is empty")
		case r.CountedQty < 0:
			fail("The counted quantity cannot be negative")
		case seen[key]:
			fail("SKU appears more than once in the file")
		case !ok:
			fail("SKU is not part of this opname")
		default:
			seen[key] = true
			o.Lines[idx].CountedQty, o.Lines[idx].Counted = r.CountedQty, true
			if err := uc.repo.UpdateOpnameLine(ctx, &o.Lines[idx]); err != nil {
				return nil, apperrors.NewInternal(err, "Failed to update opname line")
			}
			out.Applied++
		}
	}
	if out.Applied > 0 && o.Status == domain.OpnameDraft {
		o.Status = domain.OpnameInProgress
		if err := uc.repo.UpdateOpname(ctx, o); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to update stock opname status")
		}
	}
	out.Opname = uc.toOpnameResponse(ctx, o)
	return out, nil
}
