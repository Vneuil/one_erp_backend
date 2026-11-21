package infrastructure

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/inventory/application"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type inboundReader struct {
	db *gorm.DB
}

func NewInboundReader(db *gorm.DB) application.InboundReader {
	return &inboundReader{db: db}
}

func tolerateMissingTable(err error) error {
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		return nil
	}
	return err
}

func (r *inboundReader) ReadLBMContext(ctx context.Context, goodsReceiptID uuid.UUID) (*application.LBMInboundRaw, error) {
	type grRow struct {
		ID               uuid.UUID  `gorm:"column:id"`
		ReceiptNo        string     `gorm:"column:receipt_no"`
		PurchaseOrderID  uuid.UUID  `gorm:"column:purchase_order_id"`
		ReceivedDate     string     `gorm:"column:received_date"`
		WarehouseID      *uuid.UUID `gorm:"column:warehouse_id"`
		ReceivedBy       string     `gorm:"column:received_by"`
		Status           string     `gorm:"column:status"`
		DeliveryNoteNo   string     `gorm:"column:delivery_note_no"`
		SugarCondition   string     `gorm:"column:sugar_condition"`
		PackageCondition string     `gorm:"column:package_condition"`
		Notes            string     `gorm:"column:notes"`
		OrderNo          string     `gorm:"column:order_no"`
		SupplierID       *uuid.UUID `gorm:"column:supplier_id"`
		SupplierName     string     `gorm:"column:supplier_name"`
	}
	var gr grRow
	err := r.db.WithContext(ctx).Table("procurement_goods_receipts g").
		Select("g.id, g.receipt_no, g.purchase_order_id, g.received_date, g.warehouse_id, g.received_by, g.status, g.delivery_note_no, g.sugar_condition, g.package_condition, g.notes, p.order_no, p.supplier_id, p.supplier_name").
		Joins("LEFT JOIN procurement_purchase_orders p ON p.id = g.purchase_order_id").
		Where("g.id = ? AND g.deleted_at IS NULL", goodsReceiptID).
		Scan(&gr).Error
	if err != nil {
		return nil, tolerateMissingTable(err)
	}
	if gr.ID == uuid.Nil {
		return nil, nil
	}

	// Read GR lines + bag counts + product metadata
	type lineRow struct {
		ID               uuid.UUID `gorm:"column:id"`
		ProductID        uuid.UUID `gorm:"column:product_id"`
		QuantityReceived float64   `gorm:"column:quantity_received"`
		WeightGross      *float64  `gorm:"column:weight_gross"`
		WeightTare       *float64  `gorm:"column:weight_tare"`
		WeightNet        *float64  `gorm:"column:weight_net"`
		LotNumber        string    `gorm:"column:lot_number"`
		PackageSize      float64   `gorm:"column:package_size"`
		ProductSKU       string    `gorm:"column:sku"`
		ProductName      string    `gorm:"column:name"`
		ProductUnit      string    `gorm:"column:unit"`
		OrganicType      string    `gorm:"column:organic_type"`
		Certifications   string    `gorm:"column:certifications"`
		BagCount         int       `gorm:"column:bag_count"`
	}
	var lineRows []lineRow
	err = r.db.WithContext(ctx).Table("procurement_goods_receipt_lines l").
		Select(`l.id, l.product_id, l.quantity_received, l.weight_gross, l.weight_tare, l.weight_net,
			l.lot_number, l.package_size,
			COALESCE(pr.sku, '') AS sku, COALESCE(pr.name, '') AS name, COALESCE(pr.unit, 'KG') AS unit,
			COALESCE(pr.organic_type, '') AS organic_type, COALESCE(pr.certifications, '') AS certifications,
			(SELECT COUNT(1) FROM procurement_goods_receipt_bag_entries be WHERE be.goods_receipt_line_id = l.id AND be.deleted_at IS NULL) AS bag_count`).
		Joins("LEFT JOIN products pr ON pr.id = l.product_id").
		Where("l.goods_receipt_id = ? AND l.deleted_at IS NULL", goodsReceiptID).
		Scan(&lineRows).Error
	if err := tolerateMissingTable(err); err != nil {
		return nil, err
	}

	out := &application.LBMInboundRaw{
		GoodsReceiptID:   gr.ID,
		LBMNo:            gr.ReceiptNo,
		PONo:             gr.OrderNo,
		ReceivedDate:     gr.ReceivedDate,
		WarehouseID:      gr.WarehouseID,
		ReceivedBy:       gr.ReceivedBy,
		Status:           gr.Status,
		DeliveryNoteNo:   gr.DeliveryNoteNo,
		SupplierID:       gr.SupplierID,
		SupplierName:     gr.SupplierName,
		SugarCondition:   gr.SugarCondition,
		PackageCondition: gr.PackageCondition,
		Notes:            gr.Notes,
	}

	for _, l := range lineRows {
		net := l.QuantityReceived
		if l.WeightNet != nil && *l.WeightNet > 0 {
			net = *l.WeightNet
		}
		out.Lines = append(out.Lines, application.LBMInboundLineRaw{
			LineID:         l.ID,
			ProductID:      l.ProductID,
			ProductSKU:     l.ProductSKU,
			ProductName:    l.ProductName,
			UoM:            l.ProductUnit,
			OrganicType:    l.OrganicType,
			Certifications: l.Certifications,
			LotNumber:      l.LotNumber,
			WeightGross:    l.WeightGross,
			WeightTare:     l.WeightTare,
			WeightNet:      net,
			PackageSize:    l.PackageSize,
			BagCount:       l.BagCount,
		})
	}

	// Read latest CAS (QC-007) linked to this goodsReceiptId or lbmNo
	var casRow struct {
		ID            uuid.UUID `gorm:"column:id"`
		CheckDate     string    `gorm:"column:check_date"`
		DriverName    string    `gorm:"column:driver_name"`
		LicensePlate  string    `gorm:"column:license_plate"`
		VehicleType   string    `gorm:"column:vehicle_type"`
		OverallStatus string    `gorm:"column:overall_status"`
		QCName        string    `gorm:"column:qc_name"`
		QAName        string    `gorm:"column:qa_name"`
	}
	err = r.db.WithContext(ctx).Table("qc_supplier_armada_checks").
		Select("id, check_date, driver_name, license_plate, vehicle_type, overall_status, qc_name, qa_name").
		Where("deleted_at IS NULL AND (goods_receipt_id = ? OR (lbm_no <> '' AND UPPER(lbm_no) = UPPER(?)))", goodsReceiptID, gr.ReceiptNo).
		Order("created_at desc").Limit(1).Scan(&casRow).Error
	if err := tolerateMissingTable(err); err != nil {
		return nil, err
	}
	if casRow.ID != uuid.Nil {
		out.CAS = &application.QCCASSummaryDTO{
			ID:            casRow.ID,
			CheckDate:     casRow.CheckDate,
			DriverName:    casRow.DriverName,
			LicensePlate:  casRow.LicensePlate,
			VehicleType:   casRow.VehicleType,
			OverallStatus: casRow.OverallStatus,
			QCName:        casRow.QCName,
			QAName:        casRow.QAName,
		}
	}

	// Read latest PBD (QC-001) linked to this goodsReceiptId or lbmNo
	var pbdRow struct {
		ID                 uuid.UUID `gorm:"column:id"`
		CheckDate          string    `gorm:"column:check_date"`
		ProductName        string    `gorm:"column:product_name"`
		BatchNo            string    `gorm:"column:batch_no"`
		TotalBags          int       `gorm:"column:total_bags"`
		NetWeight          float64   `gorm:"column:net_weight"`
		AcceptedWeight     float64   `gorm:"column:accepted_weight"`
		RejectedWeight     float64   `gorm:"column:rejected_weight"`
		SugarCondition     string    `gorm:"column:sugar_condition"`
		PackagingCondition string    `gorm:"column:packaging_condition"`
		MoistureContent    *float64  `gorm:"column:moisture_content"`
		BrixValue          *float64  `gorm:"column:brix_value"`
		MeshSize           string    `gorm:"column:mesh_size"`
		RecommendedAction  string    `gorm:"column:recommended_action"`
		OverallStatus      string    `gorm:"column:overall_status"`
		QCName             string    `gorm:"column:qc_name"`
		QAName             string    `gorm:"column:qa_name"`
	}
	err = r.db.WithContext(ctx).Table("qc_raw_material_incoming_checks").
		Select("id, check_date, product_name, batch_no, total_bags, net_weight, accepted_weight, rejected_weight, sugar_condition, packaging_condition, moisture_content, brix_value, mesh_size, recommended_action, overall_status, qc_name, qa_name").
		Where("deleted_at IS NULL AND (goods_receipt_id = ? OR (lbm_no <> '' AND UPPER(lbm_no) = UPPER(?)))", goodsReceiptID, gr.ReceiptNo).
		Order("created_at desc").Limit(1).Scan(&pbdRow).Error
	if err := tolerateMissingTable(err); err != nil {
		return nil, err
	}
	if pbdRow.ID != uuid.Nil {
		out.PBD = &application.QCPBDSummaryDTO{
			ID:                 pbdRow.ID,
			CheckDate:          pbdRow.CheckDate,
			ProductName:        pbdRow.ProductName,
			BatchNo:            pbdRow.BatchNo,
			TotalBags:          pbdRow.TotalBags,
			NetWeight:          pbdRow.NetWeight,
			AcceptedWeight:     pbdRow.AcceptedWeight,
			RejectedWeight:     pbdRow.RejectedWeight,
			SugarCondition:     pbdRow.SugarCondition,
			PackagingCondition: pbdRow.PackagingCondition,
			MoistureContent:    pbdRow.MoistureContent,
			BrixValue:          pbdRow.BrixValue,
			MeshSize:           pbdRow.MeshSize,
			RecommendedAction:  pbdRow.RecommendedAction,
			OverallStatus:      pbdRow.OverallStatus,
			QCName:             pbdRow.QCName,
			QAName:             pbdRow.QAName,
		}
	}

	// Read latest PBK (QC-003) linked to this goodsReceiptId or lbmNo
	var pbkRow struct {
		ID            uuid.UUID `gorm:"column:id"`
		CheckDate     string    `gorm:"column:check_date"`
		ItemName      string    `gorm:"column:item_name"`
		MaterialType  string    `gorm:"column:material_type"`
		TotalQty      int       `gorm:"column:total_qty"`
		PassedQty     int       `gorm:"column:passed_qty"`
		FailedQty     int       `gorm:"column:failed_qty"`
		OverallStatus string    `gorm:"column:overall_status"`
		QCName        string    `gorm:"column:qc_name"`
		QAName        string    `gorm:"column:qa_name"`
	}
	err = r.db.WithContext(ctx).Table("qc_packaging_quality_checks").
		Select("id, check_date, item_name, material_type, total_qty, passed_qty, failed_qty, overall_status, qc_name, qa_name").
		Where("deleted_at IS NULL AND (goods_receipt_id = ? OR (lbm_no <> '' AND UPPER(lbm_no) = UPPER(?)))", goodsReceiptID, gr.ReceiptNo).
		Order("created_at desc").Limit(1).Scan(&pbkRow).Error
	if err := tolerateMissingTable(err); err != nil {
		return nil, err
	}
	if pbkRow.ID != uuid.Nil {
		out.PBK = &application.QCPBKSummaryDTO{
			ID:            pbkRow.ID,
			CheckDate:     pbkRow.CheckDate,
			ItemName:      pbkRow.ItemName,
			MaterialType:  pbkRow.MaterialType,
			TotalQty:      pbkRow.TotalQty,
			PassedQty:     pbkRow.PassedQty,
			FailedQty:     pbkRow.FailedQty,
			OverallStatus: pbkRow.OverallStatus,
			QCName:        pbkRow.QCName,
			QAName:        pbkRow.QAName,
		}
	}

	return out, nil
}
