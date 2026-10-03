package main

import (
	"fmt"
	"log"
	"os"

	cooperativeDomain "github.com/divinecoid/one-backend/internal/modules/cooperative/domain"
	kpiDomain "github.com/divinecoid/one-backend/internal/modules/kpi/domain"
	leaveDomain "github.com/divinecoid/one-backend/internal/modules/leave/domain"
	payrollDomain "github.com/divinecoid/one-backend/internal/modules/payroll/domain"
	projectDomain "github.com/divinecoid/one-backend/internal/modules/project/domain"
	projecttaskDomain "github.com/divinecoid/one-backend/internal/modules/projecttask/domain"
	projectticketDomain "github.com/divinecoid/one-backend/internal/modules/projectticket/domain"
	reimbursementDomain "github.com/divinecoid/one-backend/internal/modules/reimbursement/domain"
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

		if err := tdb.AutoMigrate(
			&leaveDomain.LeaveRequest{},
			&kpiDomain.KpiReview{},
			&cooperativeDomain.CooperativeLoan{},
			&reimbursementDomain.ReimbursementClaim{},
			&payrollDomain.PayrollEntry{},
			&projecttaskDomain.ProjectTask{},
			&projecttaskDomain.TaskChecklistItem{},
			&projectticketDomain.ProjectTicket{},
			&projectDomain.TimeEntry{},
		); err != nil {
			log.Printf("%s: automigrate failed: %v", dbName, err)
			continue
		}

		log.Printf("%s: OK", dbName)
	}

	log.Println("backfill complete")
}
