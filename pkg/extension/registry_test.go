package extension

import "testing"

type stubProvider struct{ name string }

func (s stubProvider) Name() string { return s.name }

func TestRegistryRegisterAndList(t *testing.T) {
	r := NewReasonerRegistry()
	r.Register(stubProvider{name: "a"})
	r.Register(stubProvider{name: "b"})
	if r.Len() != 2 {
		t.Fatalf("len=%d", r.Len())
	}
	if _, err := r.Get("a"); err != nil {
		t.Fatal(err)
	}
}
