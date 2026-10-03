package application

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/hrops/domain"
	reimbursementDomain "github.com/divinecoid/one-backend/internal/modules/reimbursement/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// WithClaims lets evidence uploads mark the claim as having a receipt attached.
func (s *Service) WithClaims(r reimbursementDomain.ReimbursementRepository) *Service {
	s.claims = r
	return s
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// ------------------------------------------------------ reimbursement evidence

type EvidenceInput struct {
	ClaimID        uuid.UUID
	Title, FileRef string
	Amount         float64
}

// AddEvidence attaches a receipt to a claim. Only the claimant (or a privileged
// user) may add to it, and only while the claim is still pending.
func (s *Service) AddEvidence(ctx context.Context, c Caller, in EvidenceInput) (*domain.ReimbursementEvidence, error) {
	if s.claims == nil {
		return nil, apperrors.NewBadRequest("Reimbursements are not available")
	}
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.FileRef) == "" || in.Amount < 0 {
		return nil, apperrors.NewBadRequest("title and fileRef are required, and amount cannot be negative")
	}
	claim, err := s.claims.GetClaimByID(ctx, in.ClaimID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load claim")
	}
	if claim == nil {
		return nil, apperrors.NewNotFound("Claim not found")
	}
	if !c.Privileged && !strings.EqualFold(claim.RequesterEmail, c.Email) {
		return nil, apperrors.NewForbidden("You can only add evidence to your own claims")
	}
	if claim.Status != "pending_approval" {
		return nil, apperrors.NewConflict("Evidence can only be added while the claim is pending approval")
	}
	v := &domain.ReimbursementEvidence{ClaimID: in.ClaimID, Title: strings.TrimSpace(in.Title), FileRef: strings.TrimSpace(in.FileRef), Amount: round2(in.Amount), UploadedByEmail: c.Email}
	if err := s.repo.CreateEvidence(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save evidence")
	}
	if !claim.ReceiptAttached {
		claim.ReceiptAttached = true
		if err := s.claims.UpdateClaim(ctx, claim); err != nil {
			return v, apperrors.NewInternal(err, "Evidence saved but the claim could not be marked as having a receipt")
		}
	}
	return v, nil
}

// ListEvidence lists a claim's evidence to its owner or a privileged user.
func (s *Service) ListEvidence(ctx context.Context, c Caller, claimID uuid.UUID) ([]domain.ReimbursementEvidence, error) {
	if s.claims == nil {
		return nil, apperrors.NewBadRequest("Reimbursements are not available")
	}
	claim, err := s.claims.GetClaimByID(ctx, claimID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load claim")
	}
	if claim == nil {
		return nil, apperrors.NewNotFound("Claim not found")
	}
	if !c.Privileged && !strings.EqualFold(claim.RequesterEmail, c.Email) {
		return nil, apperrors.NewForbidden("You can only view evidence on your own claims")
	}
	out, err := s.repo.ListEvidence(ctx, claimID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list evidence")
	}
	return out, nil
}

// --------------------------------------------------------------------- canteen

func (s *Service) CreateCanteenItem(ctx context.Context, name string, price float64) (*domain.CanteenItem, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 150 || price <= 0 || math.IsNaN(price) {
		return nil, apperrors.NewBadRequest("name and a positive price are required")
	}
	v := &domain.CanteenItem{Name: name, Price: round2(price), IsActive: true}
	if err := s.repo.CreateCanteenItem(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save menu item")
	}
	return v, nil
}

func (s *Service) UpdateCanteenItem(ctx context.Context, id uuid.UUID, price *float64, active *bool) (*domain.CanteenItem, error) {
	v, err := s.repo.GetCanteenItem(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load menu item")
	}
	if v == nil {
		return nil, apperrors.NewNotFound("Menu item not found")
	}
	if price != nil {
		if *price <= 0 {
			return nil, apperrors.NewBadRequest("price must be positive")
		}
		v.Price = round2(*price) // past orders keep the price they were made at
	}
	if active != nil {
		v.IsActive = *active
	}
	if err := s.repo.UpdateCanteenItem(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update menu item")
	}
	return v, nil
}

func (s *Service) ListCanteenItems(ctx context.Context, activeOnly bool) ([]domain.CanteenItem, error) {
	out, err := s.repo.ListCanteenItems(ctx, activeOnly)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list menu")
	}
	return out, nil
}

const maxCanteenQty = 20

// OrderMeal orders for the caller's own employee record on the given date (default
// today). Ordering for other days is limited to today and later, so past
// consumption cannot be back-filled into a payroll month that may be closed.
func (s *Service) OrderMeal(ctx context.Context, c Caller, nip string, itemID uuid.UUID, qty int, date string) (*domain.CanteenOrder, error) {
	if qty < 1 || qty > maxCanteenQty {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("quantity must be 1-%d", maxCanteenQty))
	}
	today := s.now().In(zone).Format("2006-01-02")
	if date == "" {
		date = today
	}
	if !ValidDate(date) || date < today {
		return nil, apperrors.NewBadRequest("date must be today or later (YYYY-MM-DD)")
	}
	item, err := s.repo.GetCanteenItem(ctx, itemID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load menu item")
	}
	if item == nil || !item.IsActive {
		return nil, apperrors.NewNotFound("Menu item not found")
	}
	emp, err := s.resolveEmployee(ctx, c, nip)
	if err != nil {
		return nil, err
	}
	v := &domain.CanteenOrder{NIP: emp.NIP, EmployeeName: emp.Name, ItemID: item.ID, ItemName: item.Name, Quantity: qty, UnitPrice: item.Price,
		Amount: round2(item.Price * float64(qty)), Date: date, Status: "ordered", CreatedByEmail: c.Email}
	if err := s.repo.CreateCanteenOrder(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save the order")
	}
	return v, nil
}

// CancelMeal cancels an order that is still to come (its date is today or later).
func (s *Service) CancelMeal(ctx context.Context, c Caller, id uuid.UUID) (*domain.CanteenOrder, error) {
	v, err := s.repo.GetCanteenOrder(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the order")
	}
	if v == nil {
		return nil, apperrors.NewNotFound("Order not found")
	}
	if !c.Privileged && !strings.EqualFold(v.CreatedByEmail, c.Email) {
		return nil, apperrors.NewForbidden("You can only cancel your own orders")
	}
	if v.Status != "ordered" {
		return nil, apperrors.NewConflict("This order is already " + v.Status)
	}
	if v.Date < s.now().In(zone).Format("2006-01-02") {
		return nil, apperrors.NewConflict("Past orders cannot be cancelled; ask HR")
	}
	v.Status = "cancelled"
	if err := s.repo.UpdateCanteenOrder(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to cancel the order")
	}
	return v, nil
}

// MyMeals lists the caller's orders in a month with their total.
func (s *Service) MyMeals(ctx context.Context, c Caller, period string) ([]domain.CanteenOrder, float64, error) {
	if _, err := time.Parse("2006-01", period); err != nil {
		return nil, 0, apperrors.NewBadRequest("period must be YYYY-MM")
	}
	nip, err := s.OwnNIP(ctx, c)
	if err != nil {
		return nil, 0, err
	}
	list, err := s.repo.ListCanteenOrders(ctx, nip, period)
	if err != nil {
		return nil, 0, apperrors.NewInternal(err, "Failed to list orders")
	}
	var total float64
	for _, o := range list {
		if o.Status == "ordered" {
			total += o.Amount
		}
	}
	return list, round2(total), nil
}

// CanteenSummary is the month's spend per employee (what payroll will deduct).
func (s *Service) CanteenSummary(ctx context.Context, period string) ([]domain.CanteenEmployeeTotal, error) {
	if _, err := time.Parse("2006-01", period); err != nil {
		return nil, apperrors.NewBadRequest("period must be YYYY-MM")
	}
	out, err := s.repo.CanteenTotals(ctx, period)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to summarise the canteen")
	}
	if out == nil {
		out = []domain.CanteenEmployeeTotal{}
	}
	return out, nil
}

// ---------------------------------------------------------------------- visits

// GeoPoint is a coordinate pair.
type GeoPoint struct{ Lat, Lng float64 }

// DistanceMeters is the great-circle distance between two points.
func DistanceMeters(a, b GeoPoint) float64 {
	const earth = 6371000.0
	rad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat, dLng := rad(b.Lat-a.Lat), rad(b.Lng-a.Lng)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(rad(a.Lat))*math.Cos(rad(b.Lat))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earth * 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}

// NearestNeighborOrder returns the indexes of pts in a greedy nearest-next order
// starting from start. It is a heuristic (not the shortest possible tour), but it
// is deterministic, fast and good enough for a day's handful of stops.
func NearestNeighborOrder(start GeoPoint, pts []GeoPoint) []int {
	order := make([]int, 0, len(pts))
	used := make([]bool, len(pts))
	cur := start
	for len(order) < len(pts) {
		best, bestD := -1, math.MaxFloat64
		for i, p := range pts {
			if used[i] {
				continue
			}
			if d := DistanceMeters(cur, p); d < bestD {
				best, bestD = i, d
			}
		}
		used[best] = true
		order = append(order, best)
		cur = pts[best]
	}
	return order
}

// RouteLength is the total driving-line distance start -> stops in order, in metres.
func RouteLength(start GeoPoint, pts []GeoPoint) float64 {
	cur, total := start, 0.0
	for _, p := range pts {
		total += DistanceMeters(cur, p)
		cur = p
	}
	return total
}

type StopInput struct {
	CustomerName, Address string
	Latitude, Longitude   float64
	RadiusMeters          int
}

func validLatLng(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 && !(lat == 0 && lng == 0)
}

// PlanVisits adds stops to an employee's plan for a date (today or later).
func (s *Service) PlanVisits(ctx context.Context, c Caller, nip, date string, stops []StopInput) ([]domain.VisitStop, error) {
	today := s.now().In(zone).Format("2006-01-02")
	if !ValidDate(date) || date < today {
		return nil, apperrors.NewBadRequest("date must be today or later (YYYY-MM-DD)")
	}
	if len(stops) == 0 || len(stops) > 30 {
		return nil, apperrors.NewBadRequest("provide between 1 and 30 stops")
	}
	emp, err := s.resolveEmployee(ctx, c, nip)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.ListStops(ctx, emp.NIP, date)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the current plan")
	}
	seq := len(existing)
	out := make([]domain.VisitStop, 0, len(stops))
	for _, st := range stops {
		if strings.TrimSpace(st.CustomerName) == "" || !validLatLng(st.Latitude, st.Longitude) {
			return nil, apperrors.NewBadRequest("every stop needs a customer name and valid coordinates")
		}
		radius := st.RadiusMeters
		if radius == 0 {
			radius = 100
		}
		if radius < 20 || radius > 5000 {
			return nil, apperrors.NewBadRequest("radiusMeters must be 20-5000")
		}
		seq++
		out = append(out, domain.VisitStop{NIP: emp.NIP, EmployeeName: emp.Name, Date: date, Seq: seq, CustomerName: strings.TrimSpace(st.CustomerName),
			Address: strings.TrimSpace(st.Address), Latitude: st.Latitude, Longitude: st.Longitude, RadiusMeters: radius, Status: "planned"})
	}
	if err := s.repo.CreateStops(ctx, out); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save the plan")
	}
	return out, nil
}

// VisitPlan is a day's stops with the planned route length.
type VisitPlan struct {
	Date        string             `json:"date"`
	Stops       []domain.VisitStop `json:"stops"`
	Visited     int                `json:"visited"`
	PlannedLenM float64            `json:"plannedRouteMeters"`
}

func (s *Service) planFor(ctx context.Context, nip, date string) (*VisitPlan, error) {
	stops, err := s.repo.ListStops(ctx, nip, date)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the plan")
	}
	p := &VisitPlan{Date: date, Stops: stops}
	if p.Stops == nil {
		p.Stops = []domain.VisitStop{}
	}
	var pts []GeoPoint
	for _, st := range stops {
		if st.Status == "visited" {
			p.Visited++
		}
		pts = append(pts, GeoPoint{st.Latitude, st.Longitude})
	}
	if len(pts) > 1 {
		p.PlannedLenM = math.Round(RouteLength(pts[0], pts[1:]))
	}
	return p, nil
}

func (s *Service) MyPlan(ctx context.Context, c Caller, date string) (*VisitPlan, error) {
	if !ValidDate(date) {
		return nil, apperrors.NewBadRequest("date must be YYYY-MM-DD")
	}
	nip, err := s.OwnNIP(ctx, c)
	if err != nil {
		return nil, err
	}
	return s.planFor(ctx, nip, date)
}

// TeamPlans lists every employee's stops for a date (or one employee's).
func (s *Service) TeamPlans(ctx context.Context, date, nip string) ([]domain.VisitStop, error) {
	if !ValidDate(date) {
		return nil, apperrors.NewBadRequest("date must be YYYY-MM-DD")
	}
	out, err := s.repo.ListStops(ctx, nip, date)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load plans")
	}
	return out, nil
}

// OptimizeRoute reorders the still-planned stops of the caller's day by nearest
// neighbour from the given start point; visited and skipped stops keep their place.
func (s *Service) OptimizeRoute(ctx context.Context, c Caller, date string, start GeoPoint) (*VisitPlan, error) {
	if !validLatLng(start.Lat, start.Lng) {
		return nil, apperrors.NewBadRequest("a valid start location is required")
	}
	nip, err := s.OwnNIP(ctx, c)
	if err != nil {
		return nil, err
	}
	stops, err := s.repo.ListStops(ctx, nip, date)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the plan")
	}
	var planned []int
	var pts []GeoPoint
	for i, st := range stops {
		if st.Status == "planned" {
			planned = append(planned, i)
			pts = append(pts, GeoPoint{st.Latitude, st.Longitude})
		}
	}
	if len(planned) > 1 {
		// Keep the sequence numbers the planned stops already own, and hand them out in the new order.
		seqs := make([]int, len(planned))
		for k, i := range planned {
			seqs[k] = stops[i].Seq
		}
		sort.Ints(seqs)
		for slot, k := range NearestNeighborOrder(start, pts) {
			stops[planned[k]].Seq = seqs[slot]
		}
		if err := s.repo.UpdateStops(ctx, stops); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to save the new order")
		}
	}
	return s.planFor(ctx, nip, date)
}

func (s *Service) ownStop(ctx context.Context, c Caller, id uuid.UUID) (*domain.VisitStop, error) {
	st, err := s.repo.GetStop(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the stop")
	}
	if st == nil {
		return nil, apperrors.NewNotFound("Stop not found")
	}
	nip, err := s.OwnNIP(ctx, c)
	if err != nil {
		return nil, err
	}
	if st.NIP != nip && !c.Privileged {
		return nil, apperrors.NewForbidden("This stop belongs to another employee")
	}
	return st, nil
}

// CheckIn confirms the employee is at the customer: the reported position must lie
// within the stop's radius, and only on the day of the plan.
func (s *Service) CheckIn(ctx context.Context, c Caller, id uuid.UUID, at GeoPoint) (*domain.VisitStop, error) {
	if !validLatLng(at.Lat, at.Lng) {
		return nil, apperrors.NewBadRequest("a valid position is required")
	}
	st, err := s.ownStop(ctx, c, id)
	if err != nil {
		return nil, err
	}
	if st.Status != "planned" {
		return nil, apperrors.NewConflict("This stop is already " + st.Status)
	}
	if st.Date != s.now().In(zone).Format("2006-01-02") {
		return nil, apperrors.NewBadRequest("You can only check in on the day of the visit (" + st.Date + ")")
	}
	d := DistanceMeters(at, GeoPoint{st.Latitude, st.Longitude})
	if d > float64(st.RadiusMeters) {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("You are %.0f m from %s; check in within %d m", d, st.CustomerName, st.RadiusMeters))
	}
	now := s.now()
	dist := math.Round(d)
	st.Status, st.CheckInAt, st.CheckInLat, st.CheckInLng, st.DistanceM = "visited", &now, &at.Lat, &at.Lng, &dist
	if err := s.repo.UpdateStops(ctx, []domain.VisitStop{*st}); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record the check-in")
	}
	return st, nil
}

// SkipStop marks a planned stop as not visited, with a reason.
func (s *Service) SkipStop(ctx context.Context, c Caller, id uuid.UUID, reason string) (*domain.VisitStop, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, apperrors.NewBadRequest("A reason is required")
	}
	st, err := s.ownStop(ctx, c, id)
	if err != nil {
		return nil, err
	}
	if st.Status != "planned" {
		return nil, apperrors.NewConflict("This stop is already " + st.Status)
	}
	st.Status, st.Note = "skipped", reason
	if err := s.repo.UpdateStops(ctx, []domain.VisitStop{*st}); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update the stop")
	}
	return st, nil
}
