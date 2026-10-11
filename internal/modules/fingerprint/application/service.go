package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/fingerprint/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// Service is bound to one company's database (repo) plus the control-plane
// registry used to resolve machines that call in without a login.
type Service struct {
	repo     domain.Repository
	registry domain.RegistryRepository
	now      func() time.Time
}

func NewService(repo domain.Repository, registry domain.RegistryRepository) *Service {
	return &Service{repo: repo, registry: registry, now: time.Now}
}

func HashToken(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}

func newToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// TokenMatches compares a presented token with the stored hash in constant time.
func TokenMatches(presented, hash string) bool {
	return presented != "" && subtle.ConstantTimeCompare([]byte(HashToken(presented)), []byte(hash)) == 1
}

type DeviceInput struct {
	Serial    string `json:"serial"`
	Name      string `json:"name"`
	Location  string `json:"location"`
	Vendor    string `json:"vendor"`
	Protocol  string `json:"protocol"`
	AllowedIP string `json:"allowedIp"`
	Active    *bool  `json:"active"`
}

// RegisteredDevice carries the one-time token of a newly registered or re-keyed machine.
type RegisteredDevice struct {
	Device *domain.Device `json:"device"`
	Token  string         `json:"token,omitempty"`
}

func (s *Service) RegisterDevice(ctx context.Context, companyID uuid.UUID, tenantID *uuid.UUID, in DeviceInput) (*RegisteredDevice, error) {
	serial := strings.TrimSpace(in.Serial)
	name := strings.TrimSpace(in.Name)
	if serial == "" || name == "" {
		return nil, apperrors.NewBadRequest("serial and name are required")
	}
	protocol := in.Protocol
	if protocol == "" {
		protocol = domain.ProtocolADMS
	}
	if protocol != domain.ProtocolADMS && protocol != domain.ProtocolGeneric {
		return nil, apperrors.NewBadRequest("protocol must be adms or generic")
	}
	if reg, err := s.registry.Get(ctx, serial); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check the serial")
	} else if reg != nil {
		return nil, apperrors.NewConflict("A machine with this serial number is already registered")
	}
	token := newToken()
	d := &domain.Device{Serial: serial, Name: name, Location: strings.TrimSpace(in.Location), Vendor: strings.TrimSpace(in.Vendor),
		Protocol: protocol, Active: true, AllowedIP: strings.TrimSpace(in.AllowedIP), CompanyID: &companyID, TenantID: tenantID}
	d.ID = uuid.New()
	if err := s.registry.Save(ctx, &domain.Registry{Serial: serial, CompanyID: companyID, TenantID: tenantID, TokenHash: HashToken(token), AllowedIP: d.AllowedIP, Active: true}); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to register the machine")
	}
	if err := s.repo.CreateDevice(ctx, d); err != nil {
		_ = s.registry.Delete(ctx, serial)
		return nil, apperrors.NewInternal(err, "Failed to save the machine")
	}
	return &RegisteredDevice{Device: d, Token: token}, nil
}

func (s *Service) device(ctx context.Context, serial string) (*domain.Device, error) {
	d, err := s.repo.GetDevice(ctx, serial)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the machine")
	}
	if d == nil {
		return nil, apperrors.NewNotFound("Machine not found")
	}
	return d, nil
}

func (s *Service) UpdateDevice(ctx context.Context, serial string, in DeviceInput) (*domain.Device, error) {
	d, err := s.device(ctx, serial)
	if err != nil {
		return nil, err
	}
	if v := strings.TrimSpace(in.Name); v != "" {
		d.Name = v
	}
	d.Location, d.Vendor, d.AllowedIP = strings.TrimSpace(in.Location), strings.TrimSpace(in.Vendor), strings.TrimSpace(in.AllowedIP)
	if in.Active != nil {
		d.Active = *in.Active
	}
	reg, err := s.registry.Get(ctx, serial)
	if err != nil || reg == nil {
		return nil, apperrors.NewInternal(err, "Failed to load the machine registry entry")
	}
	reg.AllowedIP, reg.Active = d.AllowedIP, d.Active
	if err := s.registry.Save(ctx, reg); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update the registry")
	}
	if err := s.repo.SaveDevice(ctx, d); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save the machine")
	}
	return d, nil
}

// RotateToken issues a new token for a machine, invalidating the old one.
func (s *Service) RotateToken(ctx context.Context, serial string) (*RegisteredDevice, error) {
	d, err := s.device(ctx, serial)
	if err != nil {
		return nil, err
	}
	reg, err := s.registry.Get(ctx, serial)
	if err != nil || reg == nil {
		return nil, apperrors.NewInternal(err, "Failed to load the machine registry entry")
	}
	token := newToken()
	reg.TokenHash = HashToken(token)
	if err := s.registry.Save(ctx, reg); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to rotate the token")
	}
	return &RegisteredDevice{Device: d, Token: token}, nil
}

func (s *Service) DeleteDevice(ctx context.Context, serial string) error {
	if _, err := s.device(ctx, serial); err != nil {
		return err
	}
	if err := s.registry.Delete(ctx, serial); err != nil {
		return apperrors.NewInternal(err, "Failed to remove the registry entry")
	}
	return s.repoDelete(ctx, serial)
}

func (s *Service) repoDelete(ctx context.Context, serial string) error {
	if err := s.repo.DeleteDevice(ctx, serial); err != nil {
		return apperrors.NewInternal(err, "Failed to delete the machine")
	}
	return nil
}

func (s *Service) ListDevices(ctx context.Context) ([]domain.Device, error) {
	out, err := s.repo.ListDevices(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list machines")
	}
	return out, nil
}

// PIN mapping

type PinMapInput struct {
	PIN string `json:"pin"`
	NIP string `json:"nip"`
}

// MapPIN links a PIN to an employee and resolves punches that were waiting for it.
func (s *Service) MapPIN(ctx context.Context, in PinMapInput) (*domain.PinMap, *IngestResult, error) {
	pin, nip := strings.TrimSpace(in.PIN), strings.TrimSpace(in.NIP)
	if pin == "" || nip == "" {
		return nil, nil, apperrors.NewBadRequest("pin and nip are required")
	}
	name, ok, err := s.repo.EmployeeName(ctx, nip)
	if err != nil {
		return nil, nil, apperrors.NewInternal(err, "Failed to look up the employee")
	}
	if !ok {
		return nil, nil, apperrors.NewBadRequest("No employee with NIP " + nip)
	}
	m := &domain.PinMap{PIN: pin, NIP: nip, EmployeeName: name}
	if old, err := s.repo.GetPinMap(ctx, pin); err != nil {
		return nil, nil, apperrors.NewInternal(err, "Failed to check the mapping")
	} else if old != nil {
		m.ID = old.ID
	} else {
		m.ID = uuid.New()
	}
	if err := s.repo.SavePinMap(ctx, m); err != nil {
		return nil, nil, apperrors.NewInternal(err, "Failed to save the mapping")
	}
	res, err := s.Reprocess(ctx)
	return m, res, err
}

func (s *Service) ListPinMaps(ctx context.Context) ([]domain.PinMap, error) {
	out, err := s.repo.ListPinMaps(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list mappings")
	}
	return out, nil
}

func (s *Service) DeletePinMap(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeletePinMap(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete the mapping")
	}
	return nil
}

// Ingest

type IngestResult struct {
	Received   int `json:"received"`
	Stored     int `json:"stored"`
	Duplicates int `json:"duplicates"`
	Unresolved int `json:"unresolved"`
	DaysSynced int `json:"daysSynced"`
}

// resolve maps a PIN to an employee: an explicit mapping first, then an
// employee whose NIP equals the PIN.
func (s *Service) resolve(ctx context.Context, pin string, cache map[string][2]string) (nip, name string) {
	if v, ok := cache[pin]; ok {
		return v[0], v[1]
	}
	if m, err := s.repo.GetPinMap(ctx, pin); err == nil && m != nil {
		nip, name = m.NIP, m.EmployeeName
	} else if n, ok, err := s.repo.EmployeeName(ctx, pin); err == nil && ok {
		nip, name = pin, n
	}
	cache[pin] = [2]string{nip, name}
	return nip, name
}

// Ingest stores raw punches from a machine (device serial may be a label such
// as "import" for files) and folds them into attendance.
func (s *Service) Ingest(ctx context.Context, tenantID *uuid.UUID, serial string, raws []RawPunch) (*IngestResult, error) {
	res := &IngestResult{Received: len(raws)}
	cache := map[string][2]string{}
	rows := make([]domain.Punch, 0, len(raws))
	for _, r := range raws {
		if strings.TrimSpace(r.PIN) == "" || r.At.IsZero() {
			continue
		}
		at := r.At.UTC().Truncate(time.Second)
		nip, name := s.resolve(ctx, r.PIN, cache)
		p := domain.Punch{TenantID: tenantID, DeviceSerial: serial, PIN: strings.TrimSpace(r.PIN), PunchedAt: at, Date: at.In(Zone).Format("2006-01-02"),
			Verify: r.Verify, State: r.State, NIP: nip, EmployeeName: name, Resolved: nip != ""}
		p.ID = uuid.New()
		rows = append(rows, p)
	}
	stored, err := s.repo.InsertPunches(ctx, rows)
	if err != nil {
		return res, apperrors.NewInternal(err, "Failed to store punches")
	}
	res.Stored, res.Duplicates = stored, len(rows)-stored
	days := map[[2]string][2]string{}
	for _, p := range rows {
		if p.NIP == "" {
			res.Unresolved++
			continue
		}
		days[[2]string{p.NIP, p.Date}] = [2]string{p.EmployeeName, serial}
	}
	for k, v := range days {
		if changed, err := s.syncDay(ctx, k[0], v[0], k[1], v[1]); err != nil {
			return res, err
		} else if changed {
			res.DaysSynced++
		}
	}
	if len(rows) > 0 {
		last := rows[len(rows)-1].PunchedAt
		for _, r := range rows {
			if r.PunchedAt.After(last) {
				last = r.PunchedAt
			}
		}
		_ = s.repo.TouchDevice(ctx, serial, "", &last)
	}
	return res, nil
}

// syncDay rebuilds one employee-day from every stored punch.
func (s *Service) syncDay(ctx context.Context, nip, name, date, serial string) (bool, error) {
	ps, err := s.repo.PunchesForDay(ctx, nip, date)
	if err != nil {
		return false, apperrors.NewInternal(err, "Failed to load the day's punches")
	}
	times := make([]time.Time, len(ps))
	for i, p := range ps {
		times[i] = p.PunchedAt
	}
	existing, err := s.repo.FindAttendance(ctx, nip, date)
	if err != nil {
		return false, apperrors.NewInternal(err, "Failed to load attendance")
	}
	location := serial
	if d, err := s.repo.GetDevice(ctx, serial); err == nil && d != nil {
		location = d.Name
		if d.Location != "" {
			location += " - " + d.Location
		}
	}
	att, changed := MergeDay(existing, times, nip, name, date, location)
	if !changed {
		return false, nil
	}
	if err := s.repo.SaveAttendance(ctx, att); err != nil {
		return false, apperrors.NewInternal(err, "Failed to save attendance")
	}
	return true, nil
}

// Reprocess resolves punches that were waiting for a PIN mapping and folds them into attendance.
func (s *Service) Reprocess(ctx context.Context) (*IngestResult, error) {
	pending, err := s.repo.UnresolvedPunches(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load unresolved punches")
	}
	res := &IngestResult{}
	cache := map[string][2]string{}
	days := map[[2]string][2]string{}
	for i := range pending {
		p := &pending[i]
		nip, name := s.resolve(ctx, p.PIN, cache)
		if nip == "" {
			res.Unresolved++
			continue
		}
		p.NIP, p.EmployeeName, p.Resolved = nip, name, true
		if err := s.repo.UpdatePunch(ctx, p); err != nil {
			return res, apperrors.NewInternal(err, "Failed to update a punch")
		}
		res.Stored++
		days[[2]string{nip, p.Date}] = [2]string{name, p.DeviceSerial}
	}
	for k, v := range days {
		if changed, err := s.syncDay(ctx, k[0], v[0], k[1], v[1]); err != nil {
			return res, err
		} else if changed {
			res.DaysSynced++
		}
	}
	return res, nil
}

func (s *Service) ListPunches(ctx context.Context, unresolvedOnly bool, from, to string) ([]domain.Punch, error) {
	out, err := s.repo.ListPunches(ctx, unresolvedOnly, from, to, 500)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list punches")
	}
	return out, nil
}
