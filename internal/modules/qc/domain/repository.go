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
}
