package app

import "testing"

func TestNewRequestIDIsValidAndUnique(t *testing.T) {
	first := newRequestID()
	second := newRequestID()
	if !validRequestID(first) || !validRequestID(second) {
		t.Fatalf("generated request IDs must be valid: %q %q", first, second)
	}
	if first == second {
		t.Fatal("generated request IDs should be unique")
	}
}

func TestValidRequestIDRejectsUnsafeValues(t *testing.T) {
	invalid := []string{"", " request", "request/id", "request?id", "请求", string(make([]byte, 129))}
	for _, value := range invalid {
		if validRequestID(value) {
			t.Fatalf("expected request ID %q to be rejected", value)
		}
	}
}
