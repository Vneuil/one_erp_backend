package application

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/user/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/divinecoid/one-backend/internal/shared/utils"
	"github.com/google/uuid"
)

type UserUseCase interface {
	Create(ctx context.Context, dto CreateUserDTO) (*UserResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*UserResponseDTO, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, query types.PaginationQuery, companyID *uuid.UUID) ([]UserResponseDTO, types.PaginationMeta, error)
	Update(ctx context.Context, id uuid.UUID, dto UpdateUserDTO) (*UserResponseDTO, error)
	AssignRole(ctx context.Context, id uuid.UUID, dto AssignRoleDTO) (*UserResponseDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type userUseCase struct {
	repo domain.UserRepository
}

func NewUserUseCase(repo domain.UserRepository) UserUseCase {
	return &userUseCase{repo: repo}
}

func (uc *userUseCase) Create(ctx context.Context, dto CreateUserDTO) (*UserResponseDTO, error) {
	email := strings.ToLower(strings.TrimSpace(dto.Email))
	name := strings.TrimSpace(dto.Name)

	validationErrors := make(map[string]string)
	if email == "" {
		validationErrors["email"] = "Email is required"
	}
	if name == "" {
		validationErrors["name"] = "Name is required"
	}
	if len(dto.Password) < 6 {
		validationErrors["password"] = "Password must be at least 6 characters"
	}
	if len(validationErrors) > 0 {
		return nil, apperrors.NewValidation(validationErrors)
	}

	existing, err := uc.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing email")
	}
	if existing != nil {
		return nil, apperrors.NewConflict("User with this email already exists")
	}

	hash, err := utils.HashPassword(dto.Password)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to hash password")
	}

	role := dto.Role
	if role == "" {
		role = domain.RoleStaff
	}

	user := &domain.User{
		CompanyID:    dto.CompanyID,
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		IsActive:     true,
	}

	if err := uc.repo.Create(ctx, user); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create user")
	}

	res := ToUserResponse(user)
	return &res, nil
}

func (uc *userUseCase) GetByID(ctx context.Context, id uuid.UUID) (*UserResponseDTO, error) {
	user, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve user")
	}
	if user == nil {
		return nil, apperrors.NewNotFound("User not found")
	}

	res := ToUserResponse(user)
	return &res, nil
}

func (uc *userUseCase) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := uc.repo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve user")
	}
	return user, nil
}

func (uc *userUseCase) List(ctx context.Context, query types.PaginationQuery, companyID *uuid.UUID) ([]UserResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()

	users, total, err := uc.repo.List(ctx, query, companyID)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list users")
	}

	dtos := make([]UserResponseDTO, len(users))
	for i := range users {
		dtos[i] = ToUserResponse(&users[i])
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return dtos, meta, nil
}

func (uc *userUseCase) Update(ctx context.Context, id uuid.UUID, dto UpdateUserDTO) (*UserResponseDTO, error) {
	user, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve user")
	}
	if user == nil {
		return nil, apperrors.NewNotFound("User not found")
	}

	if dto.Name != nil {
		name := strings.TrimSpace(*dto.Name)
		if name == "" {
			return nil, apperrors.NewBadRequest("Name cannot be empty")
		}
		user.Name = name
	}
	if dto.Role != nil {
		user.Role = *dto.Role
	}
	if dto.IsActive != nil {
		user.IsActive = *dto.IsActive
	}

	if err := uc.repo.Update(ctx, user); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update user")
	}
	response := ToUserResponse(user)
	return &response, nil
}

func (uc *userUseCase) AssignRole(ctx context.Context, id uuid.UUID, dto AssignRoleDTO) (*UserResponseDTO, error) {
	user, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve user")
	}
	if user == nil {
		return nil, apperrors.NewNotFound("User not found")
	}

	user.RoleID = dto.RoleID

	if err := uc.repo.Update(ctx, user); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to assign role")
	}

	res := ToUserResponse(user)
	return &res, nil
}

func (uc *userUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	user, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to retrieve user")
	}
	if user == nil {
		return apperrors.NewNotFound("User not found")
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete user")
	}

	return nil
}
