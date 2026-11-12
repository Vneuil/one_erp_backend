package infrastructure

import (
	"context"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/certification/application"
	"github.com/divinecoid/one-backend/internal/modules/certification/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type traceReader struct{ db *gorm.DB }

func NewTraceReader(db *gorm.DB) application.TraceReader { return &traceReader{db: db} }

// tolerateMissing returns nil rows when a module's table was never migrated for this tenant.
func tolerateMissing(err error) error {
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		return nil
	}
	return err
}

func (r *traceReader) Receipts(ctx context.Context, lot string) ([]application.TraceReceipt, error) {
	var out []application.TraceReceipt
	err := r.db.WithContext(ctx).Table("procurement_goods_receipt_lines AS l").
		Select("g.receipt_no, g.received_date, p.order_no, p.supplier_id, p.supplier_name, l.product_id, l.quantity_received AS quantity, l.lot_number").
		Joins("JOIN procurement_goods_receipts g ON g.id = l.goods_receipt_id").
		Joins("LEFT JOIN procurement_purchase_orders p ON p.id = g.purchase_order_id").
		Where("UPPER(l.lot_number) = ?", lot).Scan(&out).Error
	return out, tolerateMissing(err)
}

func (r *traceReader) Inspections(ctx context.Context, batchNos []string) ([]application.TraceInspection, error) {
	type row struct {
		InspectionNo, BatchNo, TargetType, OverallStatus string
		InspectedAt                                      time.Time
		IsVoided                                         bool
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("qc_inspections").
		Select("inspection_no, batch_no, target_type, overall_status, inspected_at, is_voided").
		Where("batch_no IN ?", batchNos).Order("inspected_at asc").Scan(&rows).Error
	if err := tolerateMissing(err); err != nil {
		return nil, err
	}
	out := make([]application.TraceInspection, len(rows))
	for i, x := range rows {
		out[i] = application.TraceInspection{InspectionNo: x.InspectionNo, BatchNo: x.BatchNo, TargetType: x.TargetType, OverallStatus: x.OverallStatus, InspectedAt: x.InspectedAt.Format("2006-01-02 15:04"), IsVoided: x.IsVoided}
	}
	return out, nil
}

func (r *traceReader) Usage(ctx context.Context, lot string) ([]application.TraceUsage, error) {
	type row struct {
		OrderNumber, ChildBatchNo, ChildProductName string
		QuantityUsed                                float64
		RecordedAt                                  time.Time
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("manufacturing_batch_genealogy").
		Select("order_number, child_batch_no, child_product_name, quantity_used, recorded_at").
		Where("UPPER(parent_batch_no) = ?", lot).Order("recorded_at asc").Scan(&rows).Error
	if err := tolerateMissing(err); err != nil {
		return nil, err
	}
	out := make([]application.TraceUsage, len(rows))
	for i, x := range rows {
		out[i] = application.TraceUsage{OrderNumber: x.OrderNumber, ChildBatchNo: x.ChildBatchNo, ChildProduct: x.ChildProductName, QuantityUsed: x.QuantityUsed, RecordedAt: x.RecordedAt.Format("2006-01-02")}
	}
	return out, nil
}

func (r *traceReader) SupplierCerts(ctx context.Context, ids []uuid.UUID) ([]application.CertificateResponse, error) {
	var certs []domain.Certificate
	if err := r.db.WithContext(ctx).Where("supplier_id IN ?", ids).Order("valid_until asc").Find(&certs).Error; err != nil {
		return nil, tolerateMissing(err)
	}
	today := time.Now().Format("2006-01-02")
	out := make([]application.CertificateResponse, len(certs))
	for i, c := range certs {
		out[i] = application.CertificateResponse{Certificate: c, EffectiveStatus: c.EffectiveStatus(today)}
	}
	return out, nil
}
