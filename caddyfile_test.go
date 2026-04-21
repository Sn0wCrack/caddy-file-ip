package caddy_file_ip

import (
	"context"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

func TestUnmarshalDefault(t *testing.T) {
	testUnmarshalDefault(t, `file`)
	testUnmarshalDefault(t, `file { }`)
}

func testUnmarshalDefault(t *testing.T, input string) {
	d := caddyfile.NewTestDispenser(input)

	r := FileIPSource{}
	err := r.UnmarshalCaddyfile(d)
	if err != nil {
		t.Errorf("unmarshal error for %q: %v", input, err)
	}

	if len(r.Files) != 0 {
		t.Errorf("expected no files, got %v", r.Files)
	}
	if r.Watch != false {
		t.Errorf("expected watch=false, got %v", r.Watch)
	}
	if r.Interval != 0 {
		t.Errorf("expected interval=0, got %v", r.Interval)
	}
}

func TestUnmarshal(t *testing.T) {
	input := `
	file /path/to/ips.txt {
		watch
		interval 1.5h
	}`

	d := caddyfile.NewTestDispenser(input)

	r := FileIPSource{}
	err := r.UnmarshalCaddyfile(d)
	if err != nil {
		t.Errorf("unmarshal error: %v", err)
	}

	if len(r.Files) != 1 || r.Files[0] != "/path/to/ips.txt" {
		t.Errorf("incorrect files: expected [/path/to/ips.txt], got %v", r.Files)
	}

	if !r.Watch {
		t.Errorf("expected watch=true, got %v", r.Watch)
	}

	expectedInterval := caddy.Duration(90 * time.Minute)
	if expectedInterval != r.Interval {
		t.Errorf("incorrect interval: expected %v, got %v", expectedInterval, r.Interval)
	}
}

func TestUnmarshalMultipleFiles(t *testing.T) {
	input := `
	file {
		file /path/to/ips1.txt
		file /path/to/ips2.txt
		watch
		interval 30m
	}`

	d := caddyfile.NewTestDispenser(input)

	r := FileIPSource{}
	err := r.UnmarshalCaddyfile(d)
	if err != nil {
		t.Errorf("unmarshal error: %v", err)
	}

	if len(r.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(r.Files))
	}

	if r.Files[0] != "/path/to/ips1.txt" || r.Files[1] != "/path/to/ips2.txt" {
		t.Errorf("incorrect files: got %v", r.Files)
	}
}

func TestUnmarshalNested(t *testing.T) {
	input := `{
				file /path/to/ips.txt {
					watch
					interval 1.5h
				}
				other_module 10h
			}`

	d := caddyfile.NewTestDispenser(input)

	d.Next()
	d.NextBlock(d.Nesting())

	r := FileIPSource{}
	err := r.UnmarshalCaddyfile(d)
	if err != nil {
		t.Errorf("unmarshal error: %v", err)
	}

	if len(r.Files) != 1 || r.Files[0] != "/path/to/ips.txt" {
		t.Errorf("incorrect files: got %v", r.Files)
	}

	expectedInterval := caddy.Duration(90 * time.Minute)
	if expectedInterval != r.Interval {
		t.Errorf("incorrect interval: expected %v, got %v", expectedInterval, r.Interval)
	}

	d.Next()
	if d.Val() != "other_module" {
		t.Errorf("cursor at unexpected position, expected 'other_module', got %v", d.Val())
	}
}

func TestProvision(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "ips.txt")
	err := os.WriteFile(tmpFile, []byte("10.0.0.0/8\n192.168.0.0/16\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	input := `file ` + tmpFile

	d := caddyfile.NewTestDispenser(input)

	r := FileIPSource{}
	err = r.UnmarshalCaddyfile(d)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
	defer cancel()

	err = r.Provision(ctx)
	if err != nil {
		t.Errorf("provision error: %v", err)
	}

	ranges := r.GetIPRanges(&http.Request{})
	if len(ranges) != 2 {
		t.Errorf("expected 2 ranges, got %d", len(ranges))
	}
}

func TestProvisionInvalidCIDR(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "ips.txt")
	err := os.WriteFile(tmpFile, []byte("10.0.0.0/8\ninvalid-cidr\n192.168.0.0/16\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	input := `file ` + tmpFile

	d := caddyfile.NewTestDispenser(input)

	r := FileIPSource{}
	err = r.UnmarshalCaddyfile(d)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
	defer cancel()

	err = r.Provision(ctx)
	if err != nil {
		t.Errorf("provision error: %v", err)
	}

	ranges := r.GetIPRanges(&http.Request{})
	if len(ranges) != 2 {
		t.Errorf("expected 2 ranges (skipping invalid), got %d", len(ranges))
	}
}

func TestProvisionCommentsAndEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "ips.txt")
	err := os.WriteFile(tmpFile, []byte("# This is a comment\n\n10.0.0.0/8\n\n# Another comment\n192.168.0.0/16\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	input := `file ` + tmpFile

	d := caddyfile.NewTestDispenser(input)

	r := FileIPSource{}
	err = r.UnmarshalCaddyfile(d)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
	defer cancel()

	err = r.Provision(ctx)
	if err != nil {
		t.Errorf("provision error: %v", err)
	}

	ranges := r.GetIPRanges(&http.Request{})
	if len(ranges) != 2 {
		t.Errorf("expected 2 ranges (skipping comments/empty), got %d", len(ranges))
	}
}

func TestProvisionMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	tmpFile1 := filepath.Join(tmpDir, "ips1.txt")
	err := os.WriteFile(tmpFile1, []byte("10.0.0.0/8\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file 1: %v", err)
	}

	tmpFile2 := filepath.Join(tmpDir, "ips2.txt")
	err = os.WriteFile(tmpFile2, []byte("192.168.0.0/16\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file 2: %v", err)
	}

	input := `file {
		file ` + tmpFile1 + `
		file ` + tmpFile2 + `
	}`

	d := caddyfile.NewTestDispenser(input)

	r := FileIPSource{}
	err = r.UnmarshalCaddyfile(d)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
	defer cancel()

	err = r.Provision(ctx)
	if err != nil {
		t.Errorf("provision error: %v", err)
	}

	ranges := r.GetIPRanges(&http.Request{})
	if len(ranges) != 2 {
		t.Errorf("expected 2 ranges, got %d", len(ranges))
	}
}

func TestProvisionFileNotFound(t *testing.T) {
	input := `file /nonexistent/file.txt`

	d := caddyfile.NewTestDispenser(input)

	r := FileIPSource{}
	err := r.UnmarshalCaddyfile(d)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
	defer cancel()

	err = r.Provision(ctx)
	if err != nil {
		t.Errorf("provision should not fail for missing file, got: %v", err)
	}

	ranges := r.GetIPRanges(&http.Request{})
	if len(ranges) != 0 {
		t.Errorf("expected 0 ranges for missing file, got %d", len(ranges))
	}
}

func TestGetIPRanges(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "ips.txt")
	err := os.WriteFile(tmpFile, []byte("10.0.0.0/8\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	input := `file ` + tmpFile

	d := caddyfile.NewTestDispenser(input)

	r := FileIPSource{}
	err = r.UnmarshalCaddyfile(d)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
	defer cancel()

	err = r.Provision(ctx)
	if err != nil {
		t.Fatalf("provision error: %v", err)
	}

	ranges := r.GetIPRanges(nil)
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}

	expectedPrefix := netip.MustParsePrefix("10.0.0.0/8")
	if ranges[0] != expectedPrefix {
		t.Errorf("expected %v, got %v", expectedPrefix, ranges[0])
	}
}

func TestIPv6(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "ips.txt")
	err := os.WriteFile(tmpFile, []byte("2001:db8::/32\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	input := `file ` + tmpFile

	d := caddyfile.NewTestDispenser(input)

	r := FileIPSource{}
	err = r.UnmarshalCaddyfile(d)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
	defer cancel()

	err = r.Provision(ctx)
	if err != nil {
		t.Fatalf("provision error: %v", err)
	}

	ranges := r.GetIPRanges(&http.Request{})
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}

	expectedPrefix := netip.MustParsePrefix("2001:db8::/32")
	if ranges[0] != expectedPrefix {
		t.Errorf("expected %v, got %v", expectedPrefix, ranges[0])
	}
}
