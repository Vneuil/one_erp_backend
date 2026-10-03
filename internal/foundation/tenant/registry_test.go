package tenant

import (
	"testing"

	"gorm.io/gorm"
)

// withRegistry swaps the package-level registry and seeding switch for a test.
func withRegistry(t *testing.T, demo bool, ms ...SchemaMigrator) {
	t.Helper()
	oldReg, oldDemo := registry, demoSeeding
	registry, demoSeeding = ms, demo
	t.Cleanup(func() { registry, demoSeeding = oldReg, oldDemo })
}

func TestProvisionAllSkipsDemoSeedsUnlessEnabled(t *testing.T) {
	var ran []string
	seed := func(name string) func(*gorm.DB) error {
		return func(*gorm.DB) error { ran = append(ran, name); return nil }
	}
	migrate := func(*gorm.DB) error { return nil }
	ms := []SchemaMigrator{
		{Name: "finance", Migrate: migrate, Seed: seed("finance"), SeedKind: SeedReference},
		{Name: "customer", Migrate: migrate, Seed: seed("customer")}, // zero value = demo
		{Name: "noseed", Migrate: migrate},
	}

	withRegistry(t, false, ms...)
	if err := ProvisionAll(nil); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 1 || ran[0] != "finance" {
		t.Fatalf("with demo seeding off only reference seeds may run, ran %v", ran)
	}

	ran = nil
	withRegistry(t, true, ms...)
	if err := ProvisionAll(nil); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 2 {
		t.Fatalf("with demo seeding on both seeds should run, ran %v", ran)
	}
}

func TestMigrationsAlwaysRunEvenWhenSeedsAreSkipped(t *testing.T) {
	migrated := 0
	withRegistry(t, false, SchemaMigrator{
		Name:    "customer",
		Migrate: func(*gorm.DB) error { migrated++; return nil },
		Seed:    func(*gorm.DB) error { t.Fatal("demo seed must not run"); return nil },
	})
	if err := ProvisionAll(nil); err != nil {
		t.Fatal(err)
	}
	if migrated != 1 {
		t.Fatalf("schema must still be migrated, migrated=%d", migrated)
	}
}

func TestSetDemoSeeding(t *testing.T) {
	withRegistry(t, false)
	SetDemoSeeding(true)
	if !DemoSeedingEnabled() {
		t.Fatal("SetDemoSeeding(true) should enable demo seeding")
	}
}
