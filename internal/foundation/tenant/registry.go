package tenant

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SchemaMigrator lets a business module register how its own schema should
// be created (and optionally seeded) inside every tenant database, so that
// provisioning a tenant is the single place that creates all tenant-owned
// tables - rather than each module lazily migrating on its own first
// request. A module registers one of these from its NewModule during
// startup wiring, before the server starts accepting requests.
type SchemaMigrator struct {
	Name    string
	Migrate func(*gorm.DB) error
	Seed    func(*gorm.DB) error // optional; nil if the module has no seed data
	// SeedKind says what Seed creates. The zero value is SeedDemo (sample data),
	// so a module must opt in to SeedReference deliberately.
	SeedKind SeedKind
}

// SeedKind separates data the system needs to work from sample data.
type SeedKind int

const (
	// SeedDemo is sample data (fictional customers, employees, orders...). It is
	// only created when demo seeding is enabled.
	SeedDemo SeedKind = iota
	// SeedReference is data every company needs, e.g. the default chart of
	// accounts. It is always created.
	SeedReference
)

var demoSeeding bool

// SetDemoSeeding turns sample-data seeding on or off. Call it once during
// startup, before modules are wired.
func SetDemoSeeding(enabled bool) { demoSeeding = enabled }

// DemoSeedingEnabled reports whether sample data should be created. Modules that
// seed outside the tenant registry (shared chat/drive/docflow data) check it too.
func DemoSeedingEnabled() bool { return demoSeeding }

var registry []SchemaMigrator

// RegisterSchema registers a module's tenant schema migrator. Not safe for
// concurrent use - call only during single-threaded startup wiring.
func RegisterSchema(m SchemaMigrator) {
	registry = append(registry, m)
}

// ProvisionAll runs every registered module's Migrate (then Seed) against
// the given tenant database, in registration order. Both steps must be
// idempotent: this is also re-run against already-provisioned tenant
// databases to backfill schemas registered after that tenant was created.
func ProvisionAll(tenantDB *gorm.DB) error {
	for _, m := range registry {
		if err := m.Migrate(tenantDB); err != nil {
			return fmt.Errorf("failed to migrate %s schema: %w", m.Name, err)
		}
		if m.Seed != nil && (m.SeedKind == SeedReference || demoSeeding) {
			if err := m.Seed(tenantDB); err != nil {
				return fmt.Errorf("failed to seed %s schema: %w", m.Name, err)
			}
		}
	}
	return nil
}

// OnboardingUser is a company member as seen by onboarding hooks.
type OnboardingUser struct {
	UserID    uuid.UUID
	CompanyID uuid.UUID
	TenantID  *uuid.UUID
	Name      string
	Email     string
	Role      string // legacy account role: admin / manager / staff
}

// UserOnboarding creates whatever a member needs inside the company's tenant
// database the first time they use it (for HRM: their employee record). It must
// be idempotent: it runs again after a restart or a failed attempt.
type UserOnboarding func(ctx context.Context, tenantDB *gorm.DB, u OnboardingUser) error

var onboardingHooks []UserOnboarding

// RegisterUserOnboarding registers a hook. Not safe for concurrent use - call
// only during single-threaded startup wiring.
func RegisterUserOnboarding(h UserOnboarding) { onboardingHooks = append(onboardingHooks, h) }

// RunUserOnboarding runs every registered hook. One failing hook does not stop
// the others; all errors are returned joined.
func RunUserOnboarding(ctx context.Context, tenantDB *gorm.DB, u OnboardingUser) error {
	var errs []error
	for _, h := range onboardingHooks {
		if err := h(ctx, tenantDB, u); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
