package sod

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/shared/actor"
)

func TestSamePerson(t *testing.T) {
	if !SamePerson("Ani@X.com", " ani@x.com ") {
		t.Error("emails should match ignoring case and whitespace")
	}
	if SamePerson("", "") || SamePerson("a@x.com", "") || SamePerson("a@x.com", "b@x.com") {
		t.Error("empty or different emails must not match")
	}
}

func TestForbidSelfApproval(t *testing.T) {
	ctx := actor.WithEmail(context.Background(), "ani@x.com")

	if err := ForbidSelfApproval(ctx, "ANI@x.com"); err == nil {
		t.Fatal("approving your own document must be forbidden")
	}
	if err := ForbidSelfApproval(ctx, "budi@x.com"); err != nil {
		t.Fatalf("approving someone else's document is fine: %v", err)
	}
	if err := ForbidSelfApproval(ctx, ""); err != nil {
		t.Fatalf("a document with no recorded owner must not be blocked: %v", err)
	}
	if err := ForbidSelfApproval(context.Background(), "ani@x.com"); err != nil {
		t.Fatalf("background work (no actor) must not be blocked: %v", err)
	}

	t.Setenv(AllowSelfApprovalEnv, "true")
	if err := ForbidSelfApproval(ctx, "ani@x.com"); err != nil {
		t.Fatalf("the escape hatch should allow self-approval: %v", err)
	}
}
