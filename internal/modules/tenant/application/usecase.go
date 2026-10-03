package application

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/tenant/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// tenantDBNamePattern is the safe-identifier allowlist a generated tenant
// database name must match before it is ever interpolated into SQL.
var tenantDBNamePattern = regexp.MustCompile(`^tenant_[a-f0-9]{8}$`)

type TenantUseCase interface {
	GetOrCreateTenantDatabase(ctx context.Context, companyID uuid.UUID) (*TenantDatabaseDTO, error)
	ProvisionTenantSchema(ctx context.Context, companyID uuid.UUID) error
}

type tenantUseCase struct {
	repo  domain.TenantDatabaseRepository
	dbCfg *config.DatabaseConfig
}

func NewTenantUseCase(repo domain.TenantDatabaseRepository, dbCfg *config.DatabaseConfig) TenantUseCase {
	return &tenantUseCase{repo: repo, dbCfg: dbCfg}
}

func generateTenantDBName(companyID uuid.UUID) string {
	hex := companyID.String()
	// strip dashes, take first 8 hex chars - deterministic, immutable per company
	compact := regexp.MustCompile(`-`).ReplaceAllString(hex, "")
	return fmt.Sprintf("tenant_%s", compact[:8])
}

func (uc *tenantUseCase) GetOrCreateTenantDatabase(ctx context.Context, companyID uuid.UUID) (*TenantDatabaseDTO, error) {
	existing, err := uc.repo.GetByCompanyID(ctx, companyID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to look up tenant database")
	}
	if existing != nil {
		return toDTO(existing), nil
	}

	dbName := generateTenantDBName(companyID)
	if !tenantDBNamePattern.MatchString(dbName) {
		return nil, apperrors.NewInternal(fmt.Errorf("generated invalid tenant db name: %s", dbName), "Invalid tenant database name")
	}

	record := &domain.TenantDatabase{
		CompanyID:    companyID,
		DatabaseHost: uc.dbCfg.Host,
		DatabasePort: uc.dbCfg.Port,
		DatabaseName: dbName,
		Status:       domain.TenantDBStatusProvisioning,
	}
	if err := uc.repo.Create(ctx, record); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record tenant database")
	}

	if err := createDatabase(uc.dbCfg, dbName); err != nil {
		record.Status = domain.TenantDBStatusFailed
		_ = uc.repo.Update(ctx, record)
		return nil, apperrors.NewInternal(err, "Failed to provision tenant database")
	}

	record.Status = domain.TenantDBStatusReady
	if err := uc.repo.Update(ctx, record); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update tenant database status")
	}

	if err := provisionSchema(uc.dbCfg, dbName); err != nil {
		slog.Error("failed to provision tenant demo schema", "companyId", companyID, "error", err)
	}

	return toDTO(record), nil
}

func (uc *tenantUseCase) ProvisionTenantSchema(ctx context.Context, companyID uuid.UUID) error {
	record, err := uc.repo.GetByCompanyID(ctx, companyID)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to look up tenant database")
	}
	if record == nil {
		return apperrors.NewNotFound("Tenant database not provisioned for this company")
	}
	if err := provisionSchema(uc.dbCfg, record.DatabaseName); err != nil {
		return apperrors.NewInternal(err, "Failed to provision tenant schema")
	}
	return nil
}

// createDatabase connects to the "postgres" maintenance database and runs
// CREATE DATABASE. This cannot run inside a transaction in Postgres, so a
// plain non-transactional *sql.DB Exec is used.
func createDatabase(dbCfg *config.DatabaseConfig, dbName string) error {
	if !tenantDBNamePattern.MatchString(dbName) {
		return fmt.Errorf("refusing to create database with unsafe name: %s", dbName)
	}

	sqlDB, err := sql.Open("pgx", dbCfg.DSNForDatabase("postgres"))
	if err != nil {
		return fmt.Errorf("failed to open maintenance connection: %w", err)
	}
	defer sqlDB.Close()

	var exists bool
	checkErr := sqlDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if checkErr != nil {
		return fmt.Errorf("failed to check database existence: %w", checkErr)
	}
	if exists {
		return nil
	}

	if _, err := sqlDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName)); err != nil {
		return fmt.Errorf("failed to create database %s: %w", dbName, err)
	}
	return nil
}

// provisionSchema connects to the tenant database and runs every module's
// registered schema migrator (see foundation/tenant.RegisterSchema) against
// it. Called both when a tenant database is first created and, via
// ProvisionTenantSchema, to backfill schemas registered after that.
func provisionSchema(dbCfg *config.DatabaseConfig, dbName string) error {
	tenantDB, err := gorm.Open(postgres.Open(dbCfg.DSNForDatabase(dbName)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database %s: %w", dbName, err)
	}
	defer func() {
		if sqlDB, err := tenantDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	return tenantMgr.ProvisionAll(tenantDB)
}

func toDTO(td *domain.TenantDatabase) *TenantDatabaseDTO {
	return &TenantDatabaseDTO{
		CompanyID:    td.CompanyID,
		DatabaseHost: td.DatabaseHost,
		DatabasePort: td.DatabasePort,
		DatabaseName: td.DatabaseName,
		Status:       string(td.Status),
	}
}
