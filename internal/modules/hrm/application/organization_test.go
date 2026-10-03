package application

import (
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/google/uuid"
)

func emp(name string, mgr *uuid.UUID) domain.Employee {
	e := domain.Employee{Name: name, ManagerID: mgr}
	e.ID = uuid.New()
	return e
}

func TestOrganizationTreeAndCycles(t *testing.T) {
	ceo := emp("Ceo", nil)
	vp := emp("Vp", &ceo.ID)
	dev := emp("Dev", &vp.ID)
	ops := emp("Ops", &vp.ID)
	orphanMgr := uuid.New()
	orphan := emp("Orphan", &orphanMgr) // manager was deleted
	org := BuildOrganization([]domain.Employee{dev, ops, orphan, vp, ceo})
	if len(org.Roots) != 2 || org.Total != 5 {
		t.Fatalf("roots=%d total=%d", len(org.Roots), org.Total)
	}
	var top *OrgNode
	for _, r := range org.Roots {
		if r.Name == "Ceo" {
			top = r
		}
	}
	if top == nil || top.DirectReports != 1 || top.TeamSize != 3 || top.Children[0].DirectReports != 2 {
		t.Fatalf("ceo = %+v", top)
	}

	byID := map[uuid.UUID]domain.Employee{ceo.ID: ceo, vp.ID: vp, dev.ID: dev, ops.ID: ops}
	if ValidateManager(dev.ID, dev.ID, byID) == nil {
		t.Error("self manager must be refused")
	}
	if ValidateManager(ceo.ID, dev.ID, byID) == nil {
		t.Error("making the CEO report to their own subordinate must be refused")
	}
	if ValidateManager(ops.ID, dev.ID, byID) != nil {
		t.Error("a peer branch is a valid manager")
	}
	if ValidateManager(dev.ID, uuid.New(), byID) == nil {
		t.Error("unknown manager must be refused")
	}

	// A corrupt cycle in stored data still renders.
	a, b := emp("A", nil), emp("B", nil)
	a.ManagerID, b.ManagerID = &b.ID, &a.ID
	if got := BuildOrganization([]domain.Employee{a, b}); len(got.Roots) == 0 {
		t.Error("cycle members should surface as roots")
	}
}
