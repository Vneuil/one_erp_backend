package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Goal represents an individual or team target tracked over a period
type Goal struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this goal to one business unit within the company
	// (see modules/workspace). Nil means the company's default tenant.
	TenantID        *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Title           string     `gorm:"type:varchar(255);not null" json:"title"`
	Description     string     `gorm:"type:text" json:"description"`
	GoalType        string     `gorm:"type:varchar(20);not null" json:"goalType"`
	OwnerName       string     `gorm:"type:varchar(255);not null" json:"ownerName"`
	Category        string     `gorm:"type:varchar(100);not null" json:"category"`
	TargetValue     float64    `gorm:"type:decimal(15,2);not null" json:"targetValue"`
	CurrentValue    float64    `gorm:"type:decimal(15,2);not null;default:0" json:"currentValue"`
	Unit            string     `gorm:"type:varchar(50)" json:"unit"`
	PeriodStart     string     `gorm:"type:varchar(50);not null" json:"periodStart"`
	PeriodEnd       string     `gorm:"type:varchar(50);not null" json:"periodEnd"`
	Status          string     `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	ProgressPercent float64    `gorm:"type:decimal(5,2);not null;default:0" json:"progressPercent"`
}

func (Goal) TableName() string {
	return "goal_goals"
}

// GoalCheckIn represents a periodic update recorded against a goal
type GoalCheckIn struct {
	types.BaseEntity
	GoalID        uuid.UUID `gorm:"type:uuid;not null;index" json:"goalId"`
	ValueRecorded float64   `gorm:"type:decimal(15,2);not null" json:"valueRecorded"`
	Note          string    `gorm:"type:text" json:"note"`
	CheckedInDate string    `gorm:"type:varchar(50);not null" json:"checkedInDate"`
}

func (GoalCheckIn) TableName() string {
	return "goal_check_ins"
}

type GoalSummary struct {
	TotalGoals      int64   `json:"totalGoals"`
	ActiveGoals     int64   `json:"activeGoals"`
	CompletedGoals  int64   `json:"completedGoals"`
	MissedGoals     int64   `json:"missedGoals"`
	CancelledGoals  int64   `json:"cancelledGoals"`
	AverageProgress float64 `json:"averageProgress"`
}

type GoalsRepository interface {
	CreateGoal(ctx context.Context, g *Goal) error
	GetGoalByID(ctx context.Context, id uuid.UUID) (*Goal, error)
	ListGoals(ctx context.Context, query types.PaginationQuery) ([]Goal, int64, error)
	ListAllGoals(ctx context.Context) ([]Goal, error)
	UpdateGoal(ctx context.Context, g *Goal) error
	CountGoals(ctx context.Context) (int64, error)

	CreateCheckIn(ctx context.Context, ci *GoalCheckIn) error
	ListCheckInsByGoal(ctx context.Context, goalID uuid.UUID) ([]GoalCheckIn, error)
}
