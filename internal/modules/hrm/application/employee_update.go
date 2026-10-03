package application

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// UpdateEmployeeDTO changes only the fields that are present. NIP is not
// editable: payroll and attendance are keyed by it.
type UpdateEmployeeDTO struct {
	Name         *string  `json:"name"`
	Email        *string  `json:"email"`
	Phone        *string  `json:"phone"`
	Department   *string  `json:"department"`
	Role         *string  `json:"role"`
	ContractType *string  `json:"contractType"`
	JoinDate     *string  `json:"joinDate"`
	Status       *string  `json:"status"`
	BaseSalary   *float64 `json:"baseSalary"`
	PTKPStatus   *string  `json:"ptkpStatus"`
	HasNPWP      *bool    `json:"hasNpwp"`
	// ManagerID sets the direct supervisor; an empty string clears it.
	ManagerID *string `json:"managerId"`
}

// ApplyEmployeeUpdate validates dto and writes it onto emp.
func ApplyEmployeeUpdate(emp *domain.Employee, dto UpdateEmployeeDTO) error {
	if dto.Name != nil {
		n := strings.TrimSpace(*dto.Name)
		if n == "" {
			return apperrors.NewBadRequest("Employee name cannot be empty")
		}
		emp.Name = n
	}
	if dto.BaseSalary != nil {
		if *dto.BaseSalary < 0 {
			return apperrors.NewBadRequest("Base salary cannot be negative")
		}
		emp.BaseSalary = *dto.BaseSalary
	}
	if dto.PTKPStatus != nil {
		p := strings.ToUpper(strings.TrimSpace(*dto.PTKPStatus))
		if !validPTKPStatus(p) {
			return apperrors.NewBadRequest("PTKP status must be one of TK/0-TK/3 or K/0-K/3")
		}
		emp.PTKPStatus = p
	}
	if dto.Email != nil {
		emp.Email = strings.TrimSpace(*dto.Email)
	}
	if dto.Phone != nil {
		emp.Phone = strings.TrimSpace(*dto.Phone)
	}
	for _, f := range []struct {
		in  *string
		out *string
	}{{dto.Department, &emp.Department}, {dto.Role, &emp.Role}, {dto.ContractType, &emp.ContractType}, {dto.JoinDate, &emp.JoinDate}, {dto.Status, &emp.Status}} {
		if f.in != nil {
			v := strings.TrimSpace(*f.in)
			if v == "" {
				return apperrors.NewBadRequest("Fields cannot be set to empty")
			}
			*f.out = v
		}
	}
	if dto.HasNPWP != nil {
		emp.NoNPWP = !*dto.HasNPWP
	}
	return nil
}

func (uc *hrmUseCase) UpdateEmployee(ctx context.Context, id uuid.UUID, dto UpdateEmployeeDTO) (*EmployeeResponseDTO, error) {
	emp, err := uc.repo.GetEmployeeByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get employee")
	}
	if emp == nil {
		return nil, apperrors.NewNotFound("Employee not found")
	}
	if err := ApplyEmployeeUpdate(emp, dto); err != nil {
		return nil, err
	}
	if dto.ManagerID != nil {
		if err := uc.applyManager(ctx, emp, *dto.ManagerID); err != nil {
			return nil, err
		}
	}
	if err := uc.repo.UpdateEmployee(ctx, emp); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update employee")
	}
	return ToEmployeeResponse(emp), nil
}

func (uc *hrmUseCase) DeleteEmployee(ctx context.Context, id uuid.UUID) error {
	ok, err := uc.repo.DeleteEmployee(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to delete employee")
	}
	if !ok {
		return apperrors.NewNotFound("Employee not found")
	}
	return nil
}
