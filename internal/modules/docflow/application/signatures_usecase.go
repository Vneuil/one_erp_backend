package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/storage"
	"github.com/divinecoid/one-backend/internal/modules/docflow/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type SignaturesUseCase interface {
	UploadDocument(ctx context.Context, dto UploadDocumentDTO) (*SignatureDocumentResponseDTO, error)
	GetDocumentByID(ctx context.Context, id uuid.UUID) (*DocumentWithSignersDTO, error)
	ListDocuments(ctx context.Context, query types.PaginationQuery) ([]SignatureDocumentResponseDTO, types.PaginationMeta, error)
	AddSigner(ctx context.Context, documentID uuid.UUID, dto AddSignerDTO) (*SignatureRequestResponseDTO, error)
	SignRequest(ctx context.Context, requestID uuid.UUID, callerEmail string) (*SignatureRequestResponseDTO, error)
	DeclineRequest(ctx context.Context, requestID uuid.UUID) (*SignatureRequestResponseDTO, error)
	DownloadDocument(ctx context.Context, id uuid.UUID) (*DocumentDownloadDTO, error)

	SeedInitialData(ctx context.Context) error
}

type signaturesUseCase struct {
	repo    domain.SignaturesRepository
	storage storage.Storage
}

func NewSignaturesUseCase(repo domain.SignaturesRepository, store storage.Storage) SignaturesUseCase {
	return &signaturesUseCase{repo: repo, storage: store}
}

func (uc *signaturesUseCase) UploadDocument(ctx context.Context, dto UploadDocumentDTO) (*SignatureDocumentResponseDTO, error) {
	if dto.Title == "" {
		return nil, apperrors.NewBadRequest("Document title is required")
	}
	if dto.FileName == "" {
		return nil, apperrors.NewBadRequest("File name is required")
	}
	if dto.UploadedBy == "" {
		return nil, apperrors.NewBadRequest("Uploaded by is required")
	}

	if len(dto.Content) > 0 && !uc.storage.Enabled() {
		return nil, apperrors.NewServiceUnavailable("File storage is not configured. Please set R2 credentials in the backend .env file.")
	}

	d := &domain.SignatureDocument{
		CompanyID:  dto.CompanyID,
		Title:      dto.Title,
		FileName:   dto.FileName,
		Status:     "draft",
		UploadedBy: dto.UploadedBy,
		UploadedAt: time.Now().Format(time.RFC3339),
	}

	if len(dto.Content) > 0 {
		key := fmt.Sprintf("docflow/documents/%s/%s", uuid.New().String(), dto.FileName)
		if _, err := uc.storage.Upload(ctx, key, dto.Content, dto.MimeType); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to upload document to storage")
		}
		d.StorageKey = key
		d.MimeType = dto.MimeType
	}

	if err := uc.repo.CreateDocument(ctx, d); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to upload document")
	}
	return ToSignatureDocumentResponse(d), nil
}

func (uc *signaturesUseCase) DownloadDocument(ctx context.Context, id uuid.UUID) (*DocumentDownloadDTO, error) {
	d, err := uc.repo.GetDocumentByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get document")
	}
	if d == nil {
		return nil, apperrors.NewNotFound("Document not found")
	}

	if d.StorageKey != "" {
		if !uc.storage.Enabled() {
			return nil, apperrors.NewServiceUnavailable("File storage is not configured. Please set R2 credentials in the backend .env file.")
		}
		data, err := uc.storage.Download(ctx, d.StorageKey)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to download document from storage")
		}
		return &DocumentDownloadDTO{
			SignatureDocumentResponseDTO: *ToSignatureDocumentResponse(d),
			ContentType:                  d.MimeType,
			Binary:                       data,
		}, nil
	}

	return &DocumentDownloadDTO{
		SignatureDocumentResponseDTO: *ToSignatureDocumentResponse(d),
	}, nil
}

func (uc *signaturesUseCase) GetDocumentByID(ctx context.Context, id uuid.UUID) (*DocumentWithSignersDTO, error) {
	d, err := uc.repo.GetDocumentByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get document")
	}
	if d == nil {
		return nil, apperrors.NewNotFound("Document not found")
	}
	signers, err := uc.repo.ListRequestsByDocument(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list signature requests")
	}
	return &DocumentWithSignersDTO{
		SignatureDocumentResponseDTO: *ToSignatureDocumentResponse(d),
		Signers:                      ToSignatureRequestResponseList(signers),
	}, nil
}

func (uc *signaturesUseCase) ListDocuments(ctx context.Context, query types.PaginationQuery) ([]SignatureDocumentResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	docs, total, err := uc.repo.ListDocuments(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list documents")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToSignatureDocumentResponseList(docs), meta, nil
}

func (uc *signaturesUseCase) AddSigner(ctx context.Context, documentID uuid.UUID, dto AddSignerDTO) (*SignatureRequestResponseDTO, error) {
	d, err := uc.repo.GetDocumentByID(ctx, documentID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get document")
	}
	if d == nil {
		return nil, apperrors.NewNotFound("Document not found")
	}
	if dto.SignerName == "" {
		return nil, apperrors.NewBadRequest("Signer name is required")
	}
	if dto.SignerEmail == "" {
		return nil, apperrors.NewBadRequest("Signer email is required")
	}
	if d.Status == "signed" {
		return nil, apperrors.NewConflict("Document is already fully signed; signers cannot be added")
	}
	if d.Status == "declined" {
		return nil, apperrors.NewConflict("Document has been declined; signers cannot be added")
	}

	r := &domain.SignatureRequest{
		DocumentID:  d.ID,
		SignerName:  dto.SignerName,
		SignerEmail: dto.SignerEmail,
		Status:      "pending",
		OrderIndex:  dto.OrderIndex,
	}
	if err := uc.repo.CreateRequest(ctx, r); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to add signer")
	}

	if d.Status == "draft" {
		d.Status = "pending_signature"
		if err := uc.repo.UpdateDocument(ctx, d); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to update document status")
		}
	}
	return ToSignatureRequestResponse(r), nil
}

func (uc *signaturesUseCase) SignRequest(ctx context.Context, requestID uuid.UUID, callerEmail string) (*SignatureRequestResponseDTO, error) {
	r, err := uc.repo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get signature request")
	}
	if r == nil {
		return nil, apperrors.NewNotFound("Signature request not found")
	}
	if !strings.EqualFold(strings.TrimSpace(callerEmail), strings.TrimSpace(r.SignerEmail)) {
		return nil, apperrors.NewForbidden("You are not the designated signer for this request")
	}
	if r.Status == "signed" {
		return nil, apperrors.NewBadRequest("Signature request is already signed")
	}
	if r.Status == "declined" {
		return nil, apperrors.NewBadRequest("Signature request has been declined")
	}

	now := time.Now().Format(time.RFC3339)
	r.Status = "signed"
	r.SignedAt = &now
	if err := uc.repo.UpdateRequest(ctx, r); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to sign request")
	}

	d, err := uc.repo.GetDocumentByID(ctx, r.DocumentID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get document")
	}
	if d != nil {
		all, err := uc.repo.ListRequestsByDocument(ctx, d.ID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to list signature requests")
		}
		allSigned := len(all) > 0
		for _, sr := range all {
			if sr.Status != "signed" {
				allSigned = false
				break
			}
		}
		if allSigned && d.Status != "signed" {
			d.Status = "signed"
			if err := uc.repo.UpdateDocument(ctx, d); err != nil {
				return nil, apperrors.NewInternal(err, "Failed to update document status")
			}
		}
	}
	return ToSignatureRequestResponse(r), nil
}

func (uc *signaturesUseCase) DeclineRequest(ctx context.Context, requestID uuid.UUID) (*SignatureRequestResponseDTO, error) {
	r, err := uc.repo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get signature request")
	}
	if r == nil {
		return nil, apperrors.NewNotFound("Signature request not found")
	}
	if r.Status == "signed" {
		return nil, apperrors.NewBadRequest("Signature request is already signed")
	}
	r.Status = "declined"
	if err := uc.repo.UpdateRequest(ctx, r); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to decline request")
	}

	d, err := uc.repo.GetDocumentByID(ctx, r.DocumentID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get document")
	}
	if d != nil && d.Status != "declined" {
		d.Status = "declined"
		if err := uc.repo.UpdateDocument(ctx, d); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to update document status")
		}
	}
	return ToSignatureRequestResponse(r), nil
}

// SeedInitialData populates sample documents and signature requests on first boot
func (uc *signaturesUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountDocuments(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// Document 1: fully signed
	doc1, err := uc.UploadDocument(ctx, UploadDocumentDTO{
		Title:      "Penawaran Harga Resmi Rak Gudang Heavy Duty",
		FileName:   "QUO-2026-0814.pdf",
		UploadedBy: "Nicholas Tantra",
	})
	if err == nil {
		s1, err := uc.AddSigner(ctx, doc1.ID, AddSignerDTO{SignerName: "Nicholas Tantra", SignerEmail: "nicholas@divine.co.id", OrderIndex: 0})
		if err == nil {
			_, _ = uc.SignRequest(ctx, s1.ID, s1.SignerEmail)
		}
		s2, err := uc.AddSigner(ctx, doc1.ID, AddSignerDTO{SignerName: "Ir. Bambang Sudiro", SignerEmail: "bambang@grahakonstruksi.co.id", OrderIndex: 1})
		if err == nil {
			_, _ = uc.SignRequest(ctx, s2.ID, s2.SignerEmail)
		}
	}

	// Document 2: pending
	doc2, err := uc.UploadDocument(ctx, UploadDocumentDTO{
		Title:      "Perjanjian Kerja Waktu Tertentu (PKWT) - Agus Pratama",
		FileName:   "CTR-2026-003.pdf",
		UploadedBy: "Nicholas Tantra",
	})
	if err == nil {
		_, _ = uc.AddSigner(ctx, doc2.ID, AddSignerDTO{SignerName: "Nicholas Tantra", SignerEmail: "nicholas@divine.co.id", OrderIndex: 0})
		s2, err := uc.AddSigner(ctx, doc2.ID, AddSignerDTO{SignerName: "Agus Pratama", SignerEmail: "agus.pratama@divine.co.id", OrderIndex: 1})
		if err == nil {
			_, _ = uc.SignRequest(ctx, s2.ID, s2.SignerEmail)
		}
	}

	return nil
}
