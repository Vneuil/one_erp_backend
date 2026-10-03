package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/collaboration/domain"
	"github.com/google/uuid"
)

type CreateExportJobDTO struct {
	DatasetName string `json:"datasetName"`
	Format      string `json:"format"`
	RequestedBy string `json:"requestedBy"`
}

type ExportJobResponseDTO struct {
	ID          uuid.UUID  `json:"id"`
	DatasetName string     `json:"datasetName"`
	Format      string     `json:"format"`
	Status      string     `json:"status"`
	RequestedBy string     `json:"requestedBy"`
	RowCount    *int64     `json:"rowCount,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type ExportJobDownloadDTO struct {
	ExportJobResponseDTO
	Content string `json:"content"`
}

func ToExportJobResponse(j *domain.ExportJob) *ExportJobResponseDTO {
	if j == nil {
		return nil
	}
	return &ExportJobResponseDTO{
		ID:          j.ID,
		DatasetName: j.DatasetName,
		Format:      j.Format,
		Status:      j.Status,
		RequestedBy: j.RequestedBy,
		RowCount:    j.RowCount,
		CompletedAt: j.CompletedAt,
		CreatedAt:   j.CreatedAt,
	}
}

func ToExportJobResponseList(items []domain.ExportJob) []ExportJobResponseDTO {
	result := make([]ExportJobResponseDTO, len(items))
	for i, j := range items {
		result[i] = *ToExportJobResponse(&j)
	}
	return result
}
