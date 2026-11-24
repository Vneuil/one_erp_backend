package application

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	inventoryDomain "github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	"github.com/divinecoid/one-backend/internal/shared/docseq"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// Requisitions (BPB: GDG-011-BPB, PBPK: GDG-012-BPK, LPC: GDG-008-LPC)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *manufacturingUseCase) CreateRequisition(ctx context.Context, dto CreateMaterialRequisitionDTO) (*domain.MaterialRequisition, error) {
	prefix := "REQ"
	docCode := "GDG-011-BPB"
	switch dto.RequisitionType {
	case domain.RequisitionTypeRawMaterial:
		prefix = "BPB"
		docCode = "GDG-011-BPB"
	case domain.RequisitionTypePackaging:
		prefix = "BPK"
		docCode = "GDG-012-BPK"
	case domain.RequisitionTypeLiquidSugar:
		prefix = "LPC"
		docCode = "GDG-008-LPC"
	}
	if dto.DocumentCode != "" {
		docCode = dto.DocumentCode
	}

	reqNo, err := docseq.Next(ctx, uc.repo, prefix, dto.RequestDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor dokumen permintaan")
	}

	matCategory := dto.MaterialCategory
	if matCategory == "" {
		matCategory = "GSKN"
	}
	fairtradeStatus := dto.FairtradeStatus
	if fairtradeStatus == "" {
		fairtradeStatus = "Fairtrade"
	}

	lines := make([]domain.MaterialRequisitionLine, 0, len(dto.Lines))
	for _, l := range dto.Lines {
		pkgUnit := l.PackageUnit
		if pkgUnit == "" {
			pkgUnit = "Sak"
		}
		uom := l.UoM
		if uom == "" {
			uom = "KG"
		}

		line := domain.MaterialRequisitionLine{
			RawMaterialStockID:    l.RawMaterialStockID,
			SBBEntryNo:            l.SBBEntryNo,
			ProductID:             l.ProductID,
			ProductSKU:            l.ProductSKU,
			ProductName:           l.ProductName,
			BatchNo:               l.BatchNo,
			SupplierName:          l.SupplierName,
			ArrivalDateRef:        l.ArrivalDateRef,
			RackLocation:          l.RackLocation,
			PalletNo:              l.PalletNo,
			MoistureContent:       l.MoistureContent,
			QuantityRequested:     l.QuantityRequested,
			QuantityIssued:        l.QuantityIssued,
			QuantityUsed:          l.QuantityUsed,
			QuantityReject:        l.QuantityReject,
			QuantityReturned:      l.QuantityReturned,
			PackageCountRequested: l.PackageCountRequested,
			PackageCountIssued:    l.PackageCountIssued,
			PackageUnit:           pkgUnit,
			UoM:                   uom,
			Notes:                 l.Notes,
		}

		// If line links to RawMaterialStockEntry (SBB), validate availability & reserve limit
		if l.RawMaterialStockID != nil && uc.inventoryRepo != nil {
			sbb, err := uc.inventoryRepo.GetRawMaterialStockEntry(ctx, *l.RawMaterialStockID)
			if err != nil || sbb == nil {
				return nil, apperrors.NewBadRequest(fmt.Sprintf("Data Stok Gudang Bahan Baku lot %s tidak ditemukan", l.BatchNo))
			}
			if sbb.Status != inventoryDomain.RawStockStatusAvailable {
				return nil, apperrors.NewBadRequest(fmt.Sprintf("Lot SBB %s belum berstatus available untuk produksi (status: %s)", sbb.BatchNo, sbb.Status))
			}
			availableToReserve := sbb.QtyRemaining - sbb.QtyReserved
			if l.QuantityRequested > (availableToReserve + 0.0001) {
				return nil, apperrors.NewBadRequest(fmt.Sprintf("Jumlah yang diminta (%.2f %s) melebihi sisa stok lot SBB yang tersedia (%.2f %s)", l.QuantityRequested, sbb.UoM, availableToReserve, sbb.UoM))
			}

			// Auto-fill metadata from SBB
			line.SBBEntryNo = sbb.EntryNo
			line.ProductID = sbb.ProductID
			line.ProductSKU = sbb.ProductSKU
			line.ProductName = sbb.ProductName
			line.BatchNo = sbb.BatchNo
			line.SupplierName = sbb.SupplierName
			if line.ArrivalDateRef == "" && sbb.EntryDate != "" {
				line.ArrivalDateRef = "Tgl. " + sbb.EntryDate
			}
			line.RackLocation = sbb.RackLocation
			line.PalletNo = sbb.PalletNo
			line.MoistureContent = sbb.MoistureContent
			line.PackageUnit = sbb.PackageUnit
			line.UoM = sbb.UoM

			if line.QuantityIssued == 0 {
				line.QuantityIssued = line.QuantityRequested
			}
			if line.PackageCountIssued == 0 {
				line.PackageCountIssued = line.PackageCountRequested
			}
			if dto.WarehouseID == nil && sbb.WarehouseID != uuid.Nil {
				whID := sbb.WarehouseID
				dto.WarehouseID = &whID
			}
		}

		lines = append(lines, line)
	}

	dispensing := make([]domain.LiquidSugarDispensingItem, 0, len(dto.DispensingItems))
	for _, d := range dto.DispensingItems {
		netWeight := d.GrossWeight - d.TareWeight
		if netWeight < 0 {
			netWeight = 0
		}
		dispensing = append(dispensing, domain.LiquidSugarDispensingItem{
			JerryCanNo:  d.JerryCanNo,
			GrossWeight: d.GrossWeight,
			TareWeight:  d.TareWeight,
			NetWeight:   netWeight,
			Brix:        d.Brix,
			AromaColor:  d.AromaColor,
			Notes:       d.Notes,
		})
	}

	req := &domain.MaterialRequisition{
		RequestNo:         reqNo,
		DocumentCode:      docCode,
		RequisitionType:   dto.RequisitionType,
		MaterialCategory:  matCategory,
		FairtradeStatus:   fairtradeStatus,
		WarehouseID:       dto.WarehouseID,
		ProductionOrderID: dto.ProductionOrderID,
		OrderNumber:       dto.OrderNumber,
		RequestDate:       dto.RequestDate,
		Department:        dto.Department,
		RequestedBy:       dto.RequestedBy,
		Status:            "submitted",
		Notes:             dto.Notes,
		Lines:             lines,
		DispensingItems:   dispensing,
	}

	// Atomic save requisition and reserve SBB lot stock
	err = uc.repo.WithTransactionAndInventory(ctx, func(txRepo domain.ManufacturingRepository, txInv inventoryDomain.InventoryRepository) error {
		if err := txRepo.CreateRequisition(ctx, req); err != nil {
			return err
		}

		// Reserve stock in SBB for raw material lines
		for _, line := range req.Lines {
			if line.RawMaterialStockID != nil && txInv != nil {
				sbb, err := txInv.GetRawMaterialStockEntry(ctx, *line.RawMaterialStockID)
				if err != nil || sbb == nil {
					continue
				}
				sbb.QtyReserved += line.QuantityRequested
				if err := txInv.UpdateRawMaterialStockEntry(ctx, sbb); err != nil {
					return err
				}

				// Log reserve event
				_ = txInv.CreateRawMaterialStockLog(ctx, &inventoryDomain.RawMaterialStockLog{
					EntryID:       sbb.ID,
					EventType:     inventoryDomain.RawStockEventBPBReserve,
					RefDocumentNo: req.RequestNo,
					QtyDelta:      0,
					PackageDelta:  0,
					BalanceAfter:  sbb.QtyRemaining,
					PackagesAfter: sbb.RemainingPackages,
					ActorName:     req.RequestedBy,
					Notes:         fmt.Sprintf("Reservasi BPB %s untuk departemen %s (diminta: %.2f %s)", req.RequestNo, req.Department, line.QuantityRequested, sbb.UoM),
					OccurredAt:    time.Now(),
				})
			}
		}

		return nil
	})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create material requisition and reserve stock")
	}

	return uc.repo.GetRequisitionByID(ctx, req.ID)
}

func (uc *manufacturingUseCase) GetRequisitionByID(ctx context.Context, id uuid.UUID) (*domain.MaterialRequisition, error) {
	req, err := uc.repo.GetRequisitionByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Material requisition not found")
	}
	return req, nil
}

func (uc *manufacturingUseCase) ListRequisitions(ctx context.Context, reqType string, status string) ([]domain.MaterialRequisition, error) {
	list, err := uc.repo.ListRequisitions(ctx, reqType, status)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list material requisitions")
	}
	return list, nil
}

func (uc *manufacturingUseCase) UpdateRequisitionStatus(ctx context.Context, id uuid.UUID, dto UpdateRequisitionStatusDTO) error {
	if err := uc.repo.UpdateRequisitionStatus(ctx, id, dto.Status, dto.IssuedBy); err != nil {
		return apperrors.NewInternal(err, "Failed to update requisition status")
	}
	return nil
}

// ApproveRequisitionByWarehouse handles ACC Gudang (Manager Gudang):
// It atomically deducts remaining quantities from the linked RawMaterialStockEntry (SBB),
// creates the bpb_issue stock log, deducts the main warehouse stock levels/batches,
// and advances the BPB status to warehouse_approved.
func (uc *manufacturingUseCase) ApproveRequisitionByWarehouse(ctx context.Context, id uuid.UUID, dto ApproveRequisitionByWarehouseDTO) (*domain.MaterialRequisition, error) {
	req, err := uc.repo.GetRequisitionByID(ctx, id)
	if err != nil || req == nil {
		return nil, apperrors.NewNotFound("Material requisition not found")
	}

	if req.StockDeducted || req.Status == "warehouse_approved" || req.Status == "in_production" || req.Status == "completed" {
		return nil, apperrors.NewBadRequest("Permintaan bahan baku ini sudah disetujui gudang atau telah diproses ke produksi")
	}
	if req.Status == "rejected" {
		return nil, apperrors.NewBadRequest("Permintaan bahan baku ini telah ditolak")
	}

	approverName := strings.TrimSpace(dto.WarehouseApprovedBy)
	if approverName == "" {
		approverName = "Manager Gudang"
	}
	now := time.Now()

	err = uc.repo.WithTransactionAndInventory(ctx, func(txRepo domain.ManufacturingRepository, txInv inventoryDomain.InventoryRepository) error {
		for i := range req.Lines {
			line := &req.Lines[i]
			issuedQty := line.QuantityIssued
			if issuedQty <= 0 {
				issuedQty = line.QuantityRequested
			}
			issuedPkgs := line.PackageCountIssued
			if issuedPkgs <= 0 {
				issuedPkgs = line.PackageCountRequested
			}

			// If linked to SBB, deduct lot stock
			if line.RawMaterialStockID != nil && txInv != nil {
				sbb, err := txInv.GetRawMaterialStockEntry(ctx, *line.RawMaterialStockID)
				if err != nil || sbb == nil {
					return apperrors.NewBadRequest(fmt.Sprintf("Lot SBB %s tidak ditemukan", line.SBBEntryNo))
				}
				if issuedQty > (sbb.QtyRemaining + 0.0001) {
					return apperrors.NewBadRequest(fmt.Sprintf("Jumlah pengeluaran (%.2f %s) melebihi sisa stok lot SBB %s (%.2f %s)", issuedQty, sbb.UoM, sbb.BatchNo, sbb.QtyRemaining, sbb.UoM))
				}

				// Deduct reserved and add to issued
				sbb.QtyReserved = math.Max(0, sbb.QtyReserved-line.QuantityRequested)
				sbb.QtyIssued += issuedQty
				sbb.QtyRemaining = math.Max(0, sbb.QtyReceived-sbb.QtyIssued)
				sbb.RemainingPackages = int(math.Max(0, float64(sbb.RemainingPackages-issuedPkgs)))
				if sbb.QtyRemaining <= 0.0001 {
					sbb.Status = inventoryDomain.RawStockStatusDepleted
					sbb.QtyRemaining = 0
					sbb.RemainingPackages = 0
				}

				if err := txInv.UpdateRawMaterialStockEntry(ctx, sbb); err != nil {
					return err
				}

				// Log bpb_issue
				_ = txInv.CreateRawMaterialStockLog(ctx, &inventoryDomain.RawMaterialStockLog{
					EntryID:       sbb.ID,
					EventType:     inventoryDomain.RawStockEventBPBIssue,
					RefDocumentNo: req.RequestNo,
					QtyDelta:      -issuedQty,
					PackageDelta:  -issuedPkgs,
					BalanceAfter:  sbb.QtyRemaining,
					PackagesAfter: sbb.RemainingPackages,
					ActorName:     approverName,
					Notes:         fmt.Sprintf("ACC Gudang pengeluaran bahan baku untuk %s: %s", req.Department, dto.Notes),
					OccurredAt:    now,
				})

				// Deduct from physical warehouse stock levels and batches
				targetWarehouseID := req.WarehouseID
				if targetWarehouseID == nil && sbb.WarehouseID != uuid.Nil {
					whID := sbb.WarehouseID
					targetWarehouseID = &whID
				}
				if targetWarehouseID != nil {
					stk, err := txInv.GetStockLevel(ctx, line.ProductID, *targetWarehouseID)
					if err == nil && stk != nil {
						deductInt := int(math.Round(issuedQty))
						stk.Quantity -= deductInt
						if stk.Quantity < 0 {
							stk.Quantity = 0
						}
						_ = txInv.UpsertStockLevel(ctx, stk)
					}

					if line.BatchNo != "" {
						batch, err := txInv.GetBatchByNo(ctx, line.ProductID, *targetWarehouseID, line.BatchNo)
						if err == nil && batch != nil {
							deductInt := int(math.Round(issuedQty))
							batch.Quantity -= deductInt
							if batch.Quantity < 0 {
								batch.Quantity = 0
							}
							_ = txInv.SaveBatch(ctx, batch)
						}
					}

					_ = txInv.CreateMovement(ctx, &inventoryDomain.StockMovement{
						ProductID:   line.ProductID,
						WarehouseID: *targetWarehouseID,
						Type:        "out",
						Quantity:    -int(math.Round(issuedQty)),
						Reference:   req.RequestNo,
						Reason:      fmt.Sprintf("Pengeluaran Bahan Baku BPB %s", req.RequestNo),
						BatchNo:     line.BatchNo,
						CreatedBy:   approverName,
					})
				}
			}

			line.QuantityIssued = issuedQty
			line.PackageCountIssued = issuedPkgs
			_ = txRepo.UpdateRequisitionLine(ctx, line)
		}

		// Update linked LBB reports to approved
		for _, lbb := range req.LBBReports {
			lbb.Status = "approved"
			_ = txRepo.UpdateLBBReport(ctx, &lbb)
		}

		req.Status = "warehouse_approved"
		req.StockDeducted = true
		req.IssuedBy = approverName
		req.IssuedAt = &now
		req.WarehouseApprovedBy = approverName
		req.WarehouseApprovedAt = &now
		if dto.Notes != "" {
			if req.Notes != "" {
				req.Notes += " | " + dto.Notes
			} else {
				req.Notes = dto.Notes
			}
		}

		return txRepo.UpdateRequisition(ctx, req)
	})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memproses ACC Gudang")
	}

	return uc.repo.GetRequisitionByID(ctx, req.ID)
}

// EnterProductionFromRequisition completes handing over materials to the production line:
// Advances status to in_production, creates BatchGenealogy records linking raw lot to order,
// and evaluates routing step conditions (such as moisture content check Rule BRL-01).
func (uc *manufacturingUseCase) EnterProductionFromRequisition(ctx context.Context, id uuid.UUID, dto EnterProductionDTO) (*domain.MaterialRequisition, error) {
	req, err := uc.repo.GetRequisitionByID(ctx, id)
	if err != nil || req == nil {
		return nil, apperrors.NewNotFound("Material requisition not found")
	}

	if req.Status != "warehouse_approved" {
		return nil, apperrors.NewBadRequest("Permintaan bahan baku harus di-ACC Gudang terlebih dahulu sebelum masuk ke lini produksi")
	}

	enteredBy := strings.TrimSpace(dto.EnteredProductionBy)
	if enteredBy == "" {
		enteredBy = "Produksi"
	}
	now := time.Now()

	req.Status = "in_production"
	req.EnteredProductionBy = enteredBy
	req.EnteredProductionAt = &now
	if dto.Notes != "" {
		if req.Notes != "" {
			req.Notes += " | " + dto.Notes
		} else {
			req.Notes = dto.Notes
		}
	}

	// Link with Production Order if present
	if req.ProductionOrderID != nil {
		order, err := uc.repo.GetOrderByID(ctx, *req.ProductionOrderID)
		if err == nil && order != nil {
			if order.Status == domain.OrderPlanned {
				order.Status = domain.OrderReleased
				_ = uc.repo.UpdateOrder(ctx, order)
			}

			// Record genealogy & evaluate routing
			for _, line := range req.Lines {
				issuedQty := line.QuantityIssued
				if issuedQty <= 0 {
					issuedQty = line.QuantityRequested
				}

				childProdID := line.ProductID
				childProdName := order.OrderNumber
				if bom, err := uc.repo.GetBOMByID(ctx, order.BOMID); err == nil && bom != nil {
					childProdID = bom.ProductID
					childProdName = bom.Name
				}

				_ = uc.repo.CreateGenealogy(ctx, &domain.BatchGenealogy{
					ParentBatchNo:     line.BatchNo,
					ParentProductID:   line.ProductID,
					ParentProductName: line.ProductName,
					ChildBatchNo:      order.OrderNumber,
					ChildProductID:    childProdID,
					ChildProductName:  childProdName,
					ProductionOrderID: order.ID,
					OrderNumber:       order.OrderNumber,
					QuantityUsed:      issuedQty,
					UoM:               line.UoM,
					RecordedAt:        now,
				})

				// Moisture Content routing evaluation (BRL-01)
				if line.MoistureContent != nil {
					_, _ = uc.EvaluateOrderRouting(ctx, order.ID, "moisture_content", *line.MoistureContent)
				}
			}
		}
	}

	if err := uc.repo.UpdateRequisition(ctx, req); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengupdate status requisition masuk ke produksi")
	}

	return uc.repo.GetRequisitionByID(ctx, req.ID)
}

// RejectRequisition rejects a requisition before warehouse approval, releasing reserved SBB lot stock.
func (uc *manufacturingUseCase) RejectRequisition(ctx context.Context, id uuid.UUID, dto RejectRequisitionDTO) (*domain.MaterialRequisition, error) {
	req, err := uc.repo.GetRequisitionByID(ctx, id)
	if err != nil || req == nil {
		return nil, apperrors.NewNotFound("Material requisition not found")
	}

	if req.StockDeducted || req.Status == "warehouse_approved" || req.Status == "in_production" || req.Status == "completed" {
		return nil, apperrors.NewBadRequest("Tidak dapat menolak permintaan yang sudah di-ACC Gudang atau telah masuk produksi")
	}

	reason := strings.TrimSpace(dto.RejectionReason)
	if reason == "" {
		reason = "Ditolak oleh Gudang"
	}

	err = uc.repo.WithTransactionAndInventory(ctx, func(txRepo domain.ManufacturingRepository, txInv inventoryDomain.InventoryRepository) error {
		// Release SBB reserved stock
		for _, line := range req.Lines {
			if line.RawMaterialStockID != nil && txInv != nil {
				sbb, err := txInv.GetRawMaterialStockEntry(ctx, *line.RawMaterialStockID)
				if err == nil && sbb != nil {
					sbb.QtyReserved = math.Max(0, sbb.QtyReserved-line.QuantityRequested)
					_ = txInv.UpdateRawMaterialStockEntry(ctx, sbb)

					_ = txInv.CreateRawMaterialStockLog(ctx, &inventoryDomain.RawMaterialStockLog{
						EntryID:       sbb.ID,
						EventType:     inventoryDomain.RawStockEventVoid,
						RefDocumentNo: req.RequestNo,
						QtyDelta:      0,
						PackageDelta:  0,
						BalanceAfter:  sbb.QtyRemaining,
						PackagesAfter: sbb.RemainingPackages,
						ActorName:     "Gudang",
						Notes:         fmt.Sprintf("Pembatalan reservasi BPB %s: %s", req.RequestNo, reason),
						OccurredAt:    time.Now(),
					})
				}
			}
		}

		req.Status = "rejected"
		req.RejectionReason = reason
		return txRepo.UpdateRequisition(ctx, req)
	})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menolak permintaan bahan baku")
	}

	return uc.repo.GetRequisitionByID(ctx, req.ID)
}

// ─────────────────────────────────────────────────────────────────────────────
// LBB (Laporan Permintaan Bahan Baku — GDG-013-LBB)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *manufacturingUseCase) CreateLBBReport(ctx context.Context, dto CreateLBBReportDTO) (*domain.RawMaterialRequestReport, error) {
	reportNo, err := docseq.Next(ctx, uc.repo, "LBB", dto.ReportDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor laporan LBB")
	}

	variant := strings.ToUpper(strings.TrimSpace(dto.VariantType))
	if variant != "VCO" {
		variant = "GSKN"
	}
	uom := "KG"
	opRole := "Operator Ayak"
	if variant == "VCO" {
		uom = "L"
		opRole = "Operator Filling"
	}
	if dto.UoM != "" {
		uom = dto.UoM
	}
	if dto.OperatorRole != "" {
		opRole = dto.OperatorRole
	}

	// Calculate 1..300 bag entries
	var totalGross float64
	var filledCount int
	bagEntries := make([]domain.RawMaterialRequestBagEntry, 0, len(dto.BagEntries))
	for _, entry := range dto.BagEntries {
		colGroup := ((entry.RowNumber - 1) / 50) + 1
		if colGroup < 1 {
			colGroup = 1
		} else if colGroup > 6 {
			colGroup = 6
		}
		if entry.WeightValue != nil && *entry.WeightValue > 0 {
			totalGross += *entry.WeightValue
			filledCount++
		}
		lotCode := entry.LotCode
		if lotCode == "" {
			lotCode = dto.MaterialCode
		}
		bagEntries = append(bagEntries, domain.RawMaterialRequestBagEntry{
			RowNumber:   entry.RowNumber,
			ColumnGroup: colGroup,
			LotCode:     lotCode,
			WeightValue: entry.WeightValue,
		})
	}

	plasticTare := 0.0
	if variant == "GSKN" && dto.PlasticTarePerBag > 0 {
		plasticTare = float64(filledCount) * dto.PlasticTarePerBag
	}
	totalNet := math.Max(0, totalGross-plasticTare)

	bpbNo := dto.BPBNo
	fairtradeStatus := dto.FairtradeStatus
	if fairtradeStatus == "" {
		fairtradeStatus = "Fairtrade"
	}
	orderNumber := dto.OrderNumber
	prodOrderID := dto.ProductionOrderID

	// Auto-pull from BPB if present
	if dto.RequisitionID != nil {
		req, err := uc.repo.GetRequisitionByID(ctx, *dto.RequisitionID)
		if err == nil && req != nil {
			if bpbNo == "" {
				bpbNo = req.RequestNo
			}
			if prodOrderID == nil {
				prodOrderID = req.ProductionOrderID
				orderNumber = req.OrderNumber
			}
			if dto.FairtradeStatus == "" {
				fairtradeStatus = req.FairtradeStatus
			}
		}
	}

	rep := &domain.RawMaterialRequestReport{
		ReportNo:          reportNo,
		DocumentCode:      "GDG-013-LBB",
		RequisitionID:     dto.RequisitionID,
		RequisitionLineID: dto.RequisitionLineID,
		BPBNo:             bpbNo,
		RawMaterialStockID: dto.RawMaterialStockID,
		SBBEntryNo:        dto.SBBEntryNo,
		ProductionOrderID: prodOrderID,
		OrderNumber:       orderNumber,
		ReportDate:        dto.ReportDate,
		VariantType:       variant,
		FairtradeStatus:   fairtradeStatus,
		ProductID:         dto.ProductID,
		MaterialCode:      dto.MaterialCode,
		ProductName:       dto.ProductName,
		SupplierName:      dto.SupplierName,
		CustomerName:      dto.CustomerName,
		MachineNo:         dto.MachineNo,
		MeshSize:          dto.MeshSize,
		UoM:               uom,
		PlasticTarePerBag: dto.PlasticTarePerBag,
		TotalBags:         filledCount,
		TotalGrossWeight:  totalGross,
		TotalPlasticTare:  plasticTare,
		TotalNetWeight:    totalNet,
		OperatorRole:      opRole,
		PreparedBy:        dto.PreparedBy,
		PreparedDate:      dto.PreparedDate,
		AcknowledgedBy:    dto.AcknowledgedBy,
		AcknowledgedDate:  dto.AcknowledgedDate,
		Status:            "draft",
		Notes:             dto.Notes,
	}

	if err := uc.repo.CreateLBBReport(ctx, rep); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create LBB report")
	}

	if len(bagEntries) > 0 {
		if err := uc.repo.ReplaceLBBBagEntries(ctx, rep.ID, bagEntries); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to save LBB bag entries")
		}
	}

	// Sync back to BPB RequisitionLine if connected
	if rep.RequisitionID != nil && totalNet > 0 {
		req, err := uc.repo.GetRequisitionByID(ctx, *rep.RequisitionID)
		if err == nil && req != nil {
			updatedLine := false
			for i := range req.Lines {
				line := &req.Lines[i]
				if (rep.RequisitionLineID != nil && line.ID == *rep.RequisitionLineID) ||
					(line.ProductID == rep.ProductID && (line.BatchNo == rep.MaterialCode || line.BatchNo == "")) {
					line.QuantityIssued = rep.TotalNetWeight
					line.PackageCountIssued = rep.TotalBags
					_ = uc.repo.UpdateRequisitionLine(ctx, line)
					updatedLine = true
					break
				}
			}
			if updatedLine && req.Status == "submitted" {
				req.Status = "lbb_recorded"
				_ = uc.repo.UpdateRequisition(ctx, req)
			}
		}
	}

	return uc.repo.GetLBBReportByID(ctx, rep.ID)
}

func (uc *manufacturingUseCase) GetLBBReportByID(ctx context.Context, id uuid.UUID) (*domain.RawMaterialRequestReport, error) {
	rep, err := uc.repo.GetLBBReportByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("LBB report not found")
	}
	return rep, nil
}

func (uc *manufacturingUseCase) ListLBBReports(ctx context.Context, requisitionID *uuid.UUID, variantType, status string) ([]domain.RawMaterialRequestReport, error) {
	list, err := uc.repo.ListLBBReports(ctx, requisitionID, variantType, status)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list LBB reports")
	}
	return list, nil
}

func (uc *manufacturingUseCase) UpdateLBBReport(ctx context.Context, id uuid.UUID, dto UpdateLBBReportDTO) (*domain.RawMaterialRequestReport, error) {
	rep, err := uc.repo.GetLBBReportByID(ctx, id)
	if err != nil || rep == nil {
		return nil, apperrors.NewNotFound("LBB report not found")
	}

	if dto.ReportDate != "" {
		rep.ReportDate = dto.ReportDate
	}
	if dto.CustomerName != "" {
		rep.CustomerName = dto.CustomerName
	}
	if dto.MachineNo != "" {
		rep.MachineNo = dto.MachineNo
	}
	if dto.MeshSize != "" {
		rep.MeshSize = dto.MeshSize
	}
	if dto.OperatorRole != "" {
		rep.OperatorRole = dto.OperatorRole
	}
	if dto.PreparedBy != "" {
		rep.PreparedBy = dto.PreparedBy
	}
	if dto.PreparedDate != "" {
		rep.PreparedDate = dto.PreparedDate
	}
	if dto.AcknowledgedBy != "" {
		rep.AcknowledgedBy = dto.AcknowledgedBy
	}
	if dto.AcknowledgedDate != "" {
		rep.AcknowledgedDate = dto.AcknowledgedDate
	}
	if dto.Notes != "" {
		rep.Notes = dto.Notes
	}
	if dto.PlasticTarePerBag != nil {
		rep.PlasticTarePerBag = *dto.PlasticTarePerBag
	}

	// Recalculate weights if bag entries are provided
	if dto.BagEntries != nil {
		var totalGross float64
		var filledCount int
		bagEntries := make([]domain.RawMaterialRequestBagEntry, 0, len(dto.BagEntries))
		for _, entry := range dto.BagEntries {
			colGroup := ((entry.RowNumber - 1) / 50) + 1
			if colGroup < 1 {
				colGroup = 1
			} else if colGroup > 6 {
				colGroup = 6
			}
			if entry.WeightValue != nil && *entry.WeightValue > 0 {
				totalGross += *entry.WeightValue
				filledCount++
			}
			lotCode := entry.LotCode
			if lotCode == "" {
				lotCode = rep.MaterialCode
			}
			bagEntries = append(bagEntries, domain.RawMaterialRequestBagEntry{
				RowNumber:   entry.RowNumber,
				ColumnGroup: colGroup,
				LotCode:     lotCode,
				WeightValue: entry.WeightValue,
			})
		}

		plasticTare := 0.0
		if rep.VariantType == "GSKN" && rep.PlasticTarePerBag > 0 {
			plasticTare = float64(filledCount) * rep.PlasticTarePerBag
		}
		totalNet := math.Max(0, totalGross-plasticTare)

		rep.TotalBags = filledCount
		rep.TotalGrossWeight = totalGross
		rep.TotalPlasticTare = plasticTare
		rep.TotalNetWeight = totalNet

		if err := uc.repo.ReplaceLBBBagEntries(ctx, rep.ID, bagEntries); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to update LBB bag entries")
		}

		// Sync back to BPB line
		if rep.RequisitionID != nil && totalNet > 0 {
			req, err := uc.repo.GetRequisitionByID(ctx, *rep.RequisitionID)
			if err == nil && req != nil {
				for i := range req.Lines {
					line := &req.Lines[i]
					if (rep.RequisitionLineID != nil && line.ID == *rep.RequisitionLineID) ||
						(line.ProductID == rep.ProductID && (line.BatchNo == rep.MaterialCode || line.BatchNo == "")) {
						line.QuantityIssued = rep.TotalNetWeight
						line.PackageCountIssued = rep.TotalBags
						_ = uc.repo.UpdateRequisitionLine(ctx, line)
						break
					}
				}
			}
		}
	}

	if err := uc.repo.UpdateLBBReport(ctx, rep); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update LBB report")
	}

	return uc.repo.GetLBBReportByID(ctx, rep.ID)
}

func (uc *manufacturingUseCase) DeleteLBBReport(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteLBBReport(ctx, id)
}
