package application

import (
	"context"
	"math"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/assets/domain"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/google/uuid"
)

func (r *fakeAssetsRepo) GetAssetByID(_ context.Context, id uuid.UUID) (*domain.FixedAsset, error) {
	for i := range r.assets {
		if r.assets[i].ID == id {
			return &r.assets[i], nil
		}
	}
	return nil, nil
}
func (r *fakeAssetsRepo) UpdateAsset(_ context.Context, a *domain.FixedAsset) error {
	for i := range r.assets {
		if r.assets[i].ID == a.ID {
			r.assets[i] = *a
		}
	}
	return nil
}

func sumLines(lines []financeApp.LedgerLine) (d, c float64) {
	for _, l := range lines {
		d += l.Debit
		c += l.Credit
	}
	return
}

func TestPostAcquisitionJournalsCostOnce(t *testing.T) {
	a := asset(12000, 1, monthsAgo(3))
	uc, _, led := newDepUC(a)
	if _, err := uc.PostAcquisition(context.Background(), a.ID, PostAcquisitionDTO{FundingSource: "cash"}); err != nil {
		t.Fatal(err)
	}
	lines := led.docs["asset-acquisition:"+a.ID.String()]
	if len(lines) != 2 || lines[0].AccountCode != financeApp.AccountFixedAssets || lines[0].Debit != 12000 || lines[1].AccountCode != financeApp.AccountCash {
		t.Fatalf("unexpected journal: %+v", lines)
	}
	if _, err := uc.PostAcquisition(context.Background(), a.ID, PostAcquisitionDTO{FundingSource: "cash"}); err == nil {
		t.Fatal("second acquisition post must be rejected")
	}
}

func TestDisposeRequiresAcquisitionAndDepreciation(t *testing.T) {
	a := asset(12000, 1, monthsAgo(2))
	uc, _, _ := newDepUC(a)
	ctx := context.Background()
	if _, err := uc.DisposeAsset(ctx, a.ID, DisposeAssetDTO{Proceeds: 1}); err == nil {
		t.Fatal("dispose without a posted acquisition must fail")
	}
	if _, err := uc.PostAcquisition(ctx, a.ID, PostAcquisitionDTO{FundingSource: "equity"}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.DisposeAsset(ctx, a.ID, DisposeAssetDTO{Proceeds: 1}); err == nil {
		t.Fatal("dispose with unposted depreciation must fail")
	}
}

func disposeScenario(t *testing.T, proceeds float64) (*AssetResponseDTO, []financeApp.LedgerLine) {
	t.Helper()
	a := asset(12000, 1, monthsAgo(2)) // 1000/month; 3 months due incl. current
	uc, _, led := newDepUC(a)
	ctx := context.Background()
	if _, err := uc.PostAcquisition(ctx, a.ID, PostAcquisitionDTO{FundingSource: "cash"}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.PostDepreciation(ctx, "", true); err != nil {
		t.Fatal(err)
	}
	res, err := uc.DisposeAsset(ctx, a.ID, DisposeAssetDTO{Proceeds: proceeds})
	if err != nil {
		t.Fatal(err)
	}
	lines := led.docs["asset-disposal:"+a.ID.String()]
	d, c := sumLines(lines)
	if math.Abs(d-c) > 0.001 {
		t.Fatalf("unbalanced disposal: dr %v cr %v", d, c)
	}
	return res, lines
}

func TestDisposeOnCreditBooksReceivable(t *testing.T) {
	a := asset(12000, 1, monthsAgo(2))
	uc, _, led := newDepUC(a)
	ctx := context.Background()
	if _, err := uc.PostAcquisition(ctx, a.ID, PostAcquisitionDTO{FundingSource: "cash"}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.PostDepreciation(ctx, "", true); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.DisposeAsset(ctx, a.ID, DisposeAssetDTO{Proceeds: 10000, ReceivedIn: "receivable"}); err == nil {
		t.Fatal("a sale on credit needs a buyer")
	}
	if _, err := uc.DisposeAsset(ctx, a.ID, DisposeAssetDTO{Proceeds: 10000, ReceivedIn: "credit-card", BuyerName: "X"}); err == nil {
		t.Fatal("unknown receivedIn must be refused")
	}
	if _, err := uc.DisposeAsset(ctx, a.ID, DisposeAssetDTO{Proceeds: 10000, ReceivedIn: "receivable", BuyerName: "PT Pembeli"}); err != nil {
		t.Fatal(err)
	}
	lines := led.docs["asset-disposal:"+a.ID.String()]
	var hasAR bool
	for _, l := range lines {
		if l.AccountCode == financeApp.AccountReceivble && l.Debit == 10000 {
			hasAR = true
		}
		if l.AccountCode == financeApp.AccountCash {
			t.Fatalf("a sale on credit must not touch cash: %+v", lines)
		}
	}
	if !hasAR || len(led.receivables) != 1 || led.receivables[0].PartyName != "PT Pembeli" || led.receivables[0].Total != 10000 {
		t.Fatalf("receivable: %+v lines %+v", led.receivables, lines)
	}
}

func TestDisposeGainAndLoss(t *testing.T) {
	// accumulated 3000 → book value 9000
	res, lines := disposeScenario(t, 10000)
	if res.DisposalGainLoss != 1000 || res.Status != "Disposed" {
		t.Fatalf("gain case: %+v", res)
	}
	var hasGain bool
	for _, l := range lines {
		if l.AccountCode == financeApp.AccountOtherIncome && l.Credit == 1000 {
			hasGain = true
		}
	}
	if !hasGain {
		t.Fatalf("missing gain line: %+v", lines)
	}
	res, lines = disposeScenario(t, 4000)
	if res.DisposalGainLoss != -5000 {
		t.Fatalf("loss case: %+v", res)
	}
	var hasLoss bool
	for _, l := range lines {
		if l.AccountCode == financeApp.AccountDisposalLoss && l.Debit == 5000 {
			hasLoss = true
		}
	}
	if !hasLoss {
		t.Fatalf("missing loss line: %+v", lines)
	}
}
