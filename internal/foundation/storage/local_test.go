package storage

import (
	"context"
	"testing"
)

func TestLocalStorageRoundTripAndTraversal(t *testing.T) {
	s := NewLocal(t.TempDir())
	ctx := context.Background()
	if _, err := s.Upload(ctx, "hr/abc/ktp.pdf", []byte("hello"), "application/pdf"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Download(ctx, "hr/abc/ktp.pdf")
	if err != nil || string(got) != "hello" {
		t.Fatalf("got %q %v", got, err)
	}
	if err := s.Delete(ctx, "hr/abc/ktp.pdf"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Download(ctx, "hr/abc/ktp.pdf"); err == nil {
		t.Fatal("deleted file must be gone")
	}
	if err := s.Delete(ctx, "hr/abc/ktp.pdf"); err != nil {
		t.Fatal("deleting twice is not an error")
	}
	for _, bad := range []string{"../escape", "/etc/passwd", "a/../../b", ""} {
		if _, err := s.Upload(ctx, bad, []byte("x"), ""); err == nil {
			t.Errorf("key %q must be refused", bad)
		}
	}
}
