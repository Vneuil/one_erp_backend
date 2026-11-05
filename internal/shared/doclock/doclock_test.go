package doclock

import (
	"testing"
)

func TestTC_SYS_001_FinalDocumentLockingForSupervisor(t *testing.T) {
	// TC-SYS-001: Document is Final, user is Supervisor (isSuperAdmin = false)
	err := CheckModification("Final", false)
	if err == nil {
		t.Fatalf("expected error when supervisor tries to modify final document")
	}
	if err.Error() != "Dokumen telah final dan dikunci" {
		t.Fatalf("expected 'Dokumen telah final dan dikunci', got '%s'", err.Error())
	}

	// Also test "approved" and "completed"
	if err := CheckModification("approved", false); err == nil {
		t.Fatalf("expected error for approved document")
	}
	if err := CheckModification("completed", false); err == nil {
		t.Fatalf("expected error for completed document")
	}
}

func TestTC_SYS_002_SuperAdminOverrideFinalDocument(t *testing.T) {
	// TC-SYS-002: Document is Final, user is SuperAdmin / Owner (isSuperAdmin = true)
	err := CheckModification("Final", true)
	if err != nil {
		t.Fatalf("expected nil error (allowed) for SuperAdmin override, got %v", err)
	}

	// Also test other final states
	if err := CheckModification("approved", true); err != nil {
		t.Fatalf("expected SuperAdmin override for approved doc, got %v", err)
	}
	if err := CheckModification("posted", true); err != nil {
		t.Fatalf("expected SuperAdmin override for posted doc, got %v", err)
	}
}

func TestDraftDocumentModificationAllowed(t *testing.T) {
	// Draft document should be modifiable by non-superadmins
	if err := CheckModification("Draft", false); err != nil {
		t.Fatalf("expected draft modification allowed, got %v", err)
	}
}

