package main

import (
	"context"
	"fmt"
	"log"
	"os"

	currencyApp "github.com/divinecoid/one-backend/internal/modules/currency/application"
	"github.com/divinecoid/one-backend/internal/modules/currency/domain"
	currencyInfra "github.com/divinecoid/one-backend/internal/modules/currency/infrastructure"
	salesDomain "github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect control-plane db: %v", err)
	}

	var companyDBs []string
	if err := db.Raw("SELECT database_name FROM tenant_databases WHERE database_name IS NOT NULL AND database_name <> ''").Scan(&companyDBs).Error; err != nil {
		log.Fatalf("list company dbs: %v", err)
	}

	for _, dbName := range companyDBs {
		tDsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"), dbName, os.Getenv("DB_SSLMODE"),
		)
		tdb, err := gorm.Open(postgres.Open(tDsn), &gorm.Config{})
		if err != nil {
			log.Printf("skip %s: connect failed: %v", dbName, err)
			continue
		}

		if err := tdb.AutoMigrate(&domain.Currency{}, &domain.ExchangeRate{}); err != nil {
			log.Printf("%s: currency automigrate failed: %v", dbName, err)
			continue
		}
		if err := tdb.AutoMigrate(&salesDomain.SalesOrder{}); err != nil {
			log.Printf("%s: sales_orders automigrate failed: %v", dbName, err)
			continue
		}

		repo := currencyInfra.NewCurrencyRepository(tdb)
		uc := currencyApp.NewCurrencyUseCase(repo)
		if err := uc.SeedBaseCurrency(context.Background(), "IDR", "Indonesian Rupiah", "Rp"); err != nil {
			log.Printf("%s: seed base currency failed: %v", dbName, err)
			continue
		}

		log.Printf("%s: done", dbName)
	}

	fmt.Println("backfill complete")
}
