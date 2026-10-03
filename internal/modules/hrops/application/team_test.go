package application

import (
	"testing"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/google/uuid"
)

func TestDescendants(t *testing.T) {
	mk := func(mgr *uuid.UUID) hrmdomain.Employee {
		e := hrmdomain.Employee{ManagerID: mgr}
		e.ID = uuid.New()
		return e
	}
	boss := mk(nil)
	mid := mk(&boss.ID)
	low := mk(&mid.ID)
	peer := mk(&boss.ID)
	other := mk(nil)
	all := []hrmdomain.Employee{boss, mid, low, peer, other}

	got := Descendants(all, mid.ID)
	if len(got) != 1 || got[low.ID].ID != low.ID {
		t.Fatalf("mid should manage only low: %v", got)
	}
	if len(Descendants(all, boss.ID)) != 3 {
		t.Fatal("boss manages mid, low and peer")
	}
	if len(Descendants(all, low.ID)) != 0 || len(Descendants(all, other.ID)) != 0 {
		t.Fatal("leaf and unrelated people manage nobody")
	}
	if _, ok := Descendants(all, mid.ID)[mid.ID]; ok {
		t.Fatal("a manager is not their own report")
	}
	// A corrupt cycle terminates.
	a, b := mk(nil), mk(nil)
	a.ManagerID, b.ManagerID = &b.ID, &a.ID
	if len(Descendants([]hrmdomain.Employee{a, b}, a.ID)) != 1 {
		t.Fatal("cycle should yield just the other member")
	}
}
