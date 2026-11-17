package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ListInspectionFilter struct {
	TargetType        *TargetType
	TargetID          *uuid.UUID
	ProductionOrderID *uuid.UUID
	ProductionStepID  *uuid.UUID
	BatchNo           string
	ProductID         *uuid.UUID
	Status            *InspectionStatus
	StartDate         *time.Time
	EndDate           *time.Time
	Page              int
	PerPage           int
}

type QCRepository interface {
	// Parameter
	CreateParameter(ctx context.Context, param *QCParameter) error
	GetParameterByID(ctx context.Context, id uuid.UUID) (*QCParameter, error)
	ListParameters(ctx context.Context, productID *uuid.UUID, processCode string) ([]QCParameter, error)
	UpdateParameter(ctx context.Context, param *QCParameter) error
	DeleteParameter(ctx context.Context, id uuid.UUID) error

	// Inspection
	CreateInspection(ctx context.Context, insp *QCInspection) error
	GetInspectionByID(ctx context.Context, id uuid.UUID) (*QCInspection, error)
	GetInspectionByNo(ctx context.Context, inspectionNo string) (*QCInspection, error)
	ListInspections(ctx context.Context, filter ListInspectionFilter) ([]QCInspection, int64, error)
	UpdateInspection(ctx context.Context, insp *QCInspection) error
	GetLatestPassedInspection(ctx context.Context, targetType TargetType, targetID uuid.UUID, processCode string) (*QCInspection, error)
	GetInspectionCountToday(ctx context.Context, datePrefix string) (int64, error)

	// Stats / Reports
	GetInspectionStats(ctx context.Context) (total int64, passed int64, failed int64, voided int64, err error)

	// Hygiene (CPH)
	CreateHygieneCheck(ctx context.Context, check *PersonalHygieneCheck) error
	ListHygieneChecks(ctx context.Context, date string, line string) ([]PersonalHygieneCheck, error)
	DeleteHygieneCheck(ctx context.Context, id uuid.UUID) error

	// Pre-Op Sanitation (SSO)
	CreatePreOpCheck(ctx context.Context, check *PreOpSanitationCheck) error
	ListPreOpChecks(ctx context.Context, date string, room string) ([]PreOpSanitationCheck, error)
	DeletePreOpCheck(ctx context.Context, id uuid.UUID) error

	// Packaging Quality Check (PBK — QC-003)
	CreatePackagingCheck(ctx context.Context, check *PackagingQualityCheck) error
	GetPackagingCheckByID(ctx context.Context, id uuid.UUID) (*PackagingQualityCheck, error)
	ListPackagingChecks(ctx context.Context, month string, lbmNo string) ([]PackagingQualityCheck, error)
	UpdatePackagingCheck(ctx context.Context, check *PackagingQualityCheck) error
	DeletePackagingCheck(ctx context.Context, id uuid.UUID) error

	// Supplier Armada Check (CAS — QC-007)
	CreateArmadaCheck(ctx context.Context, check *SupplierArmadaCheck) error
	GetArmadaCheckByID(ctx context.Context, id uuid.UUID) (*SupplierArmadaCheck, error)
	ListArmadaChecks(ctx context.Context, date string, lbmNo string) ([]SupplierArmadaCheck, error)
	UpdateArmadaCheck(ctx context.Context, check *SupplierArmadaCheck) error
	DeleteArmadaCheck(ctx context.Context, id uuid.UUID) error

	// Raw Material Incoming Check (PBD — QC-001)
	CreateIncomingCheck(ctx context.Context, check *RawMaterialIncomingCheck) error
	GetIncomingCheckByID(ctx context.Context, id uuid.UUID) (*RawMaterialIncomingCheck, error)
	ListIncomingChecks(ctx context.Context, date string, lbmNo string) ([]RawMaterialIncomingCheck, error)
	UpdateIncomingCheck(ctx context.Context, check *RawMaterialIncomingCheck) error
	DeleteIncomingCheck(ctx context.Context, id uuid.UUID) error
}
