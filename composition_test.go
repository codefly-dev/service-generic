package main

import "testing"

// Generic has no Settings struct — it's the null agent, just a Service.
// This test verifies the minimum contract: Service comes back non-nil
// with a live Base (for Wool/Logger promotion). Everything else in the
// agent stems from this.
func TestNewService_EmbedsBase(t *testing.T) {
	svc := NewService()
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
	if svc.Base == nil {
		t.Fatal("Service.Base is nil — services.Base embedding broken")
	}
}
