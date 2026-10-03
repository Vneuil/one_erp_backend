package types_test

import (
	"testing"

	"github.com/divinecoid/one-backend/internal/shared/types"
)

func TestPaginationQuery_SetDefaults(t *testing.T) {
	q := types.PaginationQuery{}
	q.SetDefaults()

	if q.Page != 1 {
		t.Errorf("expected default page to be 1, got %d", q.Page)
	}
	if q.PerPage != 10 {
		t.Errorf("expected default perPage to be 10, got %d", q.PerPage)
	}
	if q.SortDir != "desc" {
		t.Errorf("expected default sortDir to be desc, got %s", q.SortDir)
	}

	q2 := types.PaginationQuery{Page: 2, PerPage: 150}
	q2.SetDefaults()
	if q2.PerPage != 100 {
		t.Errorf("expected perPage cap at 100, got %d", q2.PerPage)
	}
	if q2.Offset() != 100 {
		t.Errorf("expected offset to be 100, got %d", q2.Offset())
	}
}

func TestNewPaginationMeta(t *testing.T) {
	meta := types.NewPaginationMeta(55, 1, 10)
	if meta.TotalPages != 6 {
		t.Errorf("expected 6 total pages, got %d", meta.TotalPages)
	}
	if meta.TotalItems != 55 {
		t.Errorf("expected 55 total items, got %d", meta.TotalItems)
	}
}
