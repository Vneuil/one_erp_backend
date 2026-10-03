package application

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/collaboration/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

var exportRand = rand.New(rand.NewSource(time.Now().UnixNano()))

var validExportFormats = map[string]bool{
	"csv":   true,
	"excel": true,
}

var placeholderColumns = map[string][]string{
	"sales_orders":      {"Order No", "Customer", "Order Date", "Total Amount", "Status"},
	"inventory_stock":   {"SKU", "Product Name", "Warehouse", "Qty On Hand", "Unit"},
	"employees":         {"Employee No", "Name", "Department", "Position", "Join Date"},
	"purchase_orders":   {"PO No", "Supplier", "Order Date", "Total Amount", "Status"},
	"journal_entries":   {"Entry No", "Date", "Account", "Debit", "Credit"},
	"fixed_assets":      {"Asset No", "Name", "Category", "Acquisition Value", "Book Value"},
	"support_tickets":   {"Ticket No", "Subject", "Customer", "Priority", "Status"},
	"production_orders": {"SPK No", "Product", "Quantity", "Start Date", "Status"},
}

func defaultColumns() []string {
	return []string{"ID", "Name", "Value", "Date", "Status"}
}

type ExportsUseCase interface {
	CreateJob(ctx context.Context, dto CreateExportJobDTO) (*ExportJobResponseDTO, error)
	ListJobs(ctx context.Context) ([]ExportJobResponseDTO, error)
	RunJob(ctx context.Context, id uuid.UUID) (*ExportJobResponseDTO, error)
	DownloadJob(ctx context.Context, id uuid.UUID) (*ExportJobDownloadDTO, error)

	SeedInitialData(ctx context.Context) error
}

type exportsUseCase struct {
	repo domain.ExportsRepository
}

func NewExportsUseCase(repo domain.ExportsRepository) ExportsUseCase {
	return &exportsUseCase{repo: repo}
}

func (uc *exportsUseCase) CreateJob(ctx context.Context, dto CreateExportJobDTO) (*ExportJobResponseDTO, error) {
	if dto.DatasetName == "" {
		return nil, apperrors.NewBadRequest("Dataset name is required")
	}
	if !validExportFormats[dto.Format] {
		return nil, apperrors.NewBadRequest("Format must be one of csv, excel")
	}
	if dto.RequestedBy == "" {
		return nil, apperrors.NewBadRequest("Requested by is required")
	}

	j := &domain.ExportJob{
		DatasetName: dto.DatasetName,
		Format:      dto.Format,
		Status:      "pending",
		RequestedBy: dto.RequestedBy,
	}
	if err := uc.repo.CreateJob(ctx, j); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create export job")
	}
	return ToExportJobResponse(j), nil
}

func (uc *exportsUseCase) ListJobs(ctx context.Context) ([]ExportJobResponseDTO, error) {
	items, err := uc.repo.ListJobs(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list export jobs")
	}
	return ToExportJobResponseList(items), nil
}

func (uc *exportsUseCase) RunJob(ctx context.Context, id uuid.UUID) (*ExportJobResponseDTO, error) {
	j, err := uc.repo.GetJobByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get export job")
	}
	if j == nil {
		return nil, apperrors.NewNotFound("Export job not found")
	}

	rows := int64(50 + exportRand.Intn(4000))
	j.Status = "completed"
	j.RowCount = &rows
	now := time.Now()
	j.CompletedAt = &now

	if err := uc.repo.UpdateJob(ctx, j); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to run export job")
	}
	return ToExportJobResponse(j), nil
}

func (uc *exportsUseCase) DownloadJob(ctx context.Context, id uuid.UUID) (*ExportJobDownloadDTO, error) {
	j, err := uc.repo.GetJobByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get export job")
	}
	if j == nil {
		return nil, apperrors.NewNotFound("Export job not found")
	}
	if j.Status != "completed" {
		return nil, apperrors.NewBadRequest("Export job has not completed yet")
	}

	cols, ok := placeholderColumns[j.DatasetName]
	if !ok {
		cols = defaultColumns()
	}

	var sb strings.Builder
	sb.WriteString(strings.Join(cols, ","))
	sb.WriteString("\n")
	rowCount := 3
	for i := 1; i <= rowCount; i++ {
		row := make([]string, len(cols))
		for c := range cols {
			row[c] = fmt.Sprintf("%s-%03d", j.DatasetName, i)
		}
		sb.WriteString(strings.Join(row, ","))
		sb.WriteString("\n")
	}

	return &ExportJobDownloadDTO{
		ExportJobResponseDTO: *ToExportJobResponse(j),
		Content:              sb.String(),
	}, nil
}

// SeedInitialData populates a small export job history on first boot
func (uc *exportsUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountJobs(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	seeds := []struct {
		dto CreateExportJobDTO
		run bool
	}{
		{dto: CreateExportJobDTO{DatasetName: "sales_orders", Format: "excel", RequestedBy: "Nicholas Tantra"}, run: true},
		{dto: CreateExportJobDTO{DatasetName: "inventory_stock", Format: "csv", RequestedBy: "Budi Santoso"}, run: true},
		{dto: CreateExportJobDTO{DatasetName: "employees", Format: "excel", RequestedBy: "Dewi Lestari"}, run: false},
	}

	for _, seed := range seeds {
		res, err := uc.CreateJob(ctx, seed.dto)
		if err != nil {
			continue
		}
		if seed.run {
			if _, err := uc.RunJob(ctx, res.ID); err != nil {
				continue
			}
		}
	}
	return nil
}
