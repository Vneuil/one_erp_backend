package application

import (
	"context"
	"sort"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// OrgNode is one person in the reporting tree.
type OrgNode struct {
	ID         uuid.UUID  `json:"id"`
	NIP        string     `json:"nip"`
	Name       string     `json:"name"`
	Role       string     `json:"role"`
	Department string     `json:"department"`
	Status     string     `json:"status"`
	ManagerID  *uuid.UUID `json:"managerId,omitempty"`
	// DirectReports counts the people who report straight to this person;
	// TeamSize counts everyone beneath them.
	DirectReports int        `json:"directReports"`
	TeamSize      int        `json:"teamSize"`
	Children      []*OrgNode `json:"children"`
}

// Organization is the whole reporting structure. Anyone whose manager is unset
// or no longer exists is a root.
type Organization struct {
	Roots []*OrgNode `json:"roots"`
	Total int        `json:"total"`
}

// BuildOrganization arranges employees into a reporting tree. A malformed cycle
// (which updates refuse to create) cannot hang it: members of a cycle that no
// root reaches are surfaced as extra roots.
func BuildOrganization(emps []domain.Employee) Organization {
	nodes := make(map[uuid.UUID]*OrgNode, len(emps))
	for _, e := range emps {
		nodes[e.ID] = &OrgNode{ID: e.ID, NIP: e.NIP, Name: e.Name, Role: e.Role, Department: e.Department, Status: e.Status, ManagerID: e.ManagerID, Children: []*OrgNode{}}
	}
	var roots []*OrgNode
	for _, e := range emps {
		n := nodes[e.ID]
		if p, ok := nodes[derefID(e.ManagerID)]; ok && e.ManagerID != nil && p != n {
			p.Children = append(p.Children, n)
		} else {
			roots = append(roots, n)
		}
	}
	seen := map[uuid.UUID]bool{}
	var walk func(n *OrgNode) int
	walk = func(n *OrgNode) int {
		seen[n.ID] = true
		sort.SliceStable(n.Children, func(i, j int) bool { return n.Children[i].Name < n.Children[j].Name })
		n.DirectReports = len(n.Children)
		size := 0
		for _, c := range n.Children {
			if !seen[c.ID] {
				size += 1 + walk(c)
			}
		}
		n.TeamSize = size
		return size
	}
	sort.SliceStable(roots, func(i, j int) bool { return roots[i].Name < roots[j].Name })
	for _, r := range roots {
		walk(r)
	}
	for _, e := range emps { // members of a cycle no root reaches
		if !seen[e.ID] {
			n := nodes[e.ID]
			roots = append(roots, n)
			walk(n)
		}
	}
	return Organization{Roots: roots, Total: len(emps)}
}

func derefID(p *uuid.UUID) uuid.UUID {
	if p == nil {
		return uuid.Nil
	}
	return *p
}

// ValidateManager checks that setting managerID as id's manager keeps the
// structure a tree: the manager exists, is not the employee, and does not
// (directly or indirectly) report to the employee.
func ValidateManager(id, managerID uuid.UUID, byID map[uuid.UUID]domain.Employee) error {
	if managerID == id {
		return apperrors.NewBadRequest("An employee cannot be their own manager")
	}
	m, ok := byID[managerID]
	if !ok {
		return apperrors.NewBadRequest("Manager not found")
	}
	cur := m
	for i := 0; i < len(byID)+1; i++ {
		if cur.ManagerID == nil {
			return nil
		}
		if *cur.ManagerID == id {
			return apperrors.NewBadRequest("That would make the reporting line circular")
		}
		next, ok := byID[*cur.ManagerID]
		if !ok {
			return nil
		}
		cur = next
	}
	return apperrors.NewBadRequest("Reporting line is too deep or circular")
}

func (uc *hrmUseCase) allEmployees(ctx context.Context) ([]domain.Employee, error) {
	list, _, err := uc.repo.ListEmployees(ctx, types.PaginationQuery{Page: 1, PerPage: 5000})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list employees")
	}
	return list, nil
}

func (uc *hrmUseCase) Organization(ctx context.Context) (*Organization, error) {
	list, err := uc.allEmployees(ctx)
	if err != nil {
		return nil, err
	}
	org := BuildOrganization(list)
	return &org, nil
}

// applyManager sets or clears the manager on emp (empty string clears it).
func (uc *hrmUseCase) applyManager(ctx context.Context, emp *domain.Employee, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		emp.ManagerID = nil
		return nil
	}
	mid, err := uuid.Parse(raw)
	if err != nil {
		return apperrors.NewBadRequest("Invalid manager ID")
	}
	list, err := uc.allEmployees(ctx)
	if err != nil {
		return err
	}
	byID := make(map[uuid.UUID]domain.Employee, len(list))
	for _, e := range list {
		byID[e.ID] = e
	}
	if err := ValidateManager(emp.ID, mid, byID); err != nil {
		return err
	}
	emp.ManagerID = &mid
	return nil
}
