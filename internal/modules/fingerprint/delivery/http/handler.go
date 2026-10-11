package http

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/license"
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/fingerprint/application"
	"github.com/divinecoid/one-backend/internal/modules/fingerprint/domain"
	"github.com/divinecoid/one-backend/internal/modules/fingerprint/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db      *gorm.DB // control-plane database (machine registry)
	manager *tenantMgr.Manager
}

func NewHandler(db *gorm.DB, manager *tenantMgr.Manager) *Handler {
	return &Handler{db: db, manager: manager}
}

func (h *Handler) service(tenantDB *gorm.DB) *application.Service {
	return application.NewService(infrastructure.NewRepository(tenantDB), infrastructure.NewRegistryRepository(h.db))
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

// resolve returns the caller's service and company.
func (h *Handler) resolve(c *fiber.Ctx) (*application.Service, uuid.UUID, error) {
	tenantDB := middleware.TenantDB(c)
	claims := middleware.CurrentUser(c)
	if tenantDB == nil || claims == nil || claims.CompanyID == nil {
		return nil, uuid.Nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	return h.service(tenantDB), *claims.CompanyID, nil
}

// Authenticated endpoints

func (h *Handler) ListDevices(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	out, err := svc.ListDevices(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Machines retrieved", out)
}

func (h *Handler) RegisterDevice(c *fiber.Ctx) error {
	svc, companyID, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in application.DeviceInput
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	out, err := svc.RegisterDevice(h.ctx(c), companyID, middleware.CurrentTenantID(c), in)
	if err != nil {
		return err
	}
	return response.Created(c, "Machine registered", out)
}

func (h *Handler) UpdateDevice(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in application.DeviceInput
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	out, err := svc.UpdateDevice(h.ctx(c), c.Params("serial"), in)
	if err != nil {
		return err
	}
	return response.OK(c, "Machine updated", out)
}

func (h *Handler) RotateToken(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	out, err := svc.RotateToken(h.ctx(c), c.Params("serial"))
	if err != nil {
		return err
	}
	return response.OK(c, "Token rotated", out)
}

func (h *Handler) DeleteDevice(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	if err := svc.DeleteDevice(h.ctx(c), c.Params("serial")); err != nil {
		return err
	}
	return response.OK(c, "Machine removed", nil)
}

func (h *Handler) ListPinMaps(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	out, err := svc.ListPinMaps(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "PIN mappings retrieved", out)
}

func (h *Handler) MapPIN(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in application.PinMapInput
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	m, res, err := svc.MapPIN(h.ctx(c), in)
	if err != nil {
		return err
	}
	return response.OK(c, "PIN mapped", fiber.Map{"mapping": m, "reprocessed": res})
}

func (h *Handler) DeletePinMap(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid id format")
	}
	if err := svc.DeletePinMap(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Mapping removed", nil)
}

func (h *Handler) ListPunches(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	out, err := svc.ListPunches(h.ctx(c), c.Query("unresolved") == "true", c.Query("from"), c.Query("to"))
	if err != nil {
		return err
	}
	return response.OK(c, "Punches retrieved", out)
}

func (h *Handler) Reprocess(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	res, err := svc.Reprocess(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Punches reprocessed", res)
}

type importRow struct {
	PIN  string `json:"pin"`
	Time string `json:"time"`
}

// Import takes punches from a downloaded machine log (CSV/Excel parsed by the client).
func (h *Handler) Import(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in struct {
		Label string      `json:"label"`
		Rows  []importRow `json:"rows"`
	}
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if len(in.Rows) == 0 || len(in.Rows) > 20000 {
		return apperrors.NewBadRequest("rows must hold between 1 and 20000 punches")
	}
	raws, problems := toRaw(in.Rows)
	label := strings.TrimSpace(in.Label)
	if label == "" {
		label = "import"
	}
	res, err := svc.Ingest(h.ctx(c), middleware.CurrentTenantID(c), label, raws)
	if err != nil {
		return err
	}
	return response.OK(c, "Punches imported", fiber.Map{"result": res, "problems": problems})
}

func toRaw(rows []importRow) ([]application.RawPunch, []string) {
	var raws []application.RawPunch
	var problems []string
	for i, r := range rows {
		at, err := application.ParseTime(r.Time)
		if err != nil {
			problems = append(problems, fmt.Sprintf("row %d: %v", i+1, err))
			continue
		}
		raws = append(raws, application.RawPunch{PIN: r.PIN, At: at})
	}
	return raws, problems
}

// Machine-facing (no login) endpoints

type machine struct {
	svc *application.Service
	ctx context.Context
	dev *domain.Device
	reg *domain.Registry
}

// authenticate resolves a registered machine. Token-based protocols pass
// requireToken; ADMS machines cannot send one, so they are matched by serial
// (and the optional allowed IP) only.
func (h *Handler) authenticate(c *fiber.Ctx, serial, token string, requireToken bool) (*machine, error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return nil, apperrors.NewBadRequest("device serial is required")
	}
	reg, err := infrastructure.NewRegistryRepository(h.db).Get(c.UserContext(), serial)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Registry lookup failed")
	}
	if reg == nil || !reg.Active {
		return nil, apperrors.NewForbidden("Unknown or disabled machine")
	}
	if requireToken && !application.TokenMatches(token, reg.TokenHash) {
		return nil, apperrors.NewUnauthorized("Invalid device token")
	}
	if reg.AllowedIP != "" && !ipAllowed(c.IP(), reg.AllowedIP) {
		return nil, apperrors.NewForbidden("This machine is not allowed from this address")
	}
	tenantDB, err := h.manager.GetDB(c.UserContext(), reg.CompanyID)
	if err != nil {
		slog.Error("fingerprint: tenant database unavailable", "company", reg.CompanyID, "error", err)
		return nil, apperrors.NewInternal(err, "Company database unavailable")
	}
	if active, err := license.IsActive(c.UserContext(), tenantDB, reg.CompanyID); err == nil && !active {
		return nil, apperrors.NewPaymentRequired("License expired")
	}
	svc := h.service(tenantDB)
	ctx := tenantctx.WithTenantID(c.UserContext(), reg.TenantID)
	dev, err := infrastructure.NewRepository(tenantDB).GetDevice(ctx, serial)
	if err != nil || dev == nil || !dev.Active {
		return nil, apperrors.NewForbidden("Machine is not active in this company")
	}
	_ = infrastructure.NewRepository(tenantDB).TouchDevice(ctx, serial, c.IP(), nil)
	return &machine{svc: svc, ctx: ctx, dev: dev, reg: reg}, nil
}

func ipAllowed(ip, allowed string) bool {
	for _, a := range strings.Split(allowed, ",") {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if a == ip {
			return true
		}
		if _, n, err := net.ParseCIDR(a); err == nil && n.Contains(net.ParseIP(ip)) {
			return true
		}
	}
	return false
}

// GenericPush accepts {"punches":[{"pin","time"}]} with X-Device-Serial and
// X-Device-Token headers, for bridges and custom clients.
func (h *Handler) GenericPush(c *fiber.Ctx) error {
	m, err := h.authenticate(c, c.Get("X-Device-Serial"), c.Get("X-Device-Token"), true)
	if err != nil {
		return err
	}
	var in struct {
		Punches []importRow `json:"punches"`
	}
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if len(in.Punches) == 0 || len(in.Punches) > 5000 {
		return apperrors.NewBadRequest("punches must hold between 1 and 5000 entries")
	}
	raws, problems := toRaw(in.Punches)
	res, err := m.svc.Ingest(m.ctx, m.reg.TenantID, m.dev.Serial, raws)
	if err != nil {
		return err
	}
	return response.OK(c, "Punches accepted", fiber.Map{"result": res, "problems": problems})
}

// ADMS ("iclock") protocol: machines are configured with the server address
// only and call fixed paths with their serial number as SN.

func admsText(c *fiber.Ctx, status int, body string) error {
	c.Set("Content-Type", "text/plain")
	return c.Status(status).SendString(body)
}

func (h *Handler) ADMSData(c *fiber.Ctx) error {
	serial := c.Query("SN")
	m, err := h.authenticate(c, serial, "", false)
	if err != nil {
		return admsText(c, fiber.StatusForbidden, "ERROR: unknown device")
	}
	if c.Method() == fiber.MethodGet {
		return admsText(c, fiber.StatusOK, "GET OPTION FROM: "+serial+"\nATTLOGStamp=0\nOPERLOGStamp=0\nErrorDelay=30\nDelay=10\nTransTimes=00:00;14:00\nTransInterval=1\nTransFlag=TransData AttLog\nTimeZone=7\nRealtime=1\nEncrypt=None\n")
	}
	if !strings.EqualFold(c.Query("table"), "ATTLOG") {
		return admsText(c, fiber.StatusOK, "OK")
	}
	raws, problems := application.ParseATTLOG(string(c.Body()))
	for _, p := range problems {
		slog.Warn("fingerprint: unreadable ADMS line", "serial", serial, "problem", p)
	}
	res, err := m.svc.Ingest(m.ctx, m.reg.TenantID, m.dev.Serial, raws)
	if err != nil {
		slog.Error("fingerprint: ingest failed", "serial", serial, "error", err)
		return admsText(c, fiber.StatusInternalServerError, "ERROR")
	}
	return admsText(c, fiber.StatusOK, fmt.Sprintf("OK: %d", res.Received))
}

func (h *Handler) ADMSGetRequest(c *fiber.Ctx) error {
	if _, err := h.authenticate(c, c.Query("SN"), "", false); err != nil {
		return admsText(c, fiber.StatusForbidden, "ERROR: unknown device")
	}
	return admsText(c, fiber.StatusOK, "OK")
}

func (h *Handler) ADMSDeviceCmd(c *fiber.Ctx) error {
	return admsText(c, fiber.StatusOK, "OK")
}
