package pure_test

import (
	"strings"
	"testing"

	"github.com/mohsin123321/notification-system/internal/domain/template/pure"
)

func TestRender_PasswordReset_Email(t *testing.T) {
	payload := map[string]any{"reset_link": "https://example.com/reset/abc123"}

	out, err := pure.Render("password_reset", "email", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "https://example.com/reset/abc123") {
		t.Errorf("expected reset link in output, got: %s", out)
	}
}

func TestRender_UnknownTemplate(t *testing.T) {
	_, err := pure.Render("unknown_type", "email", nil)
	if err == nil {
		t.Error("expected error for unknown template")
	}
}
