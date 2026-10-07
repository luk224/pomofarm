package api

import (
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	resp, err := New().Test(httptest.NewRequest("GET", "/api/health", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
