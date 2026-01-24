package casefile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadRegressionDefaults(t *testing.T) {
	root := t.TempDir()
	caseDir := filepath.Join(root, "snapcase", "regression", "case1")
	if err := os.MkdirAll(caseDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfg := map[string]any{
		"version": 1,
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(caseDir, "snapcase.json"), data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	c, err := Load(filepath.Join(caseDir, "snapcase.json"), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Kind != "regression" {
		t.Fatalf("Kind = %q, want regression", c.Kind)
	}
	if c.Scenario != filepath.Join(caseDir, "scenario.lua") {
		t.Fatalf("Scenario = %q", c.Scenario)
	}
	if c.DataHome != filepath.Join(caseDir, ".nvim-data") {
		t.Fatalf("DataHome = %q", c.DataHome)
	}
	if c.ConfigHome != filepath.Join(caseDir, ".nvim-config") {
		t.Fatalf("ConfigHome = %q", c.ConfigHome)
	}
}

func TestLoadRTPExpand(t *testing.T) {
	root := t.TempDir()
	caseDir := filepath.Join(root, "snapcase", "golden", "case1")
	if err := os.MkdirAll(caseDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfg := map[string]any{
		"version": 1,
		"rtp":     []string{"${ROOT}", "${CASE}/lua", "rel"},
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(caseDir, "snapcase.json"), data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	c, err := Load(filepath.Join(caseDir, "snapcase.json"), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{
		root,
		filepath.Join(caseDir, "lua"),
		filepath.Join(caseDir, "rel"),
	}
	if !slices.Equal(c.RTP, want) {
		t.Fatalf("RTP = %q, want %q", c.RTP, want)
	}
}

func TestFilterByKind(t *testing.T) {
	cases := []Case{
		{Name: "a", Kind: "regression"},
		{Name: "b", Kind: "golden"},
		{Name: "c", Kind: "regression"},
	}
	out := FilterByKind(cases, "regression")
	if len(out) != 2 {
		t.Fatalf("FilterByKind length = %d, want 2", len(out))
	}
	if out[0].Name != "a" || out[1].Name != "c" {
		t.Fatalf("FilterByKind order = %#v", out)
	}
}
