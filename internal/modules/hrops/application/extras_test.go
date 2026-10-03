package application

import (
	"context"
	"math"
	"sort"
	"strings"
	"testing"
	"time"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/modules/hrops/domain"
	reimbursementDomain "github.com/divinecoid/one-backend/internal/modules/reimbursement/domain"
	"github.com/google/uuid"
)

func TestDistanceAndNearestNeighbor(t *testing.T) {
	jakarta, bandung := GeoPoint{-6.2088, 106.8456}, GeoPoint{-6.9175, 107.6191}
	if d := DistanceMeters(jakarta, bandung); d < 110_000 || d > 120_000 {
		t.Fatalf("Jakarta-Bandung should be about 116 km, got %.0f m", d)
	}
	if DistanceMeters(jakarta, jakarta) != 0 {
		t.Fatal("same point is zero")
	}
	// Points along a line: from the origin the greedy order must be nearest-first.
	origin := GeoPoint{0.001, 100}
	pts := []GeoPoint{{0.05, 100}, {0.01, 100}, {0.03, 100}, {0.02, 100}}
	got := NearestNeighborOrder(origin, pts)
	want := []int{1, 3, 2, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order %v want %v", got, want)
		}
	}
	ordered := []GeoPoint{pts[1], pts[3], pts[2], pts[0]}
	if RouteLength(origin, ordered) >= RouteLength(origin, pts) {
		t.Fatal("the optimised route must be shorter than the naive one")
	}
	if len(NearestNeighborOrder(origin, nil)) != 0 {
		t.Fatal("no stops, no order")
	}
}

type extrasRepo struct {
	domain.Repository
	evidence []domain.ReimbursementEvidence
	items    map[uuid.UUID]*domain.CanteenItem
	orders   map[uuid.UUID]*domain.CanteenOrder
	stops    []domain.VisitStop
}

func newExtras() *extrasRepo {
	return &extrasRepo{items: map[uuid.UUID]*domain.CanteenItem{}, orders: map[uuid.UUID]*domain.CanteenOrder{}}
}
func (r *extrasRepo) CreateEvidence(_ context.Context, v *domain.ReimbursementEvidence) error {
	r.evidence = append(r.evidence, *v)
	return nil
}
func (r *extrasRepo) GetCanteenItem(_ context.Context, id uuid.UUID) (*domain.CanteenItem, error) {
	return r.items[id], nil
}
func (r *extrasRepo) CreateCanteenItem(_ context.Context, v *domain.CanteenItem) error {
	v.ID = uuid.New()
	r.items[v.ID] = v
	return nil
}
func (r *extrasRepo) UpdateCanteenItem(context.Context, *domain.CanteenItem) error { return nil }
func (r *extrasRepo) CreateCanteenOrder(_ context.Context, v *domain.CanteenOrder) error {
	v.ID = uuid.New()
	r.orders[v.ID] = v
	return nil
}
func (r *extrasRepo) GetCanteenOrder(_ context.Context, id uuid.UUID) (*domain.CanteenOrder, error) {
	return r.orders[id], nil
}
func (r *extrasRepo) UpdateCanteenOrder(context.Context, *domain.CanteenOrder) error { return nil }
func (r *extrasRepo) ListStops(_ context.Context, nip, date string) ([]domain.VisitStop, error) {
	var out []domain.VisitStop
	for _, s := range r.stops {
		if s.NIP == nip && s.Date == date {
			out = append(out, s)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Seq < out[j].Seq }) // like the real repository
	return out, nil
}
func (r *extrasRepo) CreateStops(_ context.Context, v []domain.VisitStop) error {
	for i := range v {
		v[i].ID = uuid.New()
		r.stops = append(r.stops, v[i])
	}
	return nil
}
func (r *extrasRepo) GetStop(_ context.Context, id uuid.UUID) (*domain.VisitStop, error) {
	for i := range r.stops {
		if r.stops[i].ID == id {
			c := r.stops[i]
			return &c, nil
		}
	}
	return nil, nil
}
func (r *extrasRepo) UpdateStops(_ context.Context, v []domain.VisitStop) error {
	for _, u := range v {
		for i := range r.stops {
			if r.stops[i].ID == u.ID {
				r.stops[i] = u
			}
		}
	}
	return nil
}

type claimsRepo struct {
	reimbursementDomain.ReimbursementRepository
	claim   *reimbursementDomain.ReimbursementClaim
	updated bool
}

func (c *claimsRepo) GetClaimByID(context.Context, uuid.UUID) (*reimbursementDomain.ReimbursementClaim, error) {
	return c.claim, nil
}
func (c *claimsRepo) UpdateClaim(_ context.Context, cl *reimbursementDomain.ReimbursementClaim) error {
	c.updated = true
	c.claim = cl
	return nil
}

type twoEmployees struct {
	hrmdomain.HRMRepository
	emps []hrmdomain.Employee
}

func (t twoEmployees) FindEmployeeByEmail(_ context.Context, email string) (*hrmdomain.Employee, error) {
	for i := range t.emps {
		if strings.EqualFold(t.emps[i].Email, email) {
			return &t.emps[i], nil
		}
	}
	return nil, nil
}

func extrasSvc(repo *extrasRepo, claims *claimsRepo) *Service {
	hrm := twoEmployees{emps: []hrmdomain.Employee{{NIP: "E1", Name: "Ani", Email: "ani@x.com"}, {NIP: "E2", Name: "Bob", Email: "bob@x.com"}}}
	s := NewService(repo, hrm)
	if claims != nil {
		s.WithClaims(claims)
	}
	s.now = func() time.Time { return time.Date(2026, 9, 30, 10, 0, 0, 0, zone) }
	return s
}

func TestEvidenceOnlyByClaimantWhilePendingAndMarksReceipt(t *testing.T) {
	claim := &reimbursementDomain.ReimbursementClaim{RequesterEmail: "ani@x.com", Status: "pending_approval"}
	claim.ID = uuid.New()
	claims := &claimsRepo{claim: claim}
	repo := newExtras()
	s := extrasSvc(repo, claims)
	ctx := context.Background()

	if _, err := s.AddEvidence(ctx, Caller{Email: "bob@x.com"}, EvidenceInput{ClaimID: claim.ID, Title: "Struk", FileRef: "drive:1"}); err == nil {
		t.Fatal("a colleague must not add evidence to someone else's claim")
	}
	if _, err := s.AddEvidence(ctx, Caller{Email: "ANI@x.com"}, EvidenceInput{ClaimID: claim.ID, Title: "", FileRef: "drive:1"}); err == nil {
		t.Fatal("title required")
	}
	if _, err := s.AddEvidence(ctx, Caller{Email: "ANI@x.com"}, EvidenceInput{ClaimID: claim.ID, Title: "Struk taksi", FileRef: "drive:1", Amount: 85_000}); err != nil {
		t.Fatal(err)
	}
	if !claims.updated || !claim.ReceiptAttached || len(repo.evidence) != 1 {
		t.Fatalf("claim must be marked as having a receipt: %+v", claim)
	}
	if _, err := s.AddEvidence(ctx, Caller{Email: "hr@x.com", Privileged: true}, EvidenceInput{ClaimID: claim.ID, Title: "Foto", FileRef: "drive:2"}); err != nil {
		t.Fatalf("HR may add: %v", err)
	}
	claim.Status = "approved"
	if _, err := s.AddEvidence(ctx, Caller{Email: "ani@x.com"}, EvidenceInput{ClaimID: claim.ID, Title: "Late", FileRef: "x"}); err == nil {
		t.Fatal("no evidence after a decision")
	}
	if _, err := s.ListEvidence(ctx, Caller{Email: "bob@x.com"}, claim.ID); err == nil {
		t.Fatal("evidence is private to the claimant and HR")
	}
}

func TestCanteenOrderingRules(t *testing.T) {
	repo := newExtras()
	s := extrasSvc(repo, nil)
	ctx := context.Background()
	item, err := s.CreateCanteenItem(ctx, "Nasi Ayam", 25_000.499)
	if err != nil || item.Price != 25_000.5 {
		t.Fatalf("item: %+v %v", item, err)
	}
	for _, bad := range [][2]any{{"", 5.0}, {"X", 0.0}, {"X", -1.0}} {
		if _, err := s.CreateCanteenItem(ctx, bad[0].(string), bad[1].(float64)); err == nil {
			t.Errorf("should be refused: %v", bad)
		}
	}
	ani := Caller{Email: "ani@x.com"}
	o, err := s.OrderMeal(ctx, ani, "", item.ID, 2, "")
	if err != nil || o.NIP != "E1" || o.Amount != 50_001 || o.Date != "2026-09-30" || o.Status != "ordered" {
		t.Fatalf("order: %+v %v", o, err)
	}
	// Changing the menu price later leaves the order at its original price.
	newPrice := 30_000.0
	if _, err := s.UpdateCanteenItem(ctx, item.ID, &newPrice, nil); err != nil || o.UnitPrice != 25_000.5 {
		t.Fatalf("price snapshot: %+v %v", o, err)
	}
	if _, err := s.OrderMeal(ctx, ani, "E2", item.ID, 1, ""); err == nil {
		t.Fatal("cannot order on someone else's behalf")
	}
	if _, err := s.OrderMeal(ctx, ani, "", item.ID, 0, ""); err == nil {
		t.Fatal("quantity must be positive")
	}
	if _, err := s.OrderMeal(ctx, ani, "", item.ID, 21, ""); err == nil {
		t.Fatal("quantity has an upper bound")
	}
	if _, err := s.OrderMeal(ctx, ani, "", item.ID, 1, "2026-09-01"); err == nil {
		t.Fatal("past days cannot be back-filled into payroll")
	}
	off := false
	_, _ = s.UpdateCanteenItem(ctx, item.ID, nil, &off)
	if _, err := s.OrderMeal(ctx, ani, "", item.ID, 1, ""); err == nil {
		t.Fatal("an inactive item cannot be ordered")
	}
	// Cancel: owner only, only while still to come.
	if _, err := s.CancelMeal(ctx, Caller{Email: "bob@x.com"}, o.ID); err == nil {
		t.Fatal("only the owner cancels")
	}
	if got, err := s.CancelMeal(ctx, ani, o.ID); err != nil || got.Status != "cancelled" {
		t.Fatalf("cancel: %+v %v", got, err)
	}
	if _, err := s.CancelMeal(ctx, ani, o.ID); err == nil {
		t.Fatal("already cancelled")
	}
	old := &domain.CanteenOrder{Date: "2026-09-10", Status: "ordered", CreatedByEmail: "ani@x.com"}
	old.ID = uuid.New()
	repo.orders[old.ID] = old
	if _, err := s.CancelMeal(ctx, ani, old.ID); err == nil {
		t.Fatal("a past order is settled")
	}
}

func TestVisitPlanCheckInAndRoute(t *testing.T) {
	repo := newExtras()
	s := extrasSvc(repo, nil)
	ctx := context.Background()
	ani := Caller{Email: "ani@x.com"}
	base := GeoPoint{-6.2, 106.8}
	near := func(m float64) StopInput { // ~m metres north of base
		return StopInput{CustomerName: "PT", Latitude: base.Lat + m/111_320, Longitude: base.Lng}
	}
	plan, err := s.PlanVisits(ctx, ani, "", "2026-09-30", []StopInput{near(5_000), near(1_000), near(3_000)})
	if err != nil || len(plan) != 3 || plan[0].Seq != 1 || plan[2].Seq != 3 || plan[0].RadiusMeters != 100 {
		t.Fatalf("plan: %+v %v", plan, err)
	}
	more, _ := s.PlanVisits(ctx, ani, "", "2026-09-30", []StopInput{near(2_000)})
	if more[0].Seq != 4 {
		t.Fatalf("stops append to the day's plan: %+v", more)
	}
	for _, bad := range [][]StopInput{nil, {{CustomerName: "", Latitude: 1, Longitude: 1}}, {{CustomerName: "x", Latitude: 95, Longitude: 1}}, {{CustomerName: "x", Latitude: 0, Longitude: 0}}, {{CustomerName: "x", Latitude: 1, Longitude: 1, RadiusMeters: 5}}} {
		if _, err := s.PlanVisits(ctx, ani, "", "2026-09-30", bad); err == nil {
			t.Errorf("should be refused: %+v", bad)
		}
	}
	if _, err := s.PlanVisits(ctx, ani, "", "2026-09-01", []StopInput{near(10)}); err == nil {
		t.Fatal("cannot plan in the past")
	}
	if _, err := s.PlanVisits(ctx, ani, "E2", "2026-09-30", []StopInput{near(10)}); err == nil {
		t.Fatal("cannot plan for a colleague without privilege")
	}

	// Optimise from the base: nearest first (1km, 2km, 3km, 5km).
	res, err := s.OptimizeRoute(ctx, ani, "2026-09-30", base)
	if err != nil {
		t.Fatal(err)
	}
	var dists []int
	for _, st := range res.Stops {
		dists = append(dists, int(math.Round(DistanceMeters(base, GeoPoint{st.Latitude, st.Longitude})/1000)))
	}
	if dists[0] != 1 || dists[1] != 2 || dists[2] != 3 || dists[3] != 5 {
		t.Fatalf("route order (km): %v", dists)
	}
	if res.PlannedLenM <= 0 {
		t.Fatal("planned route length expected")
	}

	// Check-in: outside the radius is refused with the distance, inside works once.
	first := res.Stops[0]
	if _, err := s.CheckIn(ctx, ani, first.ID, GeoPoint{first.Latitude + 0.01, first.Longitude}); err == nil || !strings.Contains(err.Error(), "check in within 100 m") {
		t.Fatalf("far check-in: %v", err)
	}
	if _, err := s.CheckIn(ctx, Caller{Email: "bob@x.com"}, first.ID, GeoPoint{first.Latitude, first.Longitude}); err == nil {
		t.Fatal("cannot check in to someone else's stop")
	}
	got, err := s.CheckIn(ctx, ani, first.ID, GeoPoint{first.Latitude + 0.0002, first.Longitude}) // ~22 m away
	if err != nil || got.Status != "visited" || got.DistanceM == nil || *got.DistanceM > 100 || got.CheckInAt == nil {
		t.Fatalf("check-in: %+v %v", got, err)
	}
	if _, err := s.CheckIn(ctx, ani, first.ID, GeoPoint{first.Latitude, first.Longitude}); err == nil {
		t.Fatal("a visited stop cannot be checked in twice")
	}
	if _, err := s.SkipStop(ctx, ani, res.Stops[1].ID, " "); err == nil {
		t.Fatal("skipping needs a reason")
	}
	if got, err := s.SkipStop(ctx, ani, res.Stops[1].ID, "Toko tutup"); err != nil || got.Status != "skipped" {
		t.Fatalf("skip: %+v %v", got, err)
	}
	// Visited/skipped stops keep their places when the rest is re-optimised.
	res2, _ := s.OptimizeRoute(ctx, ani, "2026-09-30", GeoPoint{base.Lat + 0.05, base.Lng})
	if res2.Stops[0].Status != "visited" || res2.Stops[1].Status != "skipped" || res2.Visited != 1 {
		t.Fatalf("settled stops must not move: %+v", res2.Stops)
	}
	// Tomorrow's stop cannot be checked in today.
	tomorrow, _ := s.PlanVisits(ctx, ani, "", "2026-10-01", []StopInput{near(10)})
	if _, err := s.CheckIn(ctx, ani, tomorrow[0].ID, GeoPoint{tomorrow[0].Latitude, tomorrow[0].Longitude}); err == nil {
		t.Fatal("check-in only on the day of the visit")
	}
}
