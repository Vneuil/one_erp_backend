package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/docseq"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// PLK: Packaging Label Check (Form QC-016-PLK. 01 Revisi 01)
// ─────────────────────────────────────────────────────────────────────────────

type CreatePackagingLabelCheckItemDTO struct {
	ItemNo                int    `json:"itemNo"`
	ItemCheckDate         string `json:"itemCheckDate,omitempty"`
	ItemName              string `json:"itemName"`
	ItemCode              string `json:"itemCode,omitempty"`
	BatchNo               string `json:"batchNo"`
	BuyerName             string `json:"buyerName,omitempty"`
	ProductionDate        string `json:"productionDate,omitempty"`
	ExpiredDate           string `json:"expiredDate,omitempty"`
	OriginCountry         string `json:"originCountry,omitempty"`
	WeightLabel           string `json:"weightLabel,omitempty"`
	PackagingType         string `json:"packagingType,omitempty"`
	StickerNo             string `json:"stickerNo,omitempty"`
	AllergenInfo          string `json:"allergenInfo,omitempty"`
	IsProductNameMatch    *bool  `json:"isProductNameMatch,omitempty"`
	IsBatchMatch          bool   `json:"isBatchMatch"`
	IsProductionDateMatch *bool  `json:"isProductionDateMatch,omitempty"`
	IsExpiryMatch         bool   `json:"isExpiryMatch"`
	IsOriginCountryMatch  *bool  `json:"isOriginCountryMatch,omitempty"`
	IsWeightMatch         *bool  `json:"isWeightMatch,omitempty"`
	IsPackagingMatch      *bool  `json:"isPackagingMatch,omitempty"`
	IsStickerNoMatch      *bool  `json:"isStickerNoMatch,omitempty"`
	IsAllergenInfoMatch   *bool  `json:"isAllergenInfoMatch,omitempty"`
	IsLineCheckPassed     *bool  `json:"isLineCheckPassed,omitempty"`
	IsCertLogoMatch       bool   `json:"isCertLogoMatch"`
	IsPrintQualityGood    bool   `json:"isPrintQualityGood"`
	IsPositionAligned     bool   `json:"isPositionAligned"`
	InspectedQty          int    `json:"inspectedQty"`
	PassedQty             int    `json:"passedQty"`
	RejectedQty           int    `json:"rejectedQty"`
	ItemDecision          string `json:"itemDecision"` // lolos, ditolak
	CorrectiveAction      string `json:"correctiveAction,omitempty"`
	QCPICName             string `json:"qcPicName,omitempty"`
	Notes                 string `json:"notes,omitempty"`
}

type CreatePackagingLabelCheckDTO struct {
	CheckDate            string                             `json:"checkDate"`
	CheckMonth           string                             `json:"checkMonth,omitempty"`
	HandoverReportID     *uuid.UUID                         `json:"handoverReportId,omitempty"`
	HandoverReportNo     string                             `json:"handoverReportNo,omitempty"`
	BatchNo              string                             `json:"batchNo,omitempty"`
	FairtradeStatus      string                             `json:"fairtradeStatus,omitempty"`
	Certification        string                             `json:"certification,omitempty"`
	QCInspectorName      string                             `json:"qcInspectorName"`
	QCInspectorSignDate  *string                            `json:"qcInspectorSignDate,omitempty"`
	QCSupervisorName     string                             `json:"qcSupervisorName,omitempty"`
	QCSupervisorSignDate *string                            `json:"qcSupervisorSignDate,omitempty"`
	OverallDecision      string                             `json:"overallDecision,omitempty"`
	Notes                string                             `json:"notes,omitempty"`
	Items                []CreatePackagingLabelCheckItemDTO `json:"items"`
}

type UpdatePackagingLabelCheckDTO struct {
	CheckDate            string                             `json:"checkDate,omitempty"`
	CheckMonth           string                             `json:"checkMonth,omitempty"`
	BatchNo              string                             `json:"batchNo,omitempty"`
	FairtradeStatus      string                             `json:"fairtradeStatus,omitempty"`
	Certification        string                             `json:"certification,omitempty"`
	QCInspectorName      string                             `json:"qcInspectorName,omitempty"`
	QCInspectorSignDate  *string                            `json:"qcInspectorSignDate,omitempty"`
	QCSupervisorName     string                             `json:"qcSupervisorName,omitempty"`
	QCSupervisorSignDate *string                            `json:"qcSupervisorSignDate,omitempty"`
	OverallDecision      string                             `json:"overallDecision,omitempty"`
	Notes                string                             `json:"notes,omitempty"`
	Items                []CreatePackagingLabelCheckItemDTO `json:"items,omitempty"`
}

func normalizePLKDecision(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "tidak", "ditolak", "rejected", "fail", "failed":
		return "ditolak"
	case "lolos", "passed", "pass":
		return "lolos"
	default:
		return ""
	}
}

func buildPLKItems(logID uuid.UUID, defaultCheckDate, defaultQCPIC string, dtoItems []CreatePackagingLabelCheckItemDTO) ([]domain.PackagingLabelCheckItem, bool) {
	items := make([]domain.PackagingLabelCheckItem, 0, len(dtoItems))
	hasRejected := false

	for i, it := range dtoItems {
		itemNo := it.ItemNo
		if itemNo <= 0 {
			itemNo = i + 1
		}
		itemCheckDate := strings.TrimSpace(it.ItemCheckDate)
		if itemCheckDate == "" {
			itemCheckDate = defaultCheckDate
		}
		itemCode := strings.TrimSpace(it.ItemCode)
		if itemCode == "" {
			itemCode = it.BatchNo
		}
		qcPicName := strings.TrimSpace(it.QCPICName)
		if qcPicName == "" {
			qcPicName = defaultQCPIC
		}

		prodNameOk := boolOrDefault(it.IsProductNameMatch, true)
		prodDateOk := boolOrDefault(it.IsProductionDateMatch, true)
		originOk := boolOrDefault(it.IsOriginCountryMatch, true)
		weightOk := boolOrDefault(it.IsWeightMatch, true)
		packOk := boolOrDefault(it.IsPackagingMatch, true)
		stickerOk := boolOrDefault(it.IsStickerNoMatch, true)
		allergenOk := boolOrDefault(it.IsAllergenInfoMatch, true)
		lineCheckOk := boolOrDefault(it.IsLineCheckPassed, true)

		allChecksOk := prodNameOk &&
			it.IsBatchMatch &&
			prodDateOk &&
			it.IsExpiryMatch &&
			originOk &&
			weightOk &&
			packOk &&
			stickerOk &&
			allergenOk &&
			lineCheckOk

		itemDecision := normalizePLKDecision(it.ItemDecision)
		if itemDecision == "" {
			if !allChecksOk || it.RejectedQty > 0 {
				itemDecision = "ditolak"
			} else {
				itemDecision = "lolos"
			}
		} else if !allChecksOk && itemDecision == "lolos" && it.RejectedQty > 0 {
			itemDecision = "ditolak"
		}
		if itemDecision == "ditolak" {
			hasRejected = true
		}

		items = append(items, domain.PackagingLabelCheckItem{
			BaseEntity:            types.BaseEntity{ID: uuid.New()},
			LogID:                 logID,
			ItemNo:                itemNo,
			ItemCheckDate:         itemCheckDate,
			ItemName:              it.ItemName,
			ItemCode:              itemCode,
			BatchNo:               it.BatchNo,
			BuyerName:             it.BuyerName,
			ProductionDate:        it.ProductionDate,
			ExpiredDate:           it.ExpiredDate,
			OriginCountry:         it.OriginCountry,
			WeightLabel:           it.WeightLabel,
			PackagingType:         it.PackagingType,
			StickerNo:             it.StickerNo,
			AllergenInfo:          it.AllergenInfo,
			IsProductNameMatch:    prodNameOk,
			IsBatchMatch:          it.IsBatchMatch,
			IsProductionDateMatch: prodDateOk,
			IsExpiryMatch:         it.IsExpiryMatch,
			IsOriginCountryMatch:  originOk,
			IsWeightMatch:         weightOk,
			IsPackagingMatch:      packOk,
			IsStickerNoMatch:      stickerOk,
			IsAllergenInfoMatch:   allergenOk,
			IsLineCheckPassed:     lineCheckOk,
			IsCertLogoMatch:       it.IsCertLogoMatch,
			IsPrintQualityGood:    it.IsPrintQualityGood,
			IsPositionAligned:     it.IsPositionAligned,
			InspectedQty:          it.InspectedQty,
			PassedQty:             it.PassedQty,
			RejectedQty:           it.RejectedQty,
			ItemDecision:          itemDecision,
			CorrectiveAction:      it.CorrectiveAction,
			QCPICName:             qcPicName,
			Notes:                 it.Notes,
		})
	}

	return items, hasRejected
}

func (uc *qcUseCase) CreatePackagingLabelCheck(ctx context.Context, dto CreatePackagingLabelCheckDTO) (*domain.PackagingLabelCheckLog, error) {
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal pemeriksaan wajib diisi")
	}
	if strings.TrimSpace(dto.QCInspectorName) == "" {
		return nil, apperrors.NewBadRequest("Nama pemeriksa QC wajib diisi")
	}

	month := dto.CheckMonth
	if month == "" && len(dto.CheckDate) >= 7 {
		month = dto.CheckDate[:7]
	}

	docNo, err := docseq.Next(ctx, uc.repo, "PLK", dto.CheckDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor dokumen PLK")
	}

	fairtrade := dto.FairtradeStatus
	if fairtrade == "" {
		fairtrade = "NON FAIRTRADE"
	}
	cert := dto.Certification
	if cert == "" {
		cert = "NATURAL"
	}

	logID := uuid.New()
	items, hasRejected := buildPLKItems(logID, dto.CheckDate, dto.QCInspectorName, dto.Items)

	batchNo := strings.TrimSpace(dto.BatchNo)
	if batchNo == "" && len(items) > 0 {
		batchNo = items[0].BatchNo
	}

	overall := normalizePLKDecision(dto.OverallDecision)
	if overall == "" {
		if hasRejected {
			overall = "ditolak"
		} else {
			overall = "lolos"
		}
	}

	log := &domain.PackagingLabelCheckLog{
		BaseEntity:           types.BaseEntity{ID: logID, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		DocumentNo:           docNo,
		DocumentCode:         "QC-016-PLK",
		RevisionCode:         "01",
		CheckDate:            dto.CheckDate,
		CheckMonth:           month,
		HandoverReportID:     dto.HandoverReportID,
		HandoverReportNo:     dto.HandoverReportNo,
		BatchNo:              batchNo,
		FairtradeStatus:      fairtrade,
		Certification:        cert,
		QCInspectorName:      dto.QCInspectorName,
		QCInspectorSignDate:  dto.QCInspectorSignDate,
		QCSupervisorName:     dto.QCSupervisorName,
		QCSupervisorSignDate: dto.QCSupervisorSignDate,
		OverallDecision:      overall,
		Notes:                dto.Notes,
		Items:                items,
	}

	if err := uc.repo.CreatePackagingLabelCheck(ctx, log); err != nil {
		return nil, fmt.Errorf("gagal membuat laporan PLK: %w", err)
	}

	if overall == "ditolak" || hasRejected {
		var totalRejQty float64
		prodName := "Bahan Kemas Berlabel"
		if len(items) > 0 {
			prodName = items[0].ItemName
			for _, it := range items {
				totalRejQty += float64(it.RejectedQty)
			}
		}
		if totalRejQty <= 0 {
			totalRejQty = 1.0
		}
		defect := "Pemeriksaan label kemasan (QC-016-PLK) ditolak"
		if log.Notes != "" {
			defect += ": " + log.Notes
		}
		_, _ = uc.AutoCreateBrontolOnQCFailure(ctx, AutoBrontolTrigger{
			SourceStage:        domain.StagePackagingLabel,
			SourceInspectionID: &log.ID,
			SourceInspectionNo: log.DocumentNo,
			ProductName:        prodName,
			BatchNo:            log.BatchNo,
			Quantity:           totalRejQty,
			Unit:               "PCS",
			MaterialType:       "BK",
			DefectReason:       defect,
			ActionDisposition:  domain.ActionReturnToSupplier,
			QCInspectorName:    log.QCInspectorName,
			Notes:              log.Notes,
		})
	}

	return log, nil
}

func (uc *qcUseCase) GetPackagingLabelCheckByID(ctx context.Context, id uuid.UUID) (*domain.PackagingLabelCheckLog, error) {
	return uc.repo.GetPackagingLabelCheckByID(ctx, id)
}

func (uc *qcUseCase) ListPackagingLabelChecks(ctx context.Context, month string, batchNo string, decision string) ([]domain.PackagingLabelCheckLog, error) {
	return uc.repo.ListPackagingLabelChecks(ctx, month, batchNo, decision)
}

func (uc *qcUseCase) UpdatePackagingLabelCheck(ctx context.Context, id uuid.UUID, dto UpdatePackagingLabelCheckDTO) (*domain.PackagingLabelCheckLog, error) {
	existing, err := uc.repo.GetPackagingLabelCheckByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.DocumentCode == "" {
		existing.DocumentCode = "QC-016-PLK"
	}
	if existing.RevisionCode == "" {
		existing.RevisionCode = "01"
	}
	if dto.CheckDate != "" {
		existing.CheckDate = dto.CheckDate
		if dto.CheckMonth != "" {
			existing.CheckMonth = dto.CheckMonth
		} else if len(dto.CheckDate) >= 7 {
			existing.CheckMonth = dto.CheckDate[:7]
		}
	}
	if dto.BatchNo != "" {
		existing.BatchNo = dto.BatchNo
	}
	if dto.FairtradeStatus != "" {
		existing.FairtradeStatus = dto.FairtradeStatus
	}
	if dto.Certification != "" {
		existing.Certification = dto.Certification
	}
	if dto.QCInspectorName != "" {
		existing.QCInspectorName = dto.QCInspectorName
	}
	if dto.QCInspectorSignDate != nil {
		existing.QCInspectorSignDate = dto.QCInspectorSignDate
	}
	if dto.QCSupervisorName != "" {
		existing.QCSupervisorName = dto.QCSupervisorName
	}
	if dto.QCSupervisorSignDate != nil {
		existing.QCSupervisorSignDate = dto.QCSupervisorSignDate
	}
	if dto.Notes != "" {
		existing.Notes = dto.Notes
	}

	hasRejected := false
	if len(dto.Items) > 0 {
		existing.Items, hasRejected = buildPLKItems(existing.ID, existing.CheckDate, existing.QCInspectorName, dto.Items)
		if existing.BatchNo == "" && len(existing.Items) > 0 {
			existing.BatchNo = existing.Items[0].BatchNo
		}
	}

	if norm := normalizePLKDecision(dto.OverallDecision); norm != "" {
		existing.OverallDecision = norm
	} else if len(dto.Items) > 0 {
		if hasRejected {
			existing.OverallDecision = "ditolak"
		} else {
			existing.OverallDecision = "lolos"
		}
	}
	existing.UpdatedAt = time.Now()

	if err := uc.repo.UpdatePackagingLabelCheck(ctx, existing); err != nil {
		return nil, fmt.Errorf("gagal memperbarui laporan PLK: %w", err)
	}

	return existing, nil
}

func (uc *qcUseCase) DeletePackagingLabelCheck(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeletePackagingLabelCheck(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// CCO: Container Checklist (Form QC-008-CCO.02)
// ─────────────────────────────────────────────────────────────────────────────

type CreateContainerChecklistDTO struct {
	CheckDate             string  `json:"checkDate"`
	CheckMonth            string  `json:"checkMonth,omitempty"`
	IDCardType            string  `json:"idCardType"` // KTP / SIM
	IDCardNumber          string  `json:"idCardNumber"`
	DriverName            string  `json:"driverName"`
	BirthDetails          string  `json:"birthDetails,omitempty"`
	Gender                string  `json:"gender,omitempty"`
	Address               string  `json:"address,omitempty"`
	Religion              string  `json:"religion,omitempty"`
	MaritalStatus         string  `json:"maritalStatus,omitempty"`
	CardValidity          string  `json:"cardValidity,omitempty"`
	TransporterCompany    string  `json:"transporterCompany"`
	VehicleLicensePlate   string  `json:"vehicleLicensePlate"`
	ContainerName         string  `json:"containerName"`
	ContainerNumber       string  `json:"containerNumber"`
	ContainerSize         string  `json:"containerSize"` // 20ft, 40ft, 40ft HC
	SealNumber            string  `json:"sealNumber"`
	ShippingInstructionNo string  `json:"shippingInstructionNo,omitempty"`
	EIRNumber             string  `json:"eirNumber,omitempty"`
	ERNumber              string  `json:"erNumber,omitempty"`
	CabinAreaStatus       string  `json:"cabinAreaStatus"`
	UnderVehicleStatus    string  `json:"underVehicleStatus"`
	InsideContainerStatus string  `json:"insideContainerStatus"`
	SuspiciousNotes       string  `json:"suspiciousNotes,omitempty"`
	FloorCleanliness      string  `json:"floorCleanliness"`
	RightWallCleanliness  string  `json:"rightWallCleanliness"`
	LeftWallCleanliness   string  `json:"leftWallCleanliness"`
	RoofCleanliness       string  `json:"roofCleanliness"`
	DoorCleanliness       string  `json:"doorCleanliness"`
	CleanlinessNotes      string  `json:"cleanlinessNotes,omitempty"`
	FloorHoleStatus       string  `json:"floorHoleStatus"`
	RightWallHoleStatus   string  `json:"rightWallHoleStatus"`
	LeftWallHoleStatus    string  `json:"leftWallHoleStatus"`
	RoofHoleStatus        string  `json:"roofHoleStatus"`
	DoorHoleStatus        string  `json:"doorHoleStatus"`
	HoleNotes             string  `json:"holeNotes,omitempty"`
	SealTime              string  `json:"sealTime,omitempty"`
	CapTime               string  `json:"capTime,omitempty"`
	DepartureTime         string  `json:"departureTime,omitempty"`
	PalletDetails         string  `json:"palletDetails,omitempty"`
	ShipmentDestination   string  `json:"shipmentDestination"`
	InspectorQC           string  `json:"inspectorQc"`
	QCSignDate            *string `json:"qcSignDate,omitempty"`
	VerifierQA            string  `json:"verifierQa"`
	QASignDate            *string `json:"qaSignDate,omitempty"`
	OverallDecision       string  `json:"overallDecision,omitempty"`
	Notes                 string  `json:"notes,omitempty"`
}

type UpdateContainerChecklistDTO struct {
	CheckDate             string  `json:"checkDate,omitempty"`
	CheckMonth            string  `json:"checkMonth,omitempty"`
	IDCardType            string  `json:"idCardType,omitempty"`
	IDCardNumber          string  `json:"idCardNumber,omitempty"`
	DriverName            string  `json:"driverName,omitempty"`
	BirthDetails          string  `json:"birthDetails,omitempty"`
	Gender                string  `json:"gender,omitempty"`
	Address               string  `json:"address,omitempty"`
	Religion              string  `json:"religion,omitempty"`
	MaritalStatus         string  `json:"maritalStatus,omitempty"`
	CardValidity          string  `json:"cardValidity,omitempty"`
	TransporterCompany    string  `json:"transporterCompany,omitempty"`
	VehicleLicensePlate   string  `json:"vehicleLicensePlate,omitempty"`
	ContainerName         string  `json:"containerName,omitempty"`
	ContainerNumber       string  `json:"containerNumber,omitempty"`
	ContainerSize         string  `json:"containerSize,omitempty"`
	SealNumber            string  `json:"sealNumber,omitempty"`
	ShippingInstructionNo string  `json:"shippingInstructionNo,omitempty"`
	EIRNumber             string  `json:"eirNumber,omitempty"`
	ERNumber              string  `json:"erNumber,omitempty"`
	CabinAreaStatus       string  `json:"cabinAreaStatus,omitempty"`
	UnderVehicleStatus    string  `json:"underVehicleStatus,omitempty"`
	InsideContainerStatus string  `json:"insideContainerStatus,omitempty"`
	SuspiciousNotes       string  `json:"suspiciousNotes,omitempty"`
	FloorCleanliness      string  `json:"floorCleanliness,omitempty"`
	RightWallCleanliness  string  `json:"rightWallCleanliness,omitempty"`
	LeftWallCleanliness   string  `json:"leftWallCleanliness,omitempty"`
	RoofCleanliness       string  `json:"roofCleanliness,omitempty"`
	DoorCleanliness       string  `json:"doorCleanliness,omitempty"`
	CleanlinessNotes      string  `json:"cleanlinessNotes,omitempty"`
	FloorHoleStatus       string  `json:"floorHoleStatus,omitempty"`
	RightWallHoleStatus   string  `json:"rightWallHoleStatus,omitempty"`
	LeftWallHoleStatus    string  `json:"leftWallHoleStatus,omitempty"`
	RoofHoleStatus        string  `json:"roofHoleStatus,omitempty"`
	DoorHoleStatus        string  `json:"doorHoleStatus,omitempty"`
	HoleNotes             string  `json:"holeNotes,omitempty"`
	SealTime              string  `json:"sealTime,omitempty"`
	CapTime               string  `json:"capTime,omitempty"`
	DepartureTime         string  `json:"departureTime,omitempty"`
	PalletDetails         string  `json:"palletDetails,omitempty"`
	ShipmentDestination   string  `json:"shipmentDestination,omitempty"`
	InspectorQC           string  `json:"inspectorQc,omitempty"`
	QCSignDate            *string `json:"qcSignDate,omitempty"`
	VerifierQA            string  `json:"verifierQa,omitempty"`
	QASignDate            *string `json:"qaSignDate,omitempty"`
	OverallDecision       string  `json:"overallDecision,omitempty"`
	Notes                 string  `json:"notes,omitempty"`
}

func (uc *qcUseCase) CreateContainerChecklist(ctx context.Context, dto CreateContainerChecklistDTO) (*domain.ContainerChecklistLog, error) {
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal checklist kontainer wajib diisi")
	}
	if strings.TrimSpace(dto.ContainerNumber) == "" {
		return nil, apperrors.NewBadRequest("Nomor kontainer wajib diisi")
	}
	if strings.TrimSpace(dto.DriverName) == "" || strings.TrimSpace(dto.VehicleLicensePlate) == "" {
		return nil, apperrors.NewBadRequest("Identitas pengemudi dan nomor polisi kendaraan wajib diisi")
	}
	if strings.TrimSpace(dto.InspectorQC) == "" {
		return nil, apperrors.NewBadRequest("Nama pemeriksa QC wajib diisi")
	}

	month := dto.CheckMonth
	if month == "" && len(dto.CheckDate) >= 7 {
		month = dto.CheckDate[:7]
	}

	docNo, err := docseq.Next(ctx, uc.repo, "CCO", dto.CheckDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor dokumen CCO")
	}

	// Decision evaluation: if any suspicious item, or dirty, or leak, mark as ditolak unless overridden
	overall := dto.OverallDecision
	if overall == "" {
		hasSuspicious := dto.CabinAreaStatus == "mencurigakan" || dto.UnderVehicleStatus == "mencurigakan" || dto.InsideContainerStatus == "mencurigakan"
		hasDirt := dto.FloorCleanliness == "kotor" || dto.RightWallCleanliness == "kotor" || dto.LeftWallCleanliness == "kotor" || dto.RoofCleanliness == "kotor" || dto.DoorCleanliness == "kotor"
		hasLeak := dto.FloorHoleStatus == "bocor" || dto.RightWallHoleStatus == "bocor" || dto.LeftWallHoleStatus == "bocor" || dto.RoofHoleStatus == "bocor" || dto.DoorHoleStatus == "bocor"

		if hasSuspicious || hasDirt || hasLeak {
			overall = "ditolak"
		} else {
			overall = "lolos"
		}
	}

	log := &domain.ContainerChecklistLog{
		BaseEntity:            types.BaseEntity{ID: uuid.New()},
		DocumentNo:            docNo,
		CheckDate:             dto.CheckDate,
		CheckMonth:            month,
		IDCardType:            dto.IDCardType,
		IDCardNumber:          dto.IDCardNumber,
		DriverName:            dto.DriverName,
		BirthDetails:          dto.BirthDetails,
		Gender:                dto.Gender,
		Address:               dto.Address,
		Religion:              dto.Religion,
		MaritalStatus:         dto.MaritalStatus,
		CardValidity:          dto.CardValidity,
		TransporterCompany:    dto.TransporterCompany,
		VehicleLicensePlate:   dto.VehicleLicensePlate,
		ContainerName:         dto.ContainerName,
		ContainerNumber:       dto.ContainerNumber,
		ContainerSize:         dto.ContainerSize,
		SealNumber:            dto.SealNumber,
		ShippingInstructionNo: dto.ShippingInstructionNo,
		EIRNumber:             dto.EIRNumber,
		ERNumber:              dto.ERNumber,
		CabinAreaStatus:       dto.CabinAreaStatus,
		UnderVehicleStatus:    dto.UnderVehicleStatus,
		InsideContainerStatus: dto.InsideContainerStatus,
		SuspiciousNotes:       dto.SuspiciousNotes,
		FloorCleanliness:      dto.FloorCleanliness,
		RightWallCleanliness:  dto.RightWallCleanliness,
		LeftWallCleanliness:   dto.LeftWallCleanliness,
		RoofCleanliness:       dto.RoofCleanliness,
		DoorCleanliness:       dto.DoorCleanliness,
		CleanlinessNotes:      dto.CleanlinessNotes,
		FloorHoleStatus:       dto.FloorHoleStatus,
		RightWallHoleStatus:   dto.RightWallHoleStatus,
		LeftWallHoleStatus:    dto.LeftWallHoleStatus,
		RoofHoleStatus:        dto.RoofHoleStatus,
		DoorHoleStatus:        dto.DoorHoleStatus,
		HoleNotes:             dto.HoleNotes,
		SealTime:              dto.SealTime,
		CapTime:               dto.CapTime,
		DepartureTime:         dto.DepartureTime,
		PalletDetails:         dto.PalletDetails,
		ShipmentDestination:   dto.ShipmentDestination,
		InspectorQC:           dto.InspectorQC,
		QCSignDate:            dto.QCSignDate,
		VerifierQA:            dto.VerifierQA,
		QASignDate:            dto.QASignDate,
		OverallDecision:       overall,
		Notes:                 dto.Notes,
	}

	if err := uc.repo.CreateContainerChecklist(ctx, log); err != nil {
		return nil, fmt.Errorf("gagal membuat checklist container CCO: %w", err)
	}

	return log, nil
}

func (uc *qcUseCase) GetContainerChecklistByID(ctx context.Context, id uuid.UUID) (*domain.ContainerChecklistLog, error) {
	return uc.repo.GetContainerChecklistByID(ctx, id)
}

func (uc *qcUseCase) ListContainerChecklists(ctx context.Context, month string, containerNo string, destination string, decision string) ([]domain.ContainerChecklistLog, error) {
	return uc.repo.ListContainerChecklists(ctx, month, containerNo, destination, decision)
}

func (uc *qcUseCase) UpdateContainerChecklist(ctx context.Context, id uuid.UUID, dto UpdateContainerChecklistDTO) (*domain.ContainerChecklistLog, error) {
	existing, err := uc.repo.GetContainerChecklistByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if dto.CheckDate != "" {
		existing.CheckDate = dto.CheckDate
		if dto.CheckMonth != "" {
			existing.CheckMonth = dto.CheckMonth
		} else if len(dto.CheckDate) >= 7 {
			existing.CheckMonth = dto.CheckDate[:7]
		}
	}
	if dto.IDCardType != "" {
		existing.IDCardType = dto.IDCardType
	}
	if dto.IDCardNumber != "" {
		existing.IDCardNumber = dto.IDCardNumber
	}
	if dto.DriverName != "" {
		existing.DriverName = dto.DriverName
	}
	if dto.BirthDetails != "" {
		existing.BirthDetails = dto.BirthDetails
	}
	if dto.Gender != "" {
		existing.Gender = dto.Gender
	}
	if dto.Address != "" {
		existing.Address = dto.Address
	}
	if dto.Religion != "" {
		existing.Religion = dto.Religion
	}
	if dto.MaritalStatus != "" {
		existing.MaritalStatus = dto.MaritalStatus
	}
	if dto.CardValidity != "" {
		existing.CardValidity = dto.CardValidity
	}
	if dto.TransporterCompany != "" {
		existing.TransporterCompany = dto.TransporterCompany
	}
	if dto.VehicleLicensePlate != "" {
		existing.VehicleLicensePlate = dto.VehicleLicensePlate
	}
	if dto.ContainerName != "" {
		existing.ContainerName = dto.ContainerName
	}
	if dto.ContainerNumber != "" {
		existing.ContainerNumber = dto.ContainerNumber
	}
	if dto.ContainerSize != "" {
		existing.ContainerSize = dto.ContainerSize
	}
	if dto.SealNumber != "" {
		existing.SealNumber = dto.SealNumber
	}
	if dto.ShippingInstructionNo != "" {
		existing.ShippingInstructionNo = dto.ShippingInstructionNo
	}
	if dto.EIRNumber != "" {
		existing.EIRNumber = dto.EIRNumber
	}
	if dto.ERNumber != "" {
		existing.ERNumber = dto.ERNumber
	}
	if dto.CabinAreaStatus != "" {
		existing.CabinAreaStatus = dto.CabinAreaStatus
	}
	if dto.UnderVehicleStatus != "" {
		existing.UnderVehicleStatus = dto.UnderVehicleStatus
	}
	if dto.InsideContainerStatus != "" {
		existing.InsideContainerStatus = dto.InsideContainerStatus
	}
	if dto.SuspiciousNotes != "" {
		existing.SuspiciousNotes = dto.SuspiciousNotes
	}
	if dto.FloorCleanliness != "" {
		existing.FloorCleanliness = dto.FloorCleanliness
	}
	if dto.RightWallCleanliness != "" {
		existing.RightWallCleanliness = dto.RightWallCleanliness
	}
	if dto.LeftWallCleanliness != "" {
		existing.LeftWallCleanliness = dto.LeftWallCleanliness
	}
	if dto.RoofCleanliness != "" {
		existing.RoofCleanliness = dto.RoofCleanliness
	}
	if dto.DoorCleanliness != "" {
		existing.DoorCleanliness = dto.DoorCleanliness
	}
	if dto.CleanlinessNotes != "" {
		existing.CleanlinessNotes = dto.CleanlinessNotes
	}
	if dto.FloorHoleStatus != "" {
		existing.FloorHoleStatus = dto.FloorHoleStatus
	}
	if dto.RightWallHoleStatus != "" {
		existing.RightWallHoleStatus = dto.RightWallHoleStatus
	}
	if dto.LeftWallHoleStatus != "" {
		existing.LeftWallHoleStatus = dto.LeftWallHoleStatus
	}
	if dto.RoofHoleStatus != "" {
		existing.RoofHoleStatus = dto.RoofHoleStatus
	}
	if dto.DoorHoleStatus != "" {
		existing.DoorHoleStatus = dto.DoorHoleStatus
	}
	if dto.HoleNotes != "" {
		existing.HoleNotes = dto.HoleNotes
	}
	if dto.SealTime != "" {
		existing.SealTime = dto.SealTime
	}
	if dto.CapTime != "" {
		existing.CapTime = dto.CapTime
	}
	if dto.DepartureTime != "" {
		existing.DepartureTime = dto.DepartureTime
	}
	if dto.PalletDetails != "" {
		existing.PalletDetails = dto.PalletDetails
	}
	if dto.ShipmentDestination != "" {
		existing.ShipmentDestination = dto.ShipmentDestination
	}
	if dto.InspectorQC != "" {
		existing.InspectorQC = dto.InspectorQC
	}
	if dto.QCSignDate != nil {
		existing.QCSignDate = dto.QCSignDate
	}
	if dto.VerifierQA != "" {
		existing.VerifierQA = dto.VerifierQA
	}
	if dto.QASignDate != nil {
		existing.QASignDate = dto.QASignDate
	}
	if dto.OverallDecision != "" {
		existing.OverallDecision = dto.OverallDecision
	}
	if dto.Notes != "" {
		existing.Notes = dto.Notes
	}
	existing.UpdatedAt = time.Now()

	if err := uc.repo.UpdateContainerChecklist(ctx, existing); err != nil {
		return nil, fmt.Errorf("gagal memperbarui checklist container CCO: %w", err)
	}

	return existing, nil
}

func (uc *qcUseCase) DeleteContainerChecklist(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteContainerChecklist(ctx, id)
}
