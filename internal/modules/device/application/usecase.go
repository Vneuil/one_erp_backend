package application

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/device/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

var validDeviceTypes = map[string]bool{
	"Smart Attendance Terminal": true,
	"NFC Card Reader":           true,
	"POS Cashier Touchscreen":   true,
	"Barcode Gate Scanner":      true,
}

var validStatuses = map[string]bool{
	"online":  true,
	"offline": true,
	"syncing": true,
}

type DeviceUseCase interface {
	CreateDevice(ctx context.Context, dto CreateDeviceDTO) (*DeviceResponseDTO, error)
	GetDeviceByID(ctx context.Context, id uuid.UUID) (*DeviceResponseDTO, error)
	ListDevices(ctx context.Context, query types.PaginationQuery) ([]DeviceResponseDTO, types.PaginationMeta, error)
	UpdateDevice(ctx context.Context, id uuid.UUID, dto UpdateDeviceDTO) (*DeviceResponseDTO, error)
	PingDevice(ctx context.Context, id uuid.UUID) (*DeviceResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type deviceUseCase struct {
	repo domain.DeviceRepository
}

func NewDeviceUseCase(repo domain.DeviceRepository) DeviceUseCase {
	return &deviceUseCase{repo: repo}
}

func (uc *deviceUseCase) CreateDevice(ctx context.Context, dto CreateDeviceDTO) (*DeviceResponseDTO, error) {
	if dto.DeviceSerial == "" {
		return nil, apperrors.NewBadRequest("Device serial is required")
	}
	if dto.Name == "" {
		return nil, apperrors.NewBadRequest("Device name is required")
	}
	if dto.DeviceType == "" {
		dto.DeviceType = "Smart Attendance Terminal"
	}
	if !validDeviceTypes[dto.DeviceType] {
		return nil, apperrors.NewBadRequest("Device type must be one of Smart Attendance Terminal, NFC Card Reader, POS Cashier Touchscreen, Barcode Gate Scanner")
	}

	existing, err := uc.repo.GetDeviceBySerial(ctx, dto.DeviceSerial)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check device serial")
	}
	if existing != nil {
		return nil, apperrors.NewBadRequest("Device serial already registered")
	}

	d := &domain.Device{
		CompanyID:    dto.CompanyID,
		DeviceSerial: dto.DeviceSerial,
		Name:         dto.Name,
		DeviceType:   dto.DeviceType,
		Location:     dto.Location,
		IPAddress:    dto.IPAddress,
		Status:       "syncing",
	}

	if err := uc.repo.CreateDevice(ctx, d); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create device")
	}
	return ToDeviceResponse(d), nil
}

func (uc *deviceUseCase) GetDeviceByID(ctx context.Context, id uuid.UUID) (*DeviceResponseDTO, error) {
	d, err := uc.repo.GetDeviceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get device")
	}
	if d == nil {
		return nil, apperrors.NewNotFound("Device not found")
	}
	return ToDeviceResponse(d), nil
}

func (uc *deviceUseCase) ListDevices(ctx context.Context, query types.PaginationQuery) ([]DeviceResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	devices, total, err := uc.repo.ListDevices(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list devices")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToDeviceResponseList(devices), meta, nil
}

func (uc *deviceUseCase) UpdateDevice(ctx context.Context, id uuid.UUID, dto UpdateDeviceDTO) (*DeviceResponseDTO, error) {
	d, err := uc.repo.GetDeviceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get device")
	}
	if d == nil {
		return nil, apperrors.NewNotFound("Device not found")
	}
	if dto.Name != "" {
		d.Name = dto.Name
	}
	if dto.DeviceType != "" {
		if !validDeviceTypes[dto.DeviceType] {
			return nil, apperrors.NewBadRequest("Device type must be one of Smart Attendance Terminal, NFC Card Reader, POS Cashier Touchscreen, Barcode Gate Scanner")
		}
		d.DeviceType = dto.DeviceType
	}
	if dto.Location != "" {
		d.Location = dto.Location
	}
	if dto.IPAddress != "" {
		d.IPAddress = dto.IPAddress
	}
	if dto.Status != "" {
		if !validStatuses[dto.Status] {
			return nil, apperrors.NewBadRequest("Status must be one of online, offline, syncing")
		}
		d.Status = dto.Status
	}

	if err := uc.repo.UpdateDevice(ctx, d); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update device")
	}
	return ToDeviceResponse(d), nil
}

func (uc *deviceUseCase) PingDevice(ctx context.Context, id uuid.UUID) (*DeviceResponseDTO, error) {
	d, err := uc.repo.GetDeviceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get device")
	}
	if d == nil {
		return nil, apperrors.NewNotFound("Device not found")
	}
	now := time.Now()
	d.LastHeartbeat = &now
	d.Status = "online"
	if err := uc.repo.UpdateDevice(ctx, d); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to ping device")
	}
	return ToDeviceResponse(d), nil
}

// SeedInitialData populates a few sample business terminals on first boot
func (uc *deviceUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountDevices(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	devices := []CreateDeviceDTO{
		{
			DeviceSerial: "TRM-ATT-CKR-01",
			Name:         "Terminal Presensi Lobby Utama Cikarang",
			DeviceType:   "Smart Attendance Terminal",
			Location:     "Head Office Cikarang",
			IPAddress:    "192.168.10.45",
		},
		{
			DeviceSerial: "TRM-POS-CKR-01",
			Name:         "Mesin Kasir Retail POS #01",
			DeviceType:   "POS Cashier Touchscreen",
			Location:     "Outlet Retail Cikarang",
			IPAddress:    "192.168.10.88",
		},
		{
			DeviceSerial: "TRM-NFC-WMS-02",
			Name:         "NFC Reader Loading Dock Gudang B",
			DeviceType:   "NFC Card Reader",
			Location:     "Gudang Utama Cikarang",
			IPAddress:    "192.168.20.12",
		},
		{
			DeviceSerial: "TRM-ATT-SBY-01",
			Name:         "Terminal Presensi Hub Surabaya",
			DeviceType:   "Smart Attendance Terminal",
			Location:     "Hub Distribusi Surabaya",
			IPAddress:    "10.20.5.11",
		},
	}

	for _, dto := range devices {
		res, err := uc.CreateDevice(ctx, dto)
		if err != nil {
			continue
		}
		id, parseErr := uuid.Parse(res.ID.String())
		if parseErr == nil {
			_, _ = uc.PingDevice(ctx, id)
		}
	}
	return nil
}
