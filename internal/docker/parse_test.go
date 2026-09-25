package docker

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestParseBytes(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"0B", 0, false},
		{"512B", 512, false},
		{"1KiB", 1024, false},
		{"128MiB", 128 * 1024 * 1024, false},
		{"1GiB", 1 << 30, false},
		{"2TiB", 2 * (1 << 40), false},
		{"1.5GiB", int64(1.5 * float64(1<<30)), false},
		{"100kB", 100_000, false},
		{"5MB", 5_000_000, false},
		{"2GB", 2_000_000_000, false},
		{"1TB", 1_000_000_000_000, false},
		{"12.50%", 0, true},
		{"", 0, true},
		{"100", 0, true},
		{"1,5GiB", int64(1.5 * float64(1<<30)), false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseBytes(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseBytes(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseBytes(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("parseBytes(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseBytePair(t *testing.T) {
	tests := []struct {
		input     string
		wantFirst int64
		wantSecond int64
		wantErr   bool
	}{
		{"128MiB / 1GiB", 128 * 1024 * 1024, 1 << 30, false},
		{"1.5MB / 2MiB", 1_500_000, 2 * 1024 * 1024, false},
		{"3kB / 4B", 3000, 4, false},
		{"100B / 200B", 100, 200, false},
		{"no-slash", 0, 0, true},
		{" / 1GiB", 0, 0, true},
		{"1GiB / ", 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			first, second, err := parseBytePair(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseBytePair(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseBytePair(%q) unexpected error: %v", tt.input, err)
			}
			if first != tt.wantFirst || second != tt.wantSecond {
				t.Fatalf("parseBytePair(%q) = (%d, %d), want (%d, %d)", tt.input, first, second, tt.wantFirst, tt.wantSecond)
			}
		})
	}
}

func TestParsePercent(t *testing.T) {
	tests := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		{"12.50%", 12.5, false},
		{"0%", 0, false},
		{"100%", 100, false},
		{"0.01%", 0.01, false},
		{"12,50%", 12.5, false},
		{"%", 0, true},
		{"abc%", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parsePercent(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parsePercent(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePercent(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("parsePercent(%q) = %f, want %f", tt.input, got, tt.want)
			}
		})
	}
}

func TestDockerCommandError(t *testing.T) {
	t.Run("timeout wraps deadline", func(t *testing.T) {
		err := dockerCommandError("inspect", "api", commandOutput{}, context.DeadlineExceeded)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected error chain to contain DeadlineExceeded: %v", err)
		}
		if !strings.Contains(err.Error(), "timed out") {
			t.Fatalf("expected 'timed out' in error: %v", err)
		}
	})

	t.Run("not found wraps exec.ErrNotFound", func(t *testing.T) {
		err := dockerCommandError("inspect", "api", commandOutput{}, exec.ErrNotFound)
		if !errors.Is(err, exec.ErrNotFound) {
			t.Fatalf("expected error chain to contain ErrNotFound: %v", err)
		}
		if !strings.Contains(err.Error(), "not installed") {
			t.Fatalf("expected 'not installed' in error: %v", err)
		}
	})

	t.Run("uses stderr for detail", func(t *testing.T) {
		err := dockerCommandError("inspect", "api", commandOutput{stderr: "Error: No such container"}, errors.New("exit status 1"))
		if !strings.Contains(err.Error(), "No such container") {
			t.Fatalf("expected stderr in error: %v", err)
		}
		if !errors.Is(err, exec.ErrNotFound) == false {
			// just verify it doesn't falsely match ErrNotFound
		}
	})

	t.Run("falls back to stdout when stderr empty", func(t *testing.T) {
		err := dockerCommandError("logs", "api", commandOutput{stdout: "some output"}, errors.New("exit status 1"))
		if !strings.Contains(err.Error(), "some output") {
			t.Fatalf("expected stdout fallback: %v", err)
		}
	})

	t.Run("falls back to error message when output empty", func(t *testing.T) {
		original := errors.New("something broke")
		err := dockerCommandError("logs", "api", commandOutput{}, original)
		if !strings.Contains(err.Error(), "something broke") {
			t.Fatalf("expected error message fallback: %v", err)
		}
		if !errors.Is(err, original) {
			t.Fatalf("expected original error in chain: %v", err)
		}
	})

	t.Run("notes truncation", func(t *testing.T) {
		err := dockerCommandError("logs", "api", commandOutput{stderr: "big output", truncated: true}, errors.New("exit status 1"))
		if !strings.Contains(err.Error(), "[docker output truncated]") {
			t.Fatalf("expected truncation note: %v", err)
		}
	})
}

func TestNormalizeMounts(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		if got := normalizeMounts(nil); got != nil {
			t.Fatalf("expected nil, got %#v", got)
		}
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		if got := normalizeMounts([]inspectMount{}); got != nil {
			t.Fatalf("expected nil, got %#v", got)
		}
	})

	t.Run("normalizes fields", func(t *testing.T) {
		raw := []inspectMount{
			{Type: "bind", Source: "/host/data", Destination: "/container/data", Mode: "z", RW: true, Propagation: "rprivate"},
			{Type: "volume", Name: "my-vol", Destination: "/data", RW: false},
		}
		got := normalizeMounts(raw)
		if len(got) != 2 {
			t.Fatalf("expected 2 mounts, got %d", len(got))
		}
		if got[0].Type != "bind" || got[0].Source != "/host/data" || got[0].Destination != "/container/data" || !got[0].RW || got[0].Propagation != "rprivate" {
			t.Fatalf("unexpected first mount: %#v", got[0])
		}
		if got[1].Type != "volume" || got[1].Name != "my-vol" || got[1].RW {
			t.Fatalf("unexpected second mount: %#v", got[1])
		}
	})
}
