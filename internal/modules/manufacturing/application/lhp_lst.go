package application

import (
	"context"
	"fmt"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/docseq"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	"github.com/google/uuid"
)

// DTOs for LHP (PRO-002-LHP)
type CreateLHPItemInput struct {
	ItemNo                 int     `json:"itemNo"`
	RawMaterialCode        string  `json:"rawMaterialCode"`
	ProductName            string  `json:"productName"`
	BatchNo                string  `json:"batchNo"`
	UnitPcs                int     `json:"unitPcs"`
	PrimaryPackagingType   string  `json:"primaryPackagingType,omitempty"`
	PrimaryPackagingCode   string  `json:"primaryPackagingCode,omitempty"`
	SecondaryPackagingType string  `json:"secondaryPackagingType,omitempty"`
	SecondaryPackagingCode string  `json:"secondaryPackagingCode,omitempty"`
	WeightPerPackKg        float64 `json:"weightPerPackKg"`
	TotalWeightKg          float64 `json:"totalWeightKg"`
	StickerNo              string  `json:"stickerNo,omitempty"`
	RemainingStockKg       float64 `json:"remainingStockKg,omitempty"`
	BuyerName              string  `json:"buyerName,omitempty"`
	Notes                  string  `json:"notes,omitempty"`
}

type CreateLHPInput struct {
	ProductionOrderID   *uuid.UUID           `json:"productionOrderId,omitempty"`
	OrderNumber         string               `json:"orderNumber,omitempty"`
	FairtradeStatus     string               `json:"fairtradeStatus"`
	Certifications      string               `json:"certifications,omitempty"`
	PackingDate         string               `json:"packingDate"`
	PackingStartTime    string               `json:"packingStartTime,omitempty"`
	PackingEndTime      string               `json:"packingEndTime,omitempty"`
	EvaluationNotes     string               `json:"evaluationNotes,omitempty"`
	ProductionPIC       string               `json:"productionPic"`
	ProductionSignDate  *string              `json:"productionSignDate,omitempty"`
	AdminPIC            string               `json:"adminPic"`
	AdminSignDate       *string              `json:"adminSignDate,omitempty"`
	Items               []CreateLHPItemInput `json:"items"`
}

type UpdateLHPInput struct {
	FairtradeStatus    string               `json:"fairtradeStatus,omitempty"`
	Certifications     string               `json:"certifications,omitempty"`
	PackingDate        string               `json:"packingDate,omitempty"`
	PackingStartTime   string               `json:"packingStartTime,omitempty"`
	PackingEndTime     string               `json:"packingEndTime,omitempty"`
	EvaluationNotes    string               `json:"evaluationNotes,omitempty"`
	ProductionPIC      string               `json:"productionPic,omitempty"`
	ProductionSignDate *string              `json:"productionSignDate,omitempty"`
	AdminPIC           string               `json:"adminPic,omitempty"`
	AdminSignDate      *string              `json:"adminSignDate,omitempty"`
	Status             string               `json:"status,omitempty"`
	Items              []CreateLHPItemInput `json:"items,omitempty"`
}

// DTOs for LST (GDG-009-LST)
type CreateLSTItemInput struct {
	ItemNo      int     `json:"itemNo"`
	ItemName    string  `json:"itemName"`
	ItemCode    string  `json:"itemCode"`
	BatchNo     string  `json:"batchNo"`
	UnitKg      float64 `json:"unitKg"`
	Packaging   string  `json:"packaging"`
	BuyerName   string  `json:"buyerName,omitempty"`
	QuantityPcs int     `json:"quantityPcs"`
	Notes       string  `json:"notes,omitempty"`
}

type CreateLSTInput struct {
	LHPID               *uuid.UUID           `json:"lhpId,omitempty"`
	LHPReportNo         string               `json:"lhpReportNo,omitempty"`
	FairtradeStatus     string               `json:"fairtradeStatus"`
	FromDepartment      string               `json:"fromDepartment,omitempty"`
	ToDepartment        string               `json:"toDepartment,omitempty"`
	HandoverDate        string               `json:"handoverDate"`
	Notes               string               `json:"notes,omitempty"`
	HandedOverBy        string               `json:"handedOverBy"`
	HandedOverSignDate  *string              `json:"handedOverSignDate,omitempty"`
	VerifiedByQC        string               `json:"verifiedByQc,omitempty"`
	VerifiedSignDate    *string              `json:"verifiedSignDate,omitempty"`
	ReceivedByWarehouse string               `json:"receivedByWarehouse,omitempty"`
	ReceivedSignDate    *string              `json:"receivedSignDate,omitempty"`
	Items               []CreateLSTItemInput `json:"items"`
}

type UpdateLSTInput struct {
	FairtradeStatus     string               `json:"fairtradeStatus,omitempty"`
	HandoverDate        string               `json:"handoverDate,omitempty"`
	Notes               string               `json:"notes,omitempty"`
	HandedOverBy        string               `json:"handedOverBy,omitempty"`
	HandedOverSignDate  *string              `json:"handedOverSignDate,omitempty"`
	VerifiedByQC        string               `json:"verifiedByQc,omitempty"`
	VerifiedSignDate    *string              `json:"verifiedSignDate,omitempty"`
	ReceivedByWarehouse string               `json:"receivedByWarehouse,omitempty"`
	ReceivedSignDate    *string              `json:"receivedSignDate,omitempty"`
	Status              string               `json:"status,omitempty"`
	Items               []CreateLSTItemInput `json:"items,omitempty"`
}

// Generate next LHP Report No (LHP-001-1126)
func (uc *manufacturingUseCase) generateLHPNumber(ctx context.Context, packingDate string) (string, error) {
	return docseq.Next(ctx, uc.repo, "LHP", packingDate)
}

// Generate next LST Report No (LST-001-1126)
func (uc *manufacturingUseCase) generateLSTNumber(ctx context.Context, handoverDate string) (string, error) {
	return docseq.Next(ctx, uc.repo, "LST", handoverDate)
}

// LHP Use Case methods
func (uc *manufacturingUseCase) CreateLHP(ctx context.Context, input CreateLHPInput) (*domain.ProductionOutputReport, error) {
	reportNo, err := uc.generateLHPNumber(ctx, input.PackingDate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate LHP number: %w", err)
	}

	fairtrade := input.FairtradeStatus
	if fairtrade == "" {
		fairtrade = "NON FAIRTRADE"
	}
	certs := input.Certifications
	if certs == "" {
		certs = "NATURAL"
	}

	var totalWeight float64
	var totalPcs int
	var items []domain.ProductionOutputReportItem

	reportID := uuid.New()
	for i, it := range input.Items {
		itemNo := it.ItemNo
		if itemNo <= 0 {
			itemNo = i + 1
		}
		itemTotalWeight := it.TotalWeightKg
		if itemTotalWeight == 0 && it.WeightPerPackKg > 0 && it.UnitPcs > 0 {
			itemTotalWeight = it.WeightPerPackKg * float64(it.UnitPcs)
		}
		totalWeight += itemTotalWeight
		totalPcs += it.UnitPcs

		items = append(items, domain.ProductionOutputReportItem{
			BaseEntity:             types.BaseEntity{ID: uuid.New()},
			ReportID:               reportID,
			ItemNo:                 itemNo,
			RawMaterialCode:        it.RawMaterialCode,
			ProductName:            it.ProductName,
			BatchNo:                it.BatchNo,
			UnitPcs:                it.UnitPcs,
			PrimaryPackagingType:   it.PrimaryPackagingType,
			PrimaryPackagingCode:   it.PrimaryPackagingCode,
			SecondaryPackagingType: it.SecondaryPackagingType,
			SecondaryPackagingCode: it.SecondaryPackagingCode,
			WeightPerPackKg:        it.WeightPerPackKg,
			TotalWeightKg:          itemTotalWeight,
			StickerNo:              it.StickerNo,
			RemainingStockKg:       it.RemainingStockKg,
			BuyerName:              it.BuyerName,
			Notes:                  it.Notes,
		})
	}

	report := &domain.ProductionOutputReport{
		BaseEntity:         types.BaseEntity{ID: reportID, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		ReportNo:           reportNo,
		ProductionOrderID:  input.ProductionOrderID,
		OrderNumber:        input.OrderNumber,
		FairtradeStatus:    fairtrade,
		Certifications:     certs,
		PackingDate:        input.PackingDate,
		PackingStartTime:   input.PackingStartTime,
		PackingEndTime:     input.PackingEndTime,
		EvaluationNotes:    input.EvaluationNotes,
		ProductionPIC:      input.ProductionPIC,
		ProductionSignDate: input.ProductionSignDate,
		AdminPIC:           input.AdminPIC,
		AdminSignDate:      input.AdminSignDate,
		Status:             "completed",
		TotalWeightKg:      totalWeight,
		TotalQuantityPcs:   totalPcs,
		Items:              items,
	}

	if err := uc.repo.CreateLHP(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to create LHP: %w", err)
	}

	return report, nil
}

func (uc *manufacturingUseCase) GetLHP(ctx context.Context, id uuid.UUID) (*domain.ProductionOutputReport, error) {
	return uc.repo.GetLHPByID(ctx, id)
}

func (uc *manufacturingUseCase) ListLHP(ctx context.Context, orderID *uuid.UUID, batchNo, status string) ([]domain.ProductionOutputReport, error) {
	return uc.repo.ListLHP(ctx, orderID, batchNo, status)
}

func (uc *manufacturingUseCase) UpdateLHP(ctx context.Context, id uuid.UUID, input UpdateLHPInput) (*domain.ProductionOutputReport, error) {
	existing, err := uc.repo.GetLHPByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.FairtradeStatus != "" {
		existing.FairtradeStatus = input.FairtradeStatus
	}
	if input.Certifications != "" {
		existing.Certifications = input.Certifications
	}
	if input.PackingDate != "" {
		existing.PackingDate = input.PackingDate
	}
	if input.PackingStartTime != "" {
		existing.PackingStartTime = input.PackingStartTime
	}
	if input.PackingEndTime != "" {
		existing.PackingEndTime = input.PackingEndTime
	}
	if input.EvaluationNotes != "" {
		existing.EvaluationNotes = input.EvaluationNotes
	}
	if input.ProductionPIC != "" {
		existing.ProductionPIC = input.ProductionPIC
	}
	if input.ProductionSignDate != nil {
		existing.ProductionSignDate = input.ProductionSignDate
	}
	if input.AdminPIC != "" {
		existing.AdminPIC = input.AdminPIC
	}
	if input.AdminSignDate != nil {
		existing.AdminSignDate = input.AdminSignDate
	}
	if input.Status != "" {
		existing.Status = input.Status
	}

	if len(input.Items) > 0 {
		var totalWeight float64
		var totalPcs int
		var items []domain.ProductionOutputReportItem
		for i, it := range input.Items {
			itemNo := it.ItemNo
			if itemNo <= 0 {
				itemNo = i + 1
			}
			itemTotalWeight := it.TotalWeightKg
			if itemTotalWeight == 0 && it.WeightPerPackKg > 0 && it.UnitPcs > 0 {
				itemTotalWeight = it.WeightPerPackKg * float64(it.UnitPcs)
			}
			totalWeight += itemTotalWeight
			totalPcs += it.UnitPcs

			items = append(items, domain.ProductionOutputReportItem{
				BaseEntity:             types.BaseEntity{ID: uuid.New()},
				ReportID:               existing.ID,
				ItemNo:                 itemNo,
				RawMaterialCode:        it.RawMaterialCode,
				ProductName:            it.ProductName,
				BatchNo:                it.BatchNo,
				UnitPcs:                it.UnitPcs,
				PrimaryPackagingType:   it.PrimaryPackagingType,
				PrimaryPackagingCode:   it.PrimaryPackagingCode,
				SecondaryPackagingType: it.SecondaryPackagingType,
				SecondaryPackagingCode: it.SecondaryPackagingCode,
				WeightPerPackKg:        it.WeightPerPackKg,
				TotalWeightKg:          itemTotalWeight,
				StickerNo:              it.StickerNo,
				RemainingStockKg:       it.RemainingStockKg,
				BuyerName:              it.BuyerName,
				Notes:                  it.Notes,
			})
		}
		existing.Items = items
		existing.TotalWeightKg = totalWeight
		existing.TotalQuantityPcs = totalPcs
	}
	existing.UpdatedAt = time.Now()

	if err := uc.repo.UpdateLHP(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update LHP: %w", err)
	}

	return existing, nil
}

func (uc *manufacturingUseCase) DeleteLHP(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteLHP(ctx, id)
}

// LST Use Case methods
func (uc *manufacturingUseCase) CreateLST(ctx context.Context, input CreateLSTInput) (*domain.HandoverReport, error) {
	reportNo, err := uc.generateLSTNumber(ctx, input.HandoverDate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate LST number: %w", err)
	}

	fairtrade := input.FairtradeStatus
	if fairtrade == "" {
		fairtrade = "NON FAIRTRADE"
	}
	fromDept := input.FromDepartment
	if fromDept == "" {
		fromDept = "Produksi"
	}
	toDept := input.ToDepartment
	if toDept == "" {
		toDept = "Gudang"
	}

	var totalWeight float64
	var totalPcs int
	var items []domain.HandoverReportItem

	reportID := uuid.New()
	for i, it := range input.Items {
		itemNo := it.ItemNo
		if itemNo <= 0 {
			itemNo = i + 1
		}
		totalWeight += it.UnitKg
		totalPcs += it.QuantityPcs

		items = append(items, domain.HandoverReportItem{
			BaseEntity:  types.BaseEntity{ID: uuid.New()},
			ReportID:    reportID,
			ItemNo:      itemNo,
			ItemName:    it.ItemName,
			ItemCode:    it.ItemCode,
			BatchNo:     it.BatchNo,
			UnitKg:      it.UnitKg,
			Packaging:   it.Packaging,
			BuyerName:   it.BuyerName,
			QuantityPcs: it.QuantityPcs,
			Notes:       it.Notes,
		})
	}

	report := &domain.HandoverReport{
		BaseEntity:          types.BaseEntity{ID: reportID, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		ReportNo:            reportNo,
		LHPID:               input.LHPID,
		LHPReportNo:         input.LHPReportNo,
		FairtradeStatus:     fairtrade,
		FromDepartment:      fromDept,
		ToDepartment:        toDept,
		HandoverDate:        input.HandoverDate,
		Notes:               input.Notes,
		HandedOverBy:        input.HandedOverBy,
		HandedOverSignDate:  input.HandedOverSignDate,
		VerifiedByQC:        input.VerifiedByQC,
		VerifiedSignDate:    input.VerifiedSignDate,
		ReceivedByWarehouse: input.ReceivedByWarehouse,
		ReceivedSignDate:    input.ReceivedSignDate,
		Status:              "draft",
		TotalQuantityPcs:    totalPcs,
		TotalWeightKg:       totalWeight,
		Items:               items,
	}

	if err := uc.repo.CreateLST(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to create LST: %w", err)
	}

	return report, nil
}

func (uc *manufacturingUseCase) GetLST(ctx context.Context, id uuid.UUID) (*domain.HandoverReport, error) {
	return uc.repo.GetLSTByID(ctx, id)
}

func (uc *manufacturingUseCase) ListLST(ctx context.Context, lhpID *uuid.UUID, batchNo, status string) ([]domain.HandoverReport, error) {
	return uc.repo.ListLST(ctx, lhpID, batchNo, status)
}

func (uc *manufacturingUseCase) UpdateLST(ctx context.Context, id uuid.UUID, input UpdateLSTInput) (*domain.HandoverReport, error) {
	existing, err := uc.repo.GetLSTByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.FairtradeStatus != "" {
		existing.FairtradeStatus = input.FairtradeStatus
	}
	if input.HandoverDate != "" {
		existing.HandoverDate = input.HandoverDate
	}
	if input.Notes != "" {
		existing.Notes = input.Notes
	}
	if input.HandedOverBy != "" {
		existing.HandedOverBy = input.HandedOverBy
	}
	if input.HandedOverSignDate != nil {
		existing.HandedOverSignDate = input.HandedOverSignDate
	}
	if input.VerifiedByQC != "" {
		existing.VerifiedByQC = input.VerifiedByQC
	}
	if input.VerifiedSignDate != nil {
		existing.VerifiedSignDate = input.VerifiedSignDate
	}
	if input.ReceivedByWarehouse != "" {
		existing.ReceivedByWarehouse = input.ReceivedByWarehouse
	}
	if input.ReceivedSignDate != nil {
		existing.ReceivedSignDate = input.ReceivedSignDate
	}
	if input.Status != "" {
		existing.Status = input.Status
	}

	if len(input.Items) > 0 {
		var totalWeight float64
		var totalPcs int
		var items []domain.HandoverReportItem
		for i, it := range input.Items {
			itemNo := it.ItemNo
			if itemNo <= 0 {
				itemNo = i + 1
			}
			totalWeight += it.UnitKg
			totalPcs += it.QuantityPcs

			items = append(items, domain.HandoverReportItem{
				BaseEntity:  types.BaseEntity{ID: uuid.New()},
				ReportID:    existing.ID,
				ItemNo:      itemNo,
				ItemName:    it.ItemName,
				ItemCode:    it.ItemCode,
				BatchNo:     it.BatchNo,
				UnitKg:      it.UnitKg,
				Packaging:   it.Packaging,
				BuyerName:   it.BuyerName,
				QuantityPcs: it.QuantityPcs,
				Notes:       it.Notes,
			})
		}
		existing.Items = items
		existing.TotalWeightKg = totalWeight
		existing.TotalQuantityPcs = totalPcs
	}
	existing.UpdatedAt = time.Now()

	if err := uc.repo.UpdateLST(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update LST: %w", err)
	}

	return existing, nil
}

func (uc *manufacturingUseCase) VerifyLSTByQC(ctx context.Context, id uuid.UUID, qcName string) (*domain.HandoverReport, error) {
	existing, err := uc.repo.GetLSTByID(ctx, id)
	if err != nil {
		return nil, err
	}
	today := time.Now().Format("2006-01-02")
	existing.VerifiedByQC = qcName
	existing.VerifiedSignDate = &today
	existing.Status = "verified_qc"
	existing.UpdatedAt = time.Now()

	if err := uc.repo.UpdateLST(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (uc *manufacturingUseCase) AcceptLSTByWarehouse(ctx context.Context, id uuid.UUID, warehouseName string) (*domain.HandoverReport, error) {
	existing, err := uc.repo.GetLSTByID(ctx, id)
	if err != nil {
		return nil, err
	}
	today := time.Now().Format("2006-01-02")
	existing.ReceivedByWarehouse = warehouseName
	existing.ReceivedSignDate = &today
	existing.Status = "accepted_warehouse"
	existing.UpdatedAt = time.Now()

	if err := uc.repo.UpdateLST(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (uc *manufacturingUseCase) DeleteLST(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteLST(ctx, id)
}

// Generate LST from existing LHP
func (uc *manufacturingUseCase) GenerateLSTFromLHP(ctx context.Context, lhpID uuid.UUID, createdBy string) (*domain.HandoverReport, error) {
	lhp, err := uc.repo.GetLHPByID(ctx, lhpID)
	if err != nil {
		return nil, fmt.Errorf("LHP not found: %w", err)
	}

	var items []CreateLSTItemInput
	for _, it := range lhp.Items {
		packaging := it.PrimaryPackagingType
		if it.SecondaryPackagingType != "" {
			packaging += " / " + it.SecondaryPackagingType
		}
		items = append(items, CreateLSTItemInput{
			ItemNo:      it.ItemNo,
			ItemName:    it.ProductName,
			ItemCode:    it.RawMaterialCode,
			BatchNo:     it.BatchNo,
			UnitKg:      it.TotalWeightKg,
			Packaging:   packaging,
			BuyerName:   it.BuyerName,
			QuantityPcs: it.UnitPcs,
			Notes:       it.Notes,
		})
	}

	today := time.Now().Format("2006-01-02")
	handedOver := lhp.ProductionPIC
	if handedOver == "" {
		handedOver = createdBy
	}

	lst, err := uc.CreateLST(ctx, CreateLSTInput{
		LHPID:              &lhp.ID,
		LHPReportNo:        lhp.ReportNo,
		FairtradeStatus:    lhp.FairtradeStatus,
		FromDepartment:     "Produksi",
		ToDepartment:       "Gudang",
		HandoverDate:       today,
		Notes:              fmt.Sprintf("Diserahkan dari LHP %s", lhp.ReportNo),
		HandedOverBy:       handedOver,
		HandedOverSignDate: &today,
		Items:              items,
	})
	if err != nil {
		return nil, err
	}

	lhp.Status = "transferred"
	_ = uc.repo.UpdateLHP(ctx, lhp)

	return lst, nil
}
