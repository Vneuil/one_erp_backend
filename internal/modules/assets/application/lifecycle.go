package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/assets/domain"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// fundingAccounts maps how an asset was paid for to the credited account.
var fundingAccounts = map[string]string{
	"cash":    financeApp.AccountCash,
	"payable": financeApp.AccountPayable,
	"equity":  financeApp.AccountOwnerCapital,
}

func (uc *assetsUseCase) postAcquisition(ctx context.Context, a *domain.FixedAsset, source, date string) error {
	if uc.ledger == nil {
		return fmt.Errorf("assets: ledger not configured")
	}
	doc := "asset-acquisition:" + a.ID.String()
	desc := a.AssetCode + " " + a.Name
	err := uc.ledger.PostEntryOn(ctx, date, doc, "Asset acquisition - "+desc, []financeApp.LedgerLine{
		{AccountCode: financeApp.AccountFixedAssets, Debit: a.PurchasePrice, Description: desc},
		{AccountCode: fundingAccounts[source], Credit: a.PurchasePrice, Description: desc},
	})
	if err != nil {
		return err
	}
	a.AcquisitionDoc = doc
	return uc.repo.UpdateAsset(ctx, a)
}

// PostAcquisition (Pembelian Aktiva) journals Dr Fixed Assets / Cr cash, payable
// or owner's equity for an asset recorded without a journal, so its cost sits
// in the ledger next to the depreciation already posted against it.
func (uc *assetsUseCase) PostAcquisition(ctx context.Context, id uuid.UUID, dto PostAcquisitionDTO) (*AssetResponseDTO, error) {
	a, err := uc.repo.GetAssetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get asset")
	}
	if a == nil {
		return nil, apperrors.NewNotFound("Asset not found")
	}
	if a.AcquisitionDoc != "" {
		return nil, apperrors.NewBadRequest("Acquisition is already posted for this asset")
	}
	if _, ok := fundingAccounts[dto.FundingSource]; !ok {
		return nil, apperrors.NewBadRequest("Funding source must be one of cash, payable, equity")
	}
	if a.PurchasePrice <= 0 {
		return nil, apperrors.NewBadRequest("Purchase price must be positive to post an acquisition")
	}
	date := dto.Date
	if date == "" {
		date = a.PurchaseDate
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, apperrors.NewBadRequest("Date must be in YYYY-MM-DD format")
	}
	if err := uc.postAcquisition(ctx, a, dto.FundingSource, date); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to post the acquisition")
	}
	return ToAssetResponse(a), nil
}

// DisposeAsset (Penjualan Aktiva) sells or writes off an asset. Depreciation
// must be posted through the disposal month first. The entry removes cost and
// accumulated depreciation and books the proceeds in cash; the difference to
// book value is a gain (Other Income) or loss (Loss on Asset Disposal).
func (uc *assetsUseCase) DisposeAsset(ctx context.Context, id uuid.UUID, dto DisposeAssetDTO) (*AssetResponseDTO, error) {
	if uc.ledger == nil {
		return nil, apperrors.NewInternal(fmt.Errorf("assets: ledger not configured"), "Ledger posting is not available")
	}
	a, err := uc.repo.GetAssetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get asset")
	}
	if a == nil {
		return nil, apperrors.NewNotFound("Asset not found")
	}
	if a.Status == "Disposed" {
		return nil, apperrors.NewBadRequest("Asset is already disposed")
	}
	if a.AcquisitionDoc == "" {
		return nil, apperrors.NewBadRequest("Post the asset's acquisition to the ledger before disposing it")
	}
	if dto.Proceeds < 0 {
		return nil, apperrors.NewBadRequest("Proceeds cannot be negative")
	}
	onCredit := dto.ReceivedIn == "receivable"
	if dto.ReceivedIn != "" && dto.ReceivedIn != "cash" && !onCredit {
		return nil, apperrors.NewBadRequest("receivedIn must be cash or receivable")
	}
	if onCredit {
		if strings.TrimSpace(dto.BuyerName) == "" {
			return nil, apperrors.NewBadRequest("buyerName is required for a sale on credit")
		}
		if dto.Proceeds <= 0 {
			return nil, apperrors.NewBadRequest("A sale on credit needs proceeds above zero")
		}
		if dto.DueDate != "" {
			if _, err := time.Parse("2006-01-02", dto.DueDate); err != nil {
				return nil, apperrors.NewBadRequest("dueDate must be in YYYY-MM-DD format")
			}
		}
	}
	date := dto.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	dt, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, apperrors.NewBadRequest("Date must be in YYYY-MM-DD format")
	}
	if date < a.PurchaseDate {
		return nil, apperrors.NewBadRequest("Disposal date cannot be before the purchase date")
	}
	period := dt.Format(periodLayout)

	postings, err := uc.repo.ListDepreciationPostings(ctx, "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load depreciation postings")
	}
	var accum float64
	var posted int
	for _, p := range postings {
		if p.AssetID != a.ID {
			continue
		}
		if p.Period > period {
			return nil, apperrors.NewBadRequest("Depreciation is already posted after the disposal date")
		}
		accum += p.Amount
		posted++
	}
	if start, _, ok := depreciable(*a); ok {
		due := len(monthsBetween(start, dt))
		if max := a.UsefulLifeYears * 12; due > max {
			due = max
		}
		if due > posted {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Post depreciation through %s first (%d month(s) outstanding)", period, due-posted))
		}
	}
	accum, proceeds := round2(accum), round2(dto.Proceeds)
	gain := round2(proceeds - (a.PurchasePrice - accum))

	desc := a.AssetCode + " " + a.Name
	var lines []financeApp.LedgerLine
	if proceeds > 0 {
		account := financeApp.AccountCash
		if onCredit {
			account = financeApp.AccountReceivble
		}
		lines = append(lines, financeApp.LedgerLine{AccountCode: account, Debit: proceeds, Description: desc})
	}
	if accum > 0 {
		lines = append(lines, financeApp.LedgerLine{AccountCode: financeApp.AccountAccumDepreciation, Debit: accum, Description: desc})
	}
	if gain < 0 {
		lines = append(lines, financeApp.LedgerLine{AccountCode: financeApp.AccountDisposalLoss, Debit: -gain, Description: desc})
	}
	lines = append(lines, financeApp.LedgerLine{AccountCode: financeApp.AccountFixedAssets, Credit: a.PurchasePrice, Description: desc})
	if gain > 0 {
		lines = append(lines, financeApp.LedgerLine{AccountCode: financeApp.AccountOtherIncome, Credit: gain, Description: desc})
	}

	doc := "asset-disposal:" + a.ID.String()
	memo := "Asset disposal - " + desc
	if dto.Notes != "" {
		memo += " (" + dto.Notes + ")"
	}
	if err := uc.ledger.PostEntryOn(ctx, date, doc, memo, lines); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to post the disposal")
	}
	if onCredit {
		due := dto.DueDate
		if due == "" {
			due = dt.AddDate(0, 0, 30).Format("2006-01-02")
		}
		err := uc.ledger.UpsertReceivable(ctx, financeApp.SubledgerDoc{
			SourceDoc: doc, PartyName: strings.TrimSpace(dto.BuyerName), DocNo: "Jual " + a.AssetCode, IssueDate: date, DueDate: due, Total: proceeds,
		})
		if err != nil {
			return nil, apperrors.NewInternal(err, "Disposal posted but the receivable could not be recorded")
		}
	}
	a.Status, a.DisposedAt, a.DisposalProceeds, a.DisposalGainLoss, a.DisposalDoc = "Disposed", date, proceeds, gain, doc
	a.AccumulatedDepreciation, a.CurrentBookValue = accum, 0
	if err := uc.repo.UpdateAsset(ctx, a); err != nil {
		return nil, apperrors.NewInternal(err, "Posted to the ledger but could not mark the asset disposed")
	}
	return ToAssetResponse(a), nil
}
