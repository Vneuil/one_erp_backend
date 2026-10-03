package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/activitylog/domain"
	"github.com/google/uuid"
)

type ActivityLogResponseDTO struct {
	ID         uuid.UUID `json:"id"`
	UserEmail  string    `json:"userEmail"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Module     string    `json:"module"`
	StatusCode int       `json:"statusCode"`
	IPAddress  string    `json:"ipAddress,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

func ToActivityLogResponse(l *domain.ActivityLog) ActivityLogResponseDTO {
	return ActivityLogResponseDTO{
		ID:         l.ID,
		UserEmail:  l.UserEmail,
		Method:     l.Method,
		Path:       l.Path,
		Module:     l.Module,
		StatusCode: l.StatusCode,
		IPAddress:  l.IPAddress,
		CreatedAt:  l.CreatedAt,
	}
}

func ToActivityLogResponseList(items []domain.ActivityLog) []ActivityLogResponseDTO {
	result := make([]ActivityLogResponseDTO, len(items))
	for i, l := range items {
		result[i] = ToActivityLogResponse(&l)
	}
	return result
}
