// backfill-ledger posts general-ledger journal entries for documents that were
// created before automatic posting existed (sales/purchase invoices and their
// payments, purchase down payments, payroll, reimbursements).
//
// It reuses the same entry builders as the live code, and every entry is keyed
// by its source document, so the command is safe to re-run: documents that
// already have a journal entry are skipped, and payments only post the amount
// not yet in the ledger.
//
// It also mirrors every sales/purchase invoice into finance receivables/payables
// (adding the source_doc column to those tables when -apply is used).
//
// Default is a dry run. Pass -apply to write. Opening balances are NOT created.
//
//	go run ./cmd/backfill-ledger            # dry run
//	go run ./cmd/backfill-ledger -apply
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeDomain "github.com/divinecoid/one-backend/internal/modules/finance/domain"
	financeInfra "github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"
	payrollApp "github.com/divinecoid/one-backend/internal/modules/payroll/application"
	payrollDomain "github.com/divinecoid/one-backend/internal/modules/payroll/domain"
	procApp "github.com/divinecoid/one-backend/internal/modules/procurement/application"
	procDomain "github.com/divinecoid/one-backend/internal/modules/procurement/domain"
	reimbApp "github.com/divinecoid/one-backend/internal/modules/reimbursement/application"
	reimbDomain "github.com/divinecoid/one-backend/internal/modules/reimbursement/domain"
	salesApp "github.com/divinecoid/one-backend/internal/modules/sales/application"
	salesDomain "github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type runner struct {
	poster financeApp.LedgerPoster
	repo   financeDomain.FinanceRepository
	apply  bool
	posted int
	failed int
	// subReady is false in a dry run against a tenant whose AR/AP tables do
	// not have the source_doc column yet (apply adds it via AutoMigrate).
	subReady bool
	subs     int
}

// upsertSub mirrors an invoice into the AR/AP sub-ledger, counting only
// documents that are not mirrored yet (existing ones are refreshed on apply).
func (r *runner) upsertSub(tenant *uuid.UUID, d financeApp.SubledgerDoc, receivable bool) {
	ctx := tenantctx.WithTenantID(context.Background(), tenant)
	exists := false
	if r.subReady {
		var err error
		if receivable {
			var rec *financeDomain.Receivable
			rec, err = r.repo.GetReceivableBySourceDoc(ctx, d.SourceDoc)
			exists = rec != nil
		} else {
			var pay *financeDomain.Payable
			pay, err = r.repo.GetPayableBySourceDoc(ctx, d.SourceDoc)
			exists = pay != nil
		}
		if err != nil {
			log.Printf("  ! lookup %s: %v", d.SourceDoc, err)
			r.failed++
			return
		}
	}
	if !exists {
		r.subs++
	}
	if !r.apply {
		return
	}
	var err error
	if receivable {
		err = r.poster.UpsertReceivable(ctx, d)
	} else {
		err = r.poster.UpsertPayable(ctx, d)
	}
	if err != nil {
		log.Printf("  ! sync %s: %v", d.SourceDoc, err)
		r.failed++
	}
}

// post writes (or, in a dry run, counts) one entry unless it already exists.
func (r *runner) post(tenant *uuid.UUID, date string, e financeApp.LedgerEntry) {
	ctx := tenantctx.WithTenantID(context.Background(), tenant)
	if existing, err := r.repo.GetJournalEntryBySourceDoc(ctx, e.SourceDoc); err != nil {
		log.Printf("  ! lookup %s: %v", e.SourceDoc, err)
		r.failed++
		return
	} else if existing != nil {
		return
	}
	if !r.apply {
		r.posted++
		return
	}
	if err := r.poster.PostEntryOn(ctx, date, e.SourceDoc, e.Memo+" (backfill)", e.Lines); err != nil {
		log.Printf("  ! post %s: %v", e.SourceDoc, err)
		r.failed++
		return
	}
	r.posted++
}

// postMissingPayments posts only the part of paid that is not yet in the
// ledger (sum of debits under prefix), under a distinct ":backfill" key.
func (r *runner) postMissingPayments(tenant *uuid.UUID, date, prefix string, paid float64, build func(amount float64) financeApp.LedgerEntry) {
	if paid <= 0 {
		return
	}
	ctx := tenantctx.WithTenantID(context.Background(), tenant)
	already, err := r.poster.PostedDebitTotal(ctx, prefix)
	if err != nil {
		log.Printf("  ! payment total %s: %v", prefix, err)
		r.failed++
		return
	}
	if missing := math.Round((paid-already)*100) / 100; missing > 0.005 {
		e := build(missing)
		e.SourceDoc += ":backfill"
		r.post(tenant, date, e)
	}
}

func main() {
	apply := flag.Bool("apply", false, "write journal entries (default: dry run)")
	flag.Parse()

	dsn := func(name string) string {
		return fmt.Sprintf("host=%s port=%s user=%s password='%s' dbname=%s sslmode=%s",
			os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"), name, os.Getenv("DB_SSLMODE"))
	}
	db, err := gorm.Open(postgres.Open(dsn(os.Getenv("DB_NAME"))), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect control-plane db: %v", err)
	}
	var companyDBs []string
	if err := db.Raw("SELECT database_name FROM tenant_databases WHERE database_name IS NOT NULL AND database_name <> ''").Scan(&companyDBs).Error; err != nil {
		log.Fatalf("list company dbs: %v", err)
	}
	mode := "DRY RUN"
	if *apply {
		mode = "APPLY"
	}
	log.Printf("backfill-ledger [%s] across %d tenant databases", mode, len(companyDBs))

	for _, dbName := range companyDBs {
		tdb, err := gorm.Open(postgres.Open(dsn(dbName)), &gorm.Config{})
		if err != nil {
			log.Printf("skip %s: connect failed: %v", dbName, err)
			continue
		}
		if !tdb.Migrator().HasTable(&financeDomain.JournalEntry{}) {
			log.Printf("skip %s: finance schema not provisioned", dbName)
			continue
		}
		repo := financeInfra.NewFinanceRepository(tdb)
		if *apply {
			if err := tdb.AutoMigrate(&financeDomain.Receivable{}, &financeDomain.Payable{}); err != nil {
				log.Printf("skip %s: migrate AR/AP failed: %v", dbName, err)
				continue
			}
		}
		r := &runner{poster: financeApp.NewLedgerPoster(repo), repo: repo, apply: *apply,
			subReady: tdb.Migrator().HasColumn(&financeDomain.Receivable{}, "source_doc") && tdb.Migrator().HasColumn(&financeDomain.Payable{}, "source_doc")}

		var invoices []salesDomain.Invoice
		if tdb.Migrator().HasTable(&salesDomain.Invoice{}) {
			tdb.Find(&invoices)
		}
		for i := range invoices {
			inv := &invoices[i]
			r.post(inv.TenantID, inv.InvoiceDate, salesApp.InvoiceLedgerEntry(inv))
			r.upsertSub(inv.TenantID, salesApp.InvoiceSubledgerDoc(inv), true)
			r.postMissingPayments(inv.TenantID, inv.InvoiceDate, salesApp.InvoicePaymentPrefix(inv), inv.PaidAmount,
				func(a float64) financeApp.LedgerEntry { return salesApp.InvoicePaymentLedgerEntry(inv, a) })
		}

		var pinvs []procDomain.PurchaseInvoice
		if tdb.Migrator().HasTable(&procDomain.PurchaseInvoice{}) {
			tdb.Find(&pinvs)
		}
		for i := range pinvs {
			inv := &pinvs[i]
			r.post(inv.TenantID, inv.InvoiceDate, procApp.PurchaseInvoiceLedgerEntry(inv))
			r.upsertSub(inv.TenantID, procApp.PurchaseInvoiceSubledgerDoc(inv), false)
			r.postMissingPayments(inv.TenantID, inv.InvoiceDate, procApp.PurchaseInvoicePaymentPrefix(inv), inv.PaidAmount,
				func(a float64) financeApp.LedgerEntry { return procApp.PurchaseInvoicePaymentLedgerEntry(inv, a) })
		}

		var dps []procDomain.PurchaseDownPayment
		if tdb.Migrator().HasTable(&procDomain.PurchaseDownPayment{}) {
			tdb.Find(&dps)
		}
		for i := range dps {
			r.post(dps[i].TenantID, dps[i].PaymentDate, procApp.DownPaymentLedgerEntry(&dps[i]))
		}

		var entries []payrollDomain.PayrollEntry
		if tdb.Migrator().HasTable(&payrollDomain.PayrollEntry{}) {
			tdb.Where("status IN ?", []string{"approved", "paid"}).Find(&entries)
		}
		for i := range entries {
			e := &entries[i]
			date := e.UpdatedAt.Format("2006-01-02") // payroll has no approval/payment date
			if acc, ok := payrollApp.PayrollAccrualLedgerEntry(e); ok {
				r.post(e.TenantID, date, acc)
			} else {
				log.Printf("  ! payroll %s skipped: negative take-home pay", e.ID)
			}
			if e.Status == "paid" {
				r.post(e.TenantID, date, payrollApp.PayrollPaymentLedgerEntry(e))
			}
		}

		var claims []reimbDomain.ReimbursementClaim
		if tdb.Migrator().HasTable(&reimbDomain.ReimbursementClaim{}) {
			tdb.Where("status IN ?", []string{"approved", "paid"}).Find(&claims)
		}
		for i := range claims {
			c := &claims[i]
			date := c.Date
			if date == "" {
				date = c.UpdatedAt.Format("2006-01-02")
			}
			r.post(c.TenantID, date, reimbApp.ClaimApprovalLedgerEntry(c))
			if c.Status == "paid" {
				r.post(c.TenantID, c.UpdatedAt.Format("2006-01-02"), reimbApp.ClaimPaymentLedgerEntry(c))
			}
		}

		verb := "would post"
		if *apply {
			verb = "posted"
		}
		log.Printf("%s: %s %d journal entries + %d AR/AP rows, %d errors", dbName, verb, r.posted, r.subs, r.failed)
	}
	fmt.Println("backfill-ledger complete (" + mode + ")")
}
