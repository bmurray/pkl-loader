package pklloader

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bmurray/pkl-loader/tests/gen/directnest"
)

func TestExternalReader(t *testing.T) {
	_, f, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Dir(f)
	projectDir := filepath.Join(moduleRoot, "tests", "external")
	configFile := filepath.Join(projectDir, "basic.pkl")

	loader := PklLoader[directnest.DirectConfig](projectDir)
	cfg, err := loader(context.Background(), configFile)
	if err != nil {
		t.Fatalf("PklLoader returned error: %v", err)
	}
	if cfg.AppName != "external-reader-app" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "external-reader-app")
	}
	if cfg.Host != "external.example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "external.example.com")
	}
	if cfg.Port != 7777 {
		t.Errorf("Port = %d, want %d", cfg.Port, 7777)
	}
	// Defaults from DirectConfig.pkl
	if cfg.EnableCache != true {
		t.Errorf("EnableCache = %v, want true", cfg.EnableCache)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want %d", cfg.MaxRetries, 3)
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", cfg.Region, "us-east-1")
	}
}

func TestExternalReaderPklEval(t *testing.T) {
	_, f, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Dir(f)
	projectDir := filepath.Join(moduleRoot, "tests", "external")
	configFile := filepath.Join(projectDir, "basic.pkl")

	cmd := exec.Command("pkl", "eval", "--project-dir", projectDir, "-f", "json", configFile)
	cmd.Dir = moduleRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pkl eval failed: %v\n%s", err, out)
	}

	var cfg directnest.DirectConfig
	if err := json.Unmarshal(out, &cfg); err != nil {
		t.Fatalf("unmarshal pkl eval output: %v\n%s", err, out)
	}
	if cfg.AppName != "external-reader-app" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "external-reader-app")
	}
	if cfg.Host != "external.example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "external.example.com")
	}
	if cfg.Port != 7777 {
		t.Errorf("Port = %d, want %d", cfg.Port, 7777)
	}
	if cfg.EnableCache != true {
		t.Errorf("EnableCache = %v, want true", cfg.EnableCache)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want %d", cfg.MaxRetries, 3)
	}
}
