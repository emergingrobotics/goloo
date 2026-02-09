package provider

import (
	"context"
	"testing"

	"github.com/emergingrobotics/goloo/internal/config"
)

type fakeProvider struct {
	name string
}

func (f *fakeProvider) Name() string                                                          { return f.name }
func (f *fakeProvider) Create(_ context.Context, _ *config.Config, _ string) error            { return nil }
func (f *fakeProvider) Delete(_ context.Context, _ *config.Config) error                      { return nil }
func (f *fakeProvider) Status(_ context.Context, _ *config.Config) (*VMStatus, error)         { return nil, nil }
func (f *fakeProvider) List(_ context.Context) ([]VMStatus, error)                            { return nil, nil }
func (f *fakeProvider) SSH(_ context.Context, _ *config.Config) error                         { return nil }
func (f *fakeProvider) Stop(_ context.Context, _ *config.Config) error                        { return nil }
func (f *fakeProvider) Start(_ context.Context, _ *config.Config) error                       { return nil }

func TestRegisterAndGet(t *testing.T) {
	Reset()
	fake := &fakeProvider{name: "fake"}
	Register("fake", fake)

	got, err := Get("fake")
	if err != nil {
		t.Fatalf("Get(\"fake\") returned error: %v", err)
	}
	if got.Name() != "fake" {
		t.Errorf("Get(\"fake\").Name() = %q, want %q", got.Name(), "fake")
	}
}

func TestGetUnknownProvider(t *testing.T) {
	Reset()

	_, err := Get("nonexistent")
	if err == nil {
		t.Fatal("Get(\"nonexistent\") should return error")
	}
}

func TestListProviders(t *testing.T) {
	Reset()
	Register("aws", &fakeProvider{name: "aws"})
	Register("multipass", &fakeProvider{name: "multipass"})

	names := List()

	if len(names) != 2 {
		t.Fatalf("List() returned %d names, want 2", len(names))
	}
	if names[0] != "aws" {
		t.Errorf("List()[0] = %q, want %q", names[0], "aws")
	}
	if names[1] != "multipass" {
		t.Errorf("List()[1] = %q, want %q", names[1], "multipass")
	}
}

func TestListEmptyRegistry(t *testing.T) {
	Reset()

	names := List()
	if len(names) != 0 {
		t.Errorf("List() returned %d names for empty registry, want 0", len(names))
	}
}

func TestRegisterOverwrites(t *testing.T) {
	Reset()
	Register("provider", &fakeProvider{name: "original"})
	Register("provider", &fakeProvider{name: "replacement"})

	got, err := Get("provider")
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if got.Name() != "replacement" {
		t.Errorf("Get().Name() = %q, want %q after overwrite", got.Name(), "replacement")
	}
}
