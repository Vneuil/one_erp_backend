package application

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	"github.com/divinecoid/one-backend/internal/shared/docseq"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// CreateBrontolReport creates a new Brontol Report (REP-013-KSB)
func (uc *qcUseCase) CreateBrontolReport(ctx context.Context, dto CreateBrontolReportDTO) (*domain.BrontolReport, error) {
	if dto.ReportDate == "" {
		dto.ReportDate = time.Now().Format("2006-01-02")
	}
	month := dto.ReportDate
	if len(month) >= 7 {
		month = month[:7]
	}
	reportNo, err := docseq.Next(ctx, uc.repo, "BRT", dto.ReportDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor laporan brontol")
	}

	matType := strings.ToUpper(strings.TrimSpace(dto.MaterialType))
	if matType == "" {
		matType = "BM"
	}
	unit := strings.ToUpper(strings.TrimSpace(dto.Unit))
	if unit == "" {
		unit = "KG"
	}

	report := &domain.BrontolReport{
		ReportNo:           reportNo,
		DocumentCode:       "REP-013-KSB",
		ReportDate:         dto.ReportDate,
		ReportMonth:        month,
		SourceStage:        dto.SourceStage,
		SourceInspectionID: dto.SourceInspectionID,
		SourceInspectionNo: dto.SourceInspectionNo,
		ProductionOrderID:  dto.ProductionOrderID,
		OrderNumber:        dto.OrderNumber,
		ProductID:          dto.ProductID,
		ProductName:        dto.ProductName,
		ProductCode:        dto.ProductCode,
		BatchNo:            dto.BatchNo,
		MaterialType:       matType,
		SupplierID:         dto.SupplierID,
		SupplierName:       dto.SupplierName,
		MachineNo:          dto.MachineNo,
		MeshSize:           dto.MeshSize,
		Quantity:           dto.Quantity,
		Unit:               unit,
		DefectReason:       dto.DefectReason,
		ActionDisposition:  dto.ActionDisposition,
		OperatorName:       dto.OperatorName,
		QCInspectorName:    dto.QCInspectorName,
		Status:             "recorded",
		Notes:              dto.Notes,
		IsReturn:           dto.ActionDisposition == domain.ActionReturnToSupplier || dto.CreateReturn != nil,
	}

	if err := uc.repo.CreateBrontolReport(ctx, report); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat laporan brontol")
	}

	// If marked for return to supplier, generate linked RejectReturnReport
	if report.IsReturn {
		retDTO := dto.CreateReturn
		if retDTO == nil {
			prepBy := dto.QCInspectorName
			if prepBy == "" {
				prepBy = dto.OperatorName
			}
			if prepBy == "" {
				prepBy = "Staff QC / Produksi"
			}
			retDTO = &CreateRejectReturnDTO{
				ReturnDate:     dto.ReportDate,
				SupplierID:     dto.SupplierID,
				SupplierName:   dto.SupplierName,
				ReturnQuantity: dto.Quantity,
				Unit:           unit,
				ReturnReason:   dto.DefectReason,
				PreparedBy:     prepBy,
			}
		}

		retNo, err := docseq.Next(ctx, uc.repo, "RTR", retDTO.ReturnDate)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Gagal membuat nomor laporan retur")
		}

		supplierName := retDTO.SupplierName
		if supplierName == "" {
			supplierName = dto.SupplierName
		}
		if supplierName == "" {
			supplierName = "Supplier Bahan / Kemasan"
		}

		returnReport := &domain.RejectReturnReport{
			BrontolReportID: report.ID,
			ReturnNo:        retNo,
			DocumentCode:    "RET-REJECT-01",
			ReturnDate:      retDTO.ReturnDate,
			SupplierID:      retDTO.SupplierID,
			SupplierName:    supplierName,
			PONo:            retDTO.PONo,
			GoodsReceiptNo:  retDTO.GoodsReceiptNo,
			DeliveryNoteNo:  retDTO.DeliveryNoteNo,
			ProductName:     dto.ProductName,
			ProductCode:     dto.ProductCode,
			BatchNo:         dto.BatchNo,
			ReturnQuantity:  retDTO.ReturnQuantity,
			Unit:            unit,
			UnitPrice:       retDTO.UnitPrice,
			EstimatedValue:  retDTO.EstimatedValue,
			ReturnReason:    retDTO.ReturnReason,
			CourierName:     retDTO.CourierName,
			VehicleNumber:   retDTO.VehicleNumber,
			DriverName:      retDTO.DriverName,
			PreparedBy:      retDTO.PreparedBy,
			ApprovedBy:      retDTO.ApprovedBy,
			Status:          "pending_pickup",
			Notes:           retDTO.Notes,
		}

		if err := uc.repo.CreateRejectReturnReport(ctx, returnReport); err != nil {
			slog.Warn("Failed to create linked reject return report", "error", err, "brontolId", report.ID)
		} else {
			report.ReturnReport = returnReport
		}
	}

	return report, nil
}

// GetBrontolReportByID retrieves a Brontol Report by ID
func (uc *qcUseCase) GetBrontolReportByID(ctx context.Context, id uuid.UUID) (*domain.BrontolReport, error) {
	report, err := uc.repo.GetBrontolReportByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Laporan brontol tidak ditemukan")
	}
	return report, nil
}

// ListBrontolReports retrieves reports matching filters
func (uc *qcUseCase) ListBrontolReports(ctx context.Context, month, stage, action, status, search string) ([]domain.BrontolReport, error) {
	reports, err := uc.repo.ListBrontolReports(ctx, month, stage, action, status, search)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil daftar laporan brontol")
	}
	return reports, nil
}

// UpdateBrontolReport updates an existing report
func (uc *qcUseCase) UpdateBrontolReport(ctx context.Context, id uuid.UUID, dto UpdateBrontolReportDTO) (*domain.BrontolReport, error) {
	report, err := uc.repo.GetBrontolReportByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Laporan brontol tidak ditemukan")
	}

	if dto.ReportDate != "" {
		report.ReportDate = dto.ReportDate
		if len(dto.ReportDate) >= 7 {
			report.ReportMonth = dto.ReportDate[:7]
		}
	}
	if dto.ProductName != "" {
		report.ProductName = dto.ProductName
	}
	if dto.ProductCode != "" {
		report.ProductCode = dto.ProductCode
	}
	if dto.BatchNo != "" {
		report.BatchNo = dto.BatchNo
	}
	if dto.MaterialType != "" {
		report.MaterialType = strings.ToUpper(dto.MaterialType)
	}
	if dto.SupplierID != nil {
		report.SupplierID = dto.SupplierID
	}
	if dto.SupplierName != "" {
		report.SupplierName = dto.SupplierName
	}
	if dto.MachineNo != "" {
		report.MachineNo = dto.MachineNo
	}
	if dto.MeshSize != "" {
		report.MeshSize = dto.MeshSize
	}
	if dto.Quantity != nil {
		report.Quantity = *dto.Quantity
	}
	if dto.Unit != "" {
		report.Unit = strings.ToUpper(dto.Unit)
	}
	if dto.DefectReason != "" {
		report.DefectReason = dto.DefectReason
	}
	if dto.ActionDisposition != "" {
		report.ActionDisposition = dto.ActionDisposition
		if dto.ActionDisposition == domain.ActionReturnToSupplier {
			report.IsReturn = true
		}
	}
	if dto.OperatorName != "" {
		report.OperatorName = dto.OperatorName
	}
	if dto.QCInspectorName != "" {
		report.QCInspectorName = dto.QCInspectorName
	}
	if dto.Status != "" {
		report.Status = dto.Status
	}
	if dto.Notes != "" {
		report.Notes = dto.Notes
	}

	if err := uc.repo.UpdateBrontolReport(ctx, report); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui laporan brontol")
	}
	return report, nil
}

// DeleteBrontolReport removes a brontol report
func (uc *qcUseCase) DeleteBrontolReport(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteBrontolReport(ctx, id)
}

// CreateRejectReturn generates or attaches a return report for an existing brontol report
func (uc *qcUseCase) CreateRejectReturn(ctx context.Context, brontolID uuid.UUID, dto CreateRejectReturnDTO) (*domain.RejectReturnReport, error) {
	brontol, err := uc.repo.GetBrontolReportByID(ctx, brontolID)
	if err != nil {
		return nil, apperrors.NewNotFound("Laporan brontol tidak ditemukan")
	}

	existing, _ := uc.repo.GetRejectReturnReportByBrontolID(ctx, brontolID)
	if existing != nil {
		return existing, nil
	}

	if dto.ReturnDate == "" {
		dto.ReturnDate = time.Now().Format("2006-01-02")
	}
	returnNo, err := docseq.Next(ctx, uc.repo, "RTR", dto.ReturnDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor laporan retur")
	}

	unit := dto.Unit
	if unit == "" {
		unit = brontol.Unit
	}
	qty := dto.ReturnQuantity
	if qty <= 0 {
		qty = brontol.Quantity
	}

	ret := &domain.RejectReturnReport{
		BrontolReportID: brontol.ID,
		ReturnNo:        returnNo,
		DocumentCode:    "RET-REJECT-01",
		ReturnDate:      dto.ReturnDate,
		SupplierID:      dto.SupplierID,
		SupplierName:    dto.SupplierName,
		PONo:            dto.PONo,
		GoodsReceiptNo:  dto.GoodsReceiptNo,
		DeliveryNoteNo:  dto.DeliveryNoteNo,
		ProductName:     brontol.ProductName,
		ProductCode:     brontol.ProductCode,
		BatchNo:         brontol.BatchNo,
		ReturnQuantity:  qty,
		Unit:            unit,
		UnitPrice:       dto.UnitPrice,
		EstimatedValue:  dto.EstimatedValue,
		ReturnReason:    dto.ReturnReason,
		CourierName:     dto.CourierName,
		VehicleNumber:   dto.VehicleNumber,
		DriverName:      dto.DriverName,
		PreparedBy:      dto.PreparedBy,
		ApprovedBy:      dto.ApprovedBy,
		Status:          "pending_pickup",
		Notes:           dto.Notes,
	}

	if err := uc.repo.CreateRejectReturnReport(ctx, ret); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat laporan retur barang reject")
	}

	// Update brontol action & isReturn
	brontol.ActionDisposition = domain.ActionReturnToSupplier
	brontol.IsReturn = true
	brontol.Status = "returned"
	_ = uc.repo.UpdateBrontolReport(ctx, brontol)

	return ret, nil
}

// GetRejectReturnByID retrieves a reject return report by ID
func (uc *qcUseCase) GetRejectReturnByID(ctx context.Context, id uuid.UUID) (*domain.RejectReturnReport, error) {
	ret, err := uc.repo.GetRejectReturnReportByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Laporan retur barang reject tidak ditemukan")
	}
	return ret, nil
}

// ListRejectReturns retrieves reject return reports matching filters
func (uc *qcUseCase) ListRejectReturns(ctx context.Context, month, status, search string) ([]domain.RejectReturnReport, error) {
	return uc.repo.ListRejectReturnReports(ctx, month, status, search)
}

// UpdateRejectReturn updates shipping/status info on a return report
func (uc *qcUseCase) UpdateRejectReturn(ctx context.Context, id uuid.UUID, dto UpdateRejectReturnDTO) (*domain.RejectReturnReport, error) {
	ret, err := uc.repo.GetRejectReturnReportByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Laporan retur tidak ditemukan")
	}

	if dto.ReturnDate != "" {
		ret.ReturnDate = dto.ReturnDate
	}
	if dto.SupplierName != "" {
		ret.SupplierName = dto.SupplierName
	}
	if dto.PONo != "" {
		ret.PONo = dto.PONo
	}
	if dto.GoodsReceiptNo != "" {
		ret.GoodsReceiptNo = dto.GoodsReceiptNo
	}
	if dto.DeliveryNoteNo != "" {
		ret.DeliveryNoteNo = dto.DeliveryNoteNo
	}
	if dto.ReturnQuantity != nil {
		ret.ReturnQuantity = *dto.ReturnQuantity
	}
	if dto.UnitPrice != nil {
		ret.UnitPrice = *dto.UnitPrice
	}
	if dto.EstimatedValue != nil {
		ret.EstimatedValue = *dto.EstimatedValue
	}
	if dto.ReturnReason != "" {
		ret.ReturnReason = dto.ReturnReason
	}
	if dto.CourierName != "" {
		ret.CourierName = dto.CourierName
	}
	if dto.VehicleNumber != "" {
		ret.VehicleNumber = dto.VehicleNumber
	}
	if dto.DriverName != "" {
		ret.DriverName = dto.DriverName
	}
	if dto.ApprovedBy != "" {
		ret.ApprovedBy = dto.ApprovedBy
	}
	if dto.ReceivedBySupplier != "" {
		ret.ReceivedBySupplier = dto.ReceivedBySupplier
	}
	if dto.ShippedDate != nil {
		ret.ShippedDate = dto.ShippedDate
	}
	if dto.CompletedDate != nil {
		ret.CompletedDate = dto.CompletedDate
	}
	if dto.Status != "" {
		ret.Status = dto.Status
		if dto.Status == "shipped" && ret.ShippedDate == nil {
			now := time.Now()
			ret.ShippedDate = &now
		}
		if dto.Status == "completed" && ret.CompletedDate == nil {
			now := time.Now()
			ret.CompletedDate = &now
		}
	}
	if dto.Notes != "" {
		ret.Notes = dto.Notes
	}

	if err := uc.repo.UpdateRejectReturnReport(ctx, ret); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui laporan retur")
	}
	return ret, nil
}

// DeleteRejectReturn removes a return report
func (uc *qcUseCase) DeleteRejectReturn(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteRejectReturnReport(ctx, id)
}

// GetBrontolStats computes KPI statistics
func (uc *qcUseCase) GetBrontolStats(ctx context.Context) (*BrontolStatsResponseDTO, error) {
	totalCount, totalQty, returnCount, retQty, reworkCount, err := uc.repo.GetBrontolStats(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menghitung statistik brontol")
	}
	return &BrontolStatsResponseDTO{
		TotalCount:      totalCount,
		TotalQuantityKg: totalQty,
		ReturnCount:     returnCount,
		ReturnQtyKg:     retQty,
		ReworkCount:     reworkCount,
	}, nil
}

// GetBrontolStockCard produces the formatted table matching Form REP-013-KSB
func (uc *qcUseCase) GetBrontolStockCard(ctx context.Context, month string, productCode string) ([]BrontolStockCardItemDTO, error) {
	reports, err := uc.repo.ListBrontolReports(ctx, month, "", "", "", productCode)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil data kartu stok brontol")
	}

	// Order chronological for running balance calculation
	items := make([]BrontolStockCardItemDTO, 0, len(reports))
	var runningBalance float64

	for i, r := range reports {
		isBM := r.MaterialType == "BM"
		isBO := r.MaterialType == "BO"
		inQty := r.Quantity
		var outQty float64
		var outDate string

		// If disposed or reprocessed or return completed, calculate as Keluar
		if r.ActionDisposition == domain.ActionReturnToSupplier || r.ActionDisposition == domain.ActionRework || r.ActionDisposition == domain.ActionDisposal {
			if r.Status == "returned" || r.Status == "reprocessed" || r.Status == "disposed" || r.Status == "closed" {
				outQty = r.Quantity
				outDate = r.ReportDate
			}
		}

		runningBalance += inQty - outQty
		if runningBalance < 0 {
			runningBalance = 0
		}

		retNo := ""
		if r.ReturnReport != nil {
			retNo = r.ReturnReport.ReturnNo
		}

		items = append(items, BrontolStockCardItemDTO{
			ID:                r.ID,
			RowNo:             i + 1,
			ReportNo:          r.ReportNo,
			BatchNo:           r.BatchNo,
			ProductCode:       r.ProductCode,
			ProductName:       r.ProductName,
			IsBM:              isBM,
			IsBO:              isBO,
			InDate:            r.ReportDate,
			InQty:             inQty,
			OutDate:           outDate,
			OutQty:            outQty,
			BalanceQty:        runningBalance,
			MachineNo:         r.MachineNo,
			MeshSize:          r.MeshSize,
			OperatorName:      r.OperatorName,
			ActionDisposition: r.ActionDisposition,
			Status:            r.Status,
			HasReturnReport:   r.IsReturn,
			ReturnNo:          retNo,
			SupplierName:      r.SupplierName,
		})
	}

	return items, nil
}

// AutoCreateBrontolOnQCFailure handles automated report generation across any QC failure point
func (uc *qcUseCase) AutoCreateBrontolOnQCFailure(ctx context.Context, trigger AutoBrontolTrigger) (res *domain.BrontolReport, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("Recovered from panic during auto-create brontol report", "panic", r, "stage", trigger.SourceStage)
			res = nil
			err = nil
		}
	}()

	qty := trigger.Quantity
	if qty <= 0 {
		qty = 1.0 // default minimal 1 kg/unit placeholder if not explicitly weighed yet
	}

	unit := trigger.Unit
	if unit == "" {
		unit = "KG"
	}

	action := trigger.ActionDisposition
	if action == "" {
		action = domain.ActionHold
	}

	dto := CreateBrontolReportDTO{
		ReportDate:         time.Now().Format("2006-01-02"),
		SourceStage:        trigger.SourceStage,
		SourceInspectionID: trigger.SourceInspectionID,
		SourceInspectionNo: trigger.SourceInspectionNo,
		ProductionOrderID:  trigger.ProductionOrderID,
		OrderNumber:        trigger.OrderNumber,
		ProductID:          trigger.ProductID,
		ProductName:        trigger.ProductName,
		ProductCode:        trigger.ProductCode,
		BatchNo:            trigger.BatchNo,
		MaterialType:       trigger.MaterialType,
		SupplierID:         trigger.SupplierID,
		SupplierName:       trigger.SupplierName,
		MachineNo:          trigger.MachineNo,
		MeshSize:           trigger.MeshSize,
		Quantity:           qty,
		Unit:               unit,
		DefectReason:       trigger.DefectReason,
		ActionDisposition:  action,
		OperatorName:       trigger.OperatorName,
		QCInspectorName:    trigger.QCInspectorName,
		Notes:              trigger.Notes,
	}

	if action == domain.ActionReturnToSupplier {
		dto.CreateReturn = &CreateRejectReturnDTO{
			ReturnDate:     time.Now().Format("2006-01-02"),
			SupplierID:     trigger.SupplierID,
			SupplierName:   trigger.SupplierName,
			PONo:           trigger.PONo,
			GoodsReceiptNo: trigger.GoodsReceiptNo,
			DeliveryNoteNo: trigger.DeliveryNoteNo,
			ReturnQuantity: qty,
			Unit:           unit,
			UnitPrice:      trigger.UnitPrice,
			EstimatedValue: qty * trigger.UnitPrice,
			ReturnReason:   trigger.DefectReason,
			PreparedBy:     trigger.QCInspectorName,
		}
	}

	report, err := uc.CreateBrontolReport(ctx, dto)
	if err != nil {
		slog.Error("Failed to auto-create brontol report on QC failure", "stage", trigger.SourceStage, "batch", trigger.BatchNo, "error", err)
		return nil, err
	}

	slog.Info("Auto-created brontol report on QC failure", "reportNo", report.ReportNo, "stage", report.SourceStage, "isReturn", report.IsReturn)
	return report, nil
}
