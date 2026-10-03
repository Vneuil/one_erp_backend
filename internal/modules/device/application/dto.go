package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/device/domain"
	"github.com/google/uuid"
)

type CreateDeviceDTO struct {
	DeviceSerial string     `json:"deviceSerial"`
	Name         string     `json:"name"`
	DeviceType   string     `json:"deviceType"`
	Location     string     `json:"location"`
	IPAddress    string     `json:"ipAddress"`
	CompanyID    *uuid.UUID `json:"companyId,omitempty"`
}

type UpdateDeviceDTO struct {
	Name       string `json:"name"`
	DeviceType string `json:"deviceType"`
	Location   string `json:"location"`
	IPAddress  string `json:"ipAddress"`
	Status     string `json:"status"`
}

type DeviceResponseDTO struct {
	ID            uuid.UUID  `json:"id"`
	CompanyID     *uuid.UUID `json:"companyId,omitempty"`
	DeviceSerial  string     `json:"deviceSerial"`
	Name          string     `json:"name"`
	DeviceType    string     `json:"deviceType"`
	Location      string     `json:"location"`
	IPAddress     string     `json:"ipAddress"`
	LastHeartbeat *time.Time `json:"lastHeartbeat,omitempty"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"createdAt"`
}

func ToDeviceResponse(d *domain.Device) *DeviceResponseDTO {
	if d == nil {
		return nil
	}
	return &DeviceResponseDTO{
		ID:            d.ID,
		CompanyID:     d.CompanyID,
		DeviceSerial:  d.DeviceSerial,
		Name:          d.Name,
		DeviceType:    d.DeviceType,
		Location:      d.Location,
		IPAddress:     d.IPAddress,
		LastHeartbeat: d.LastHeartbeat,
		Status:        effectiveStatus(d, time.Now()),
		CreatedAt:     d.CreatedAt,
	}
}

func ToDeviceResponseList(devices []domain.Device) []DeviceResponseDTO {
	result := make([]DeviceResponseDTO, len(devices))
	for i, d := range devices {
		result[i] = *ToDeviceResponse(&d)
	}
	return result
}

// heartbeatTimeout is how long a terminal may stay silent before it is
// reported offline. A stored "online" status alone is not trusted: it only says
// what the last heartbeat claimed, not that the device is still alive.
const heartbeatTimeout = 10 * time.Minute

// effectiveStatus derives the status to show from the stored status and the age
// of the last heartbeat. Only "online" is downgraded; explicit states such as
// "offline" or "syncing" are reported as stored.
func effectiveStatus(d *domain.Device, now time.Time) string {
	if d.Status != "online" {
		return d.Status
	}
	if d.LastHeartbeat == nil || now.Sub(*d.LastHeartbeat) > heartbeatTimeout {
		return "offline"
	}
	return d.Status
}
