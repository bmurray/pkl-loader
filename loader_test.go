package pklloader

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/bmurray/pkl-loader/tests/config"
	"github.com/bmurray/pkl-loader/tests/extras"
	"github.com/bmurray/pkl-loader/tests/fixtures"
	"github.com/bmurray/pkl-loader/tests/gen"
	"github.com/bmurray/pkl-loader/tests/gen/directnest"
	_ "github.com/bmurray/pkl-loader/tests/gen/nested"
	_ "github.com/bmurray/pkl-loader/tests/gen/subconfig"
)

type testConfig struct {
	AppName  string       `json:"appName"  yaml:"appName"`
	Database testDatabase `json:"database" yaml:"database"`
	Features testFeatures `json:"features" yaml:"features"`
	Sub      testSub      `json:"sub"      yaml:"sub"`
	Nested   testNested   `json:"nested"   yaml:"nested"`
}

type testDatabase struct {
	Host string `json:"host" yaml:"host"`
	Port uint16 `json:"port" yaml:"port"`
	Name string `json:"name" yaml:"name"`
}

type testFeatures struct {
	EnableCache bool  `json:"enableCache" yaml:"enableCache"`
	MaxRetries  uint8 `json:"maxRetries"  yaml:"maxRetries"`
}

type testSub struct {
	Region  string `json:"region"  yaml:"region"`
	Enabled bool   `json:"enabled" yaml:"enabled"`
}

type testNested struct {
	Label    string `json:"label"    yaml:"label"`
	Priority uint8  `json:"priority" yaml:"priority"`
}

func testdataDir() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "tests", "testdata")
}

func expectedConfig() testConfig {
	return testConfig{
		AppName:  "test-app",
		Database: testDatabase{Host: "localhost", Port: 5432, Name: "testdb"},
		Features: testFeatures{EnableCache: true, MaxRetries: 3},
		Sub:      testSub{Region: "us-east-1", Enabled: true},
		Nested:   testNested{Label: "primary", Priority: 1},
	}
}

func TestLoadJSON(t *testing.T) {
	cfg, err := Load[testConfig](context.Background(), filepath.Join(testdataDir(), "basic.json"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !reflect.DeepEqual(*cfg, expectedConfig()) {
		t.Errorf("got %+v, want %+v", *cfg, expectedConfig())
	}
}

func TestLoadYAML(t *testing.T) {
	cfg, err := Load[testConfig](context.Background(), filepath.Join(testdataDir(), "basic.yaml"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !reflect.DeepEqual(*cfg, expectedConfig()) {
		t.Errorf("got %+v, want %+v", *cfg, expectedConfig())
	}
}

func TestLoadExtensionFallback(t *testing.T) {
	// Write a temporary JSON file so the fallback resolver finds it
	// without a competing .pkl file in the same directory.
	dir := t.TempDir()
	src := filepath.Join(testdataDir(), "basic.json")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fallback.json"), data, 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	cfg, err := Load[testConfig](context.Background(), filepath.Join(dir, "fallback"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !reflect.DeepEqual(*cfg, expectedConfig()) {
		t.Errorf("got %+v, want %+v", *cfg, expectedConfig())
	}
}

func TestLoadUnsupportedExtension(t *testing.T) {
	_, err := Load[testConfig](context.Background(), filepath.Join(testdataDir(), "basic.toml"))
	if err == nil {
		t.Fatal("expected error for unsupported extension, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported file extension") {
		t.Errorf("error %q should contain %q", err.Error(), "unsupported file extension")
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load[testConfig](context.Background(), filepath.Join(testdataDir(), "nonexistent"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "no configuration file found") {
		t.Errorf("error %q should contain %q", err.Error(), "no configuration file found")
	}
}

func TestLoadBasicEmbeddedPkl(t *testing.T) {
	pklData := `amends "@schema/Config.pkl"

appName = "test-app"

database {
  host = "localhost"
  port = 5432
  name = "testdb"
}

features {
  enableCache = true
  maxRetries = 3
}

sub {
  region = "us-east-1"
  enabled = true
}

nested {
  label = "primary"
  priority = 1
}
`
	configFS := fstest.MapFS{
		"basic.pkl": &fstest.MapFile{Data: []byte(pklData)},
	}

	cfg, err := Load[gen.AppConfig](
		context.Background(),
		"basic.pkl",
		WithSchema(config.FS),
		WithConfigFS(configFS),
	)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.AppName != "test-app" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "test-app")
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "localhost")
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d, want %d", cfg.Database.Port, 5432)
	}
	if cfg.Features.EnableCache != true {
		t.Errorf("Features.EnableCache = %v, want true", cfg.Features.EnableCache)
	}
	if cfg.Sub.Region != "us-east-1" {
		t.Errorf("Sub.Region = %q, want %q", cfg.Sub.Region, "us-east-1")
	}
	if cfg.Nested.Label != "primary" {
		t.Errorf("Nested.Label = %q, want %q", cfg.Nested.Label, "primary")
	}
	// direct field should have all defaults from DirectConfig.pkl
	if cfg.Direct.AppName != "default-app" {
		t.Errorf("Direct.AppName = %q, want %q", cfg.Direct.AppName, "default-app")
	}
	if cfg.Direct.Host != "localhost" {
		t.Errorf("Direct.Host = %q, want %q", cfg.Direct.Host, "localhost")
	}
	if cfg.Direct.Port != 8080 {
		t.Errorf("Direct.Port = %d, want %d", cfg.Direct.Port, 8080)
	}
	if cfg.Direct.EnableCache != true {
		t.Errorf("Direct.EnableCache = %v, want true", cfg.Direct.EnableCache)
	}
	if cfg.Direct.MaxRetries != 3 {
		t.Errorf("Direct.MaxRetries = %d, want %d", cfg.Direct.MaxRetries, 3)
	}
	if cfg.Direct.Region != "us-east-1" {
		t.Errorf("Direct.Region = %q, want %q", cfg.Direct.Region, "us-east-1")
	}
}

func TestLoadDirectOverrides(t *testing.T) {
	pklData := `amends "@schema/Config.pkl"

appName = "override-app"

database {
  host = "localhost"
  port = 5432
  name = "testdb"
}

features {
  enableCache = true
  maxRetries = 3
}

sub {
  region = "us-east-1"
  enabled = true
}

nested {
  label = "primary"
  priority = 1
}

direct {
  appName = "custom-direct"
  host = "10.0.0.1"
  port = 9090
  enableCache = false
  maxRetries = 7
  region = "ap-south-1"
  label = "override"
  priority = 99
}
`
	configFS := fstest.MapFS{
		"override.pkl": &fstest.MapFile{Data: []byte(pklData)},
	}

	cfg, err := Load[gen.AppConfig](
		context.Background(),
		"override.pkl",
		WithSchema(config.FS),
		WithConfigFS(configFS),
	)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Direct.AppName != "custom-direct" {
		t.Errorf("Direct.AppName = %q, want %q", cfg.Direct.AppName, "custom-direct")
	}
	if cfg.Direct.Host != "10.0.0.1" {
		t.Errorf("Direct.Host = %q, want %q", cfg.Direct.Host, "10.0.0.1")
	}
	if cfg.Direct.Port != 9090 {
		t.Errorf("Direct.Port = %d, want %d", cfg.Direct.Port, 9090)
	}
	if cfg.Direct.EnableCache != false {
		t.Errorf("Direct.EnableCache = %v, want false", cfg.Direct.EnableCache)
	}
	if cfg.Direct.MaxRetries != 7 {
		t.Errorf("Direct.MaxRetries = %d, want %d", cfg.Direct.MaxRetries, 7)
	}
	if cfg.Direct.Region != "ap-south-1" {
		t.Errorf("Direct.Region = %q, want %q", cfg.Direct.Region, "ap-south-1")
	}
	if cfg.Direct.Label != "override" {
		t.Errorf("Direct.Label = %q, want %q", cfg.Direct.Label, "override")
	}
	if cfg.Direct.Priority != 99 {
		t.Errorf("Direct.Priority = %d, want %d", cfg.Direct.Priority, 99)
	}
}

func TestLoadDirectFromImport(t *testing.T) {
	directData := `amends "@schema/directnest/DirectConfig.pkl"

appName = "imported-direct"
host = "import.example.com"
port = 4433
`
	mainData := `amends "@schema/Config.pkl"

import "direct_override.pkl" as directCfg

appName = "import-app"

database {
  host = "localhost"
  port = 5432
  name = "testdb"
}

features {
  enableCache = true
  maxRetries = 3
}

sub {
  region = "us-east-1"
  enabled = true
}

nested {
  label = "primary"
  priority = 1
}

direct = directCfg
`
	configFS := fstest.MapFS{
		"main.pkl":             &fstest.MapFile{Data: []byte(mainData)},
		"direct_override.pkl":  &fstest.MapFile{Data: []byte(directData)},
	}

	cfg, err := Load[gen.AppConfig](
		context.Background(),
		"main.pkl",
		WithSchema(config.FS),
		WithConfigFS(configFS),
	)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	// Overridden values from the imported file
	if cfg.Direct.AppName != "imported-direct" {
		t.Errorf("Direct.AppName = %q, want %q", cfg.Direct.AppName, "imported-direct")
	}
	if cfg.Direct.Host != "import.example.com" {
		t.Errorf("Direct.Host = %q, want %q", cfg.Direct.Host, "import.example.com")
	}
	if cfg.Direct.Port != 4433 {
		t.Errorf("Direct.Port = %d, want %d", cfg.Direct.Port, 4433)
	}
	// Defaults from DirectConfig.pkl should still apply
	if cfg.Direct.EnableCache != true {
		t.Errorf("Direct.EnableCache = %v, want true", cfg.Direct.EnableCache)
	}
	if cfg.Direct.MaxRetries != 3 {
		t.Errorf("Direct.MaxRetries = %d, want %d", cfg.Direct.MaxRetries, 3)
	}
	if cfg.Direct.Region != "us-east-1" {
		t.Errorf("Direct.Region = %q, want %q", cfg.Direct.Region, "us-east-1")
	}
}

func TestLoadDirectInlineImport(t *testing.T) {
	directData := `amends "@schema/directnest/DirectConfig.pkl"

appName = "imported-direct"
host = "import.example.com"
port = 4433
`
	mainData := `amends "@schema/Config.pkl"

appName = "import-app"

database {
  host = "localhost"
  port = 5432
  name = "testdb"
}

features {
  enableCache = true
  maxRetries = 3
}

sub {
  region = "us-east-1"
  enabled = true
}

nested {
  label = "primary"
  priority = 1
}

direct = import("direct_override.pkl")
`
	configFS := fstest.MapFS{
		"main.pkl":            &fstest.MapFile{Data: []byte(mainData)},
		"direct_override.pkl": &fstest.MapFile{Data: []byte(directData)},
	}

	cfg, err := Load[gen.AppConfig](
		context.Background(),
		"main.pkl",
		WithSchema(config.FS),
		WithConfigFS(configFS),
	)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Direct.AppName != "imported-direct" {
		t.Errorf("Direct.AppName = %q, want %q", cfg.Direct.AppName, "imported-direct")
	}
	if cfg.Direct.Host != "import.example.com" {
		t.Errorf("Direct.Host = %q, want %q", cfg.Direct.Host, "import.example.com")
	}
	if cfg.Direct.Port != 4433 {
		t.Errorf("Direct.Port = %d, want %d", cfg.Direct.Port, 4433)
	}
	if cfg.Direct.EnableCache != true {
		t.Errorf("Direct.EnableCache = %v, want true", cfg.Direct.EnableCache)
	}
	if cfg.Direct.MaxRetries != 3 {
		t.Errorf("Direct.MaxRetries = %d, want %d", cfg.Direct.MaxRetries, 3)
	}
	if cfg.Direct.Region != "us-east-1" {
		t.Errorf("Direct.Region = %q, want %q", cfg.Direct.Region, "us-east-1")
	}
}

func TestLoadDirectInlineImportSubdir(t *testing.T) {
	directData := `amends "@schema/directnest/DirectConfig.pkl"

appName = "imported-direct"
host = "import.example.com"
port = 4433
`
	mainData := `amends "@schema/Config.pkl"

appName = "import-app"

database {
  host = "localhost"
  port = 5432
  name = "testdb"
}

features {
  enableCache = true
  maxRetries = 3
}

sub {
  region = "us-east-1"
  enabled = true
}

nested {
  label = "primary"
  priority = 1
}

direct = import("overrides/direct_override.pkl")
`
	configFS := fstest.MapFS{
		"main.pkl":                          &fstest.MapFile{Data: []byte(mainData)},
		"overrides/direct_override.pkl":     &fstest.MapFile{Data: []byte(directData)},
	}

	cfg, err := Load[gen.AppConfig](
		context.Background(),
		"main.pkl",
		WithSchema(config.FS),
		WithConfigFS(configFS),
	)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Direct.AppName != "imported-direct" {
		t.Errorf("Direct.AppName = %q, want %q", cfg.Direct.AppName, "imported-direct")
	}
	if cfg.Direct.Host != "import.example.com" {
		t.Errorf("Direct.Host = %q, want %q", cfg.Direct.Host, "import.example.com")
	}
	if cfg.Direct.Port != 4433 {
		t.Errorf("Direct.Port = %d, want %d", cfg.Direct.Port, 4433)
	}
	if cfg.Direct.EnableCache != true {
		t.Errorf("Direct.EnableCache = %v, want true", cfg.Direct.EnableCache)
	}
	if cfg.Direct.MaxRetries != 3 {
		t.Errorf("Direct.MaxRetries = %d, want %d", cfg.Direct.MaxRetries, 3)
	}
	if cfg.Direct.Region != "us-east-1" {
		t.Errorf("Direct.Region = %q, want %q", cfg.Direct.Region, "us-east-1")
	}
}

func TestLoadMultiFileEmbeddedPkl(t *testing.T) {
	// db.pkl provides database config, imported by the main config file.
	dbData := `amends "@schema/Config.pkl"

database {
  host = "db.example.com"
  port = 3306
  name = "production"
}
`
	mainData := `amends "@schema/Config.pkl"

import "db.pkl"

appName = "multi-app"

database = db.database

features {
  enableCache = false
  maxRetries = 5
}

sub {
  region = "eu-west-1"
  enabled = false
}

nested {
  label = "secondary"
  priority = 2
}
`
	configFS := fstest.MapFS{
		"main.pkl": &fstest.MapFile{Data: []byte(mainData)},
		"db.pkl":   &fstest.MapFile{Data: []byte(dbData)},
	}

	cfg, err := Load[gen.AppConfig](
		context.Background(),
		"main.pkl",
		WithSchema(config.FS),
		WithConfigFS(configFS),
	)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.AppName != "multi-app" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "multi-app")
	}
	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "db.example.com")
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("Database.Port = %d, want %d", cfg.Database.Port, 3306)
	}
	if cfg.Database.Name != "production" {
		t.Errorf("Database.Name = %q, want %q", cfg.Database.Name, "production")
	}
	if cfg.Features.MaxRetries != 5 {
		t.Errorf("Features.MaxRetries = %d, want %d", cfg.Features.MaxRetries, 5)
	}
	if cfg.Sub.Region != "eu-west-1" {
		t.Errorf("Sub.Region = %q, want %q", cfg.Sub.Region, "eu-west-1")
	}
	if cfg.Nested.Label != "secondary" {
		t.Errorf("Nested.Label = %q, want %q", cfg.Nested.Label, "secondary")
	}
}

func TestLoadNestedImportEmbeddedPkl(t *testing.T) {
	nestedData := `import "@schema/nested/NestedConfig.pkl"

result: NestedConfig.Nested = new {
  label = "deep"
  priority = 10
}
`
	mainData := `amends "@schema/Config.pkl"

import "nested.pkl" as nestedCfg

appName = "nested-app"

database {
  host = "localhost"
  port = 5432
  name = "testdb"
}

features {
  enableCache = true
  maxRetries = 1
}

sub {
  region = "ap-south-1"
  enabled = true
}

nested = nestedCfg.result
`
	configFS := fstest.MapFS{
		"main.pkl":   &fstest.MapFile{Data: []byte(mainData)},
		"nested.pkl": &fstest.MapFile{Data: []byte(nestedData)},
	}

	cfg, err := Load[gen.AppConfig](
		context.Background(),
		"main.pkl",
		WithSchema(config.FS),
		WithConfigFS(configFS),
	)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.AppName != "nested-app" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "nested-app")
	}
	if cfg.Nested.Label != "deep" {
		t.Errorf("Nested.Label = %q, want %q", cfg.Nested.Label, "deep")
	}
	if cfg.Nested.Priority != 10 {
		t.Errorf("Nested.Priority = %d, want %d", cfg.Nested.Priority, 10)
	}
}

func TestLoadDirectNestDefaults(t *testing.T) {
	pklData := `amends "@schema/directnest/DirectConfig.pkl"
`
	configFS := fstest.MapFS{
		"defaults.pkl": &fstest.MapFile{Data: []byte(pklData)},
	}

	cfg, err := Load[directnest.DirectConfig](
		context.Background(),
		"defaults.pkl",
		WithSchema(config.FS),
		WithConfigFS(configFS),
	)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.AppName != "default-app" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "default-app")
	}
	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}
	if cfg.EnableCache != true {
		t.Errorf("EnableCache = %v, want true", cfg.EnableCache)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want %d", cfg.MaxRetries, 3)
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", cfg.Region, "us-east-1")
	}
	if cfg.Label != "primary" {
		t.Errorf("Label = %q, want %q", cfg.Label, "primary")
	}
	if cfg.Priority != 1 {
		t.Errorf("Priority = %d, want %d", cfg.Priority, 1)
	}
}

func TestLoadDirectNestOverrides(t *testing.T) {
	pklData := `amends "@schema/directnest/DirectConfig.pkl"

appName = "custom-app"
host = "db.example.com"
port = 3306
enableCache = false
maxRetries = 10
region = "eu-west-1"
label = "secondary"
priority = 5
`
	configFS := fstest.MapFS{
		"overrides.pkl": &fstest.MapFile{Data: []byte(pklData)},
	}

	cfg, err := Load[directnest.DirectConfig](
		context.Background(),
		"overrides.pkl",
		WithSchema(config.FS),
		WithConfigFS(configFS),
	)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.AppName != "custom-app" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "custom-app")
	}
	if cfg.Host != "db.example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "db.example.com")
	}
	if cfg.Port != 3306 {
		t.Errorf("Port = %d, want %d", cfg.Port, 3306)
	}
	if cfg.EnableCache != false {
		t.Errorf("EnableCache = %v, want false", cfg.EnableCache)
	}
	if cfg.MaxRetries != 10 {
		t.Errorf("MaxRetries = %d, want %d", cfg.MaxRetries, 10)
	}
	if cfg.Region != "eu-west-1" {
		t.Errorf("Region = %q, want %q", cfg.Region, "eu-west-1")
	}
	if cfg.Label != "secondary" {
		t.Errorf("Label = %q, want %q", cfg.Label, "secondary")
	}
	if cfg.Priority != 5 {
		t.Errorf("Priority = %d, want %d", cfg.Priority, 5)
	}
}

func TestLoadFixturesEmbeddedPkl(t *testing.T) {
	cfg, err := Load[gen.AppConfig](
		context.Background(),
		"main.pkl",
		WithSchema(config.FS),
		WithConfigFS(fixtures.FS),
	)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.AppName != "import-app" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "import-app")
	}
	if cfg.Direct.AppName != "imported-direct" {
		t.Errorf("Direct.AppName = %q, want %q", cfg.Direct.AppName, "imported-direct")
	}
	if cfg.Direct.Host != "import.example.com" {
		t.Errorf("Direct.Host = %q, want %q", cfg.Direct.Host, "import.example.com")
	}
	if cfg.Direct.Port != 4433 {
		t.Errorf("Direct.Port = %d, want %d", cfg.Direct.Port, 4433)
	}
	// Defaults from DirectConfig.pkl
	if cfg.Direct.EnableCache != true {
		t.Errorf("Direct.EnableCache = %v, want true", cfg.Direct.EnableCache)
	}
	if cfg.Direct.MaxRetries != 3 {
		t.Errorf("Direct.MaxRetries = %d, want %d", cfg.Direct.MaxRetries, 3)
	}
	if cfg.Direct.Region != "us-east-1" {
		t.Errorf("Direct.Region = %q, want %q", cfg.Direct.Region, "us-east-1")
	}
}

func TestLoadCustomDependencyName(t *testing.T) {
	pklData := `amends "@my-config/directnest/DirectConfig.pkl"

appName = "custom-dep-name"
host = "custom.example.com"
`
	configFS := fstest.MapFS{
		"custom.pkl": &fstest.MapFile{Data: []byte(pklData)},
	}

	cfg, err := Load[directnest.DirectConfig](
		context.Background(),
		"custom.pkl",
		WithNamedSchema("my-config", config.FS),
		WithConfigFS(configFS),
	)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.AppName != "custom-dep-name" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "custom-dep-name")
	}
	if cfg.Host != "custom.example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "custom.example.com")
	}
	// Defaults still apply
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}
	if cfg.EnableCache != true {
		t.Errorf("EnableCache = %v, want true", cfg.EnableCache)
	}
}

func TestLoadMultiDependency(t *testing.T) {
	pklData := `amends "@schema/directnest/DirectConfig.pkl"

import "@monitoring/Monitoring.pkl" as mon

appName = "multi-dep-app"
host = mon.endpoint
`
	configFS := fstest.MapFS{
		"multi.pkl": &fstest.MapFile{Data: []byte(pklData)},
	}

	cfg, err := Load[directnest.DirectConfig](
		context.Background(),
		"multi.pkl",
		WithSchema(config.FS),
		WithDependency(Dependency{Name: "monitoring", FS: extras.FS}),
		WithConfigFS(configFS),
	)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.AppName != "multi-dep-app" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "multi-dep-app")
	}
	// host should come from Monitoring.pkl default endpoint
	if cfg.Host != "https://monitoring.example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "https://monitoring.example.com")
	}
	// Defaults from DirectConfig.pkl
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}
}

