package attachment

import (
	"bytes"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	f, err := Check(`C:\Users\x\ktp.PDF`, []byte("%PDF"))
	if err != nil || f.Name != "ktp.PDF" || f.ContentType != "application/pdf" || f.Size != 4 {
		t.Fatalf("%+v %v", f, err)
	}
	for name, c := range map[string][]byte{"run.exe": []byte("x"), "a.pdf": nil, "": []byte("x"), "big.pdf": bytes.Repeat([]byte("x"), MaxBytes+1)} {
		if _, err := Check(name, c); err == nil {
			t.Errorf("%q should be refused", name)
		}
	}
	if k := Key("hr", "my file (1).pdf"); !strings.HasPrefix(k, "hr/") || strings.ContainsAny(k[3:], " ()") && strings.Count(k, "/") != 2 {
		t.Errorf("key = %q", k)
	}
}
