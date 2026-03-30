// Package pklloader loads application configuration from a file detected by
// extension. Resolution order when no extension is present:
//
//	.pklbin → .pkl → .json → .yaml → .yml
package pklloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/apple/pkl-go/pkl"
	"gopkg.in/yaml.v3"
)

// Option configures the behaviour of Load and EmbeddedPklLoader.
type Option func(*options)

// Dependency describes a Pkl schema package to make available to config files.
// Config files reference it via @Name imports (e.g. @schema/Config.pkl).
type Dependency struct {
	// Name is the import prefix used in config files (e.g. "schema" for @schema/...).
	Name string
	// FS contains the schema .pkl files, PklProject, and PklProject.deps.json.
	FS fs.FS
	// PackageURI is the full Pkl package URI (e.g. "package://example.com/config@1.0.0").
	// If empty, a synthetic URI is generated from the Name.
	PackageURI string
}

type options struct {
	configFS     fs.FS
	deps         []Dependency
	outputFormat string // "json", "yaml", "pcf", etc. for text rendering
}

// WithSchema adds a schema dependency using the default name "schema".
// Shorthand for WithDependency(Dependency{Name: "schema", FS: fsys}).
func WithSchema(fsys fs.FS) Option {
	return func(o *options) {
		o.deps = append(o.deps, Dependency{Name: "schema", FS: fsys})
	}
}

// WithSchemaDir adds a schema dependency from a directory on disk
// using the default name "schema".
func WithSchemaDir(path string) Option {
	return func(o *options) {
		o.deps = append(o.deps, Dependency{Name: "schema", FS: os.DirFS(path)})
	}
}

// WithNamedSchema adds a schema dependency with a custom import name.
// Config files reference it via @name imports (e.g. @my-config/Config.pkl).
func WithNamedSchema(name string, fsys fs.FS) Option {
	return func(o *options) {
		o.deps = append(o.deps, Dependency{Name: name, FS: fsys})
	}
}

// WithDependency adds a named schema dependency. Config files reference
// it via @Name imports. Multiple dependencies can be added.
func WithDependency(dep Dependency) Option {
	return func(o *options) { o.deps = append(o.deps, dep) }
}

// WithOutputFormat sets the output format for text rendering.
// Supported values: "json", "jsonnet", "pcf", "properties", "plist",
// "textproto", "xml", "yaml". Used by LoadText and EmbeddedPklTextLoader.
func WithOutputFormat(format string) Option {
	return func(o *options) { o.outputFormat = format }
}

// WithConfigFS sets an fs.FS containing the user config files.
// Required for loading .pkl files when using EmbeddedPklLoader through Load.
// If not set, config files are read from disk relative to filePath.
func WithConfigFS(fsys fs.FS) Option {
	return func(o *options) { o.configFS = fsys }
}

// WithConfigDir sets a directory on disk as the config file source.
func WithConfigDir(path string) Option {
	return func(o *options) { o.configFS = os.DirFS(path) }
}

// Load reads the configuration from filePath and decodes it into *T.
// .pklbin files are evaluated with a plain evaluator; .pkl files use the
// schema FS provided via WithSchema or WithSchemaDir; .json/.yaml are decoded
// directly. If the path has no recognised extension, extensions are tried in
// preference order: .pklbin → .pkl → .json → .yaml → .yml.
func Load[T any](ctx context.Context, filePath string, opts ...Option) (*T, error) {
	var o options
	for _, fn := range opts {
		fn(&o)
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	if ext != "" {
		return loadByExt[T](ctx, filePath, ext, &o, opts)
	}

	for _, e := range []string{".pklbin", ".pkl", ".json", ".yaml", ".yml"} {
		candidate := filePath + e
		if _, err := os.Stat(candidate); err == nil {
			return loadByExt[T](ctx, candidate, e, &o, opts)
		}
	}
	return nil, fmt.Errorf("config: no configuration file found for path %q", filePath)
}

func loadByExt[T any](ctx context.Context, filePath, ext string, o *options, opts []Option) (*T, error) {
	switch ext {
	case ".pklbin":
		cfg, err := loadPklBin[T](ctx, filePath)
		if err != nil {
			return nil, err
		}
		return &cfg, nil
	case ".pkl":
		if len(o.deps) == 0 {
			return nil, fmt.Errorf("config: loading .pkl files requires WithSchema, WithSchemaDir, or WithDependency option")
		}
		configFS := o.configFS
		if configFS == nil {
			configFS = os.DirFS(filepath.Dir(filePath))
			filePath = filepath.Base(filePath)
		}
		loader := EmbeddedPklLoader[T](configFS, opts...)
		cfg, err := loader(ctx, filePath)
		if err != nil {
			return nil, err
		}
		return &cfg, nil
	case ".json":
		return loadJSON[T](filePath)
	case ".yaml", ".yml":
		return loadYAML[T](filePath)
	default:
		return nil, fmt.Errorf("config: unsupported file extension %q", ext)
	}
}

func evaluate[T any](ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (T, error) {
	var ret T
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}

// PklLoader returns a loader that resolves package dependencies using the
// PklProject found in projectDir. Useful in dev when the config/ directory is
// present on disk and you want schema changes to be reflected without recompiling.
func PklLoader[T any](projectDir string) func(context.Context, string) (T, error) {
	load := evaluate[T]
	
	return func(ctx context.Context, filePath string) (T, error) {
		var zero T

		abs, err := filepath.Abs(projectDir)
		if err != nil {
			return zero, fmt.Errorf("config: resolve project dir: %w", err)
		}
		projectURL := &url.URL{Scheme: "file", Path: abs + "/"}
		evaluator, err := pkl.NewProjectEvaluator(ctx, projectURL, pkl.PreconfiguredOptions)
		if err != nil {
			return zero, fmt.Errorf("config: create pkl evaluator: %w", err)
		}
		defer evaluator.Close()
		return load(ctx, evaluator, pkl.FileSource(filePath))
	}
}

// EmbeddedPklLoader returns a loader that serves schema files from one or more
// embedded dependencies and reads config data files from configFS, so the
// binary can evaluate .pkl config files without any directory present on disk.
//
// Each dependency FS must contain at its root:
//   - PklProject — the schema package's project file
//   - PklProject.deps.json — resolved dependencies
//   - The schema .pkl files (and any subdirectories)
//
// configFS contains the user config files (e.g. mounted secrets, embedded
// test fixtures). The file at filePath is read from configFS and fed to the
// evaluator; relative imports from that file also resolve through configFS.
//
// Dependencies are configured via options (WithSchema, WithDependency, etc.)
// and referenced in config files via @name imports.
func EmbeddedPklLoader[T any](configFS fs.FS, opts ...Option) func(context.Context, string) (T, error) {
	var o options
	for _, fn := range opts {
		fn(&o)
	}

	// Fill in defaults for each dependency.
	for i := range o.deps {
		if o.deps[i].PackageURI == "" {
			o.deps[i].PackageURI = "package://localhost/" + o.deps[i].Name + "@0.0.0"
		}
	}

	load := evaluate[T]
	return func(ctx context.Context, filePath string) (T, error) {
		var zero T

		if len(o.deps) == 0 {
			return zero, fmt.Errorf("config: no schema dependencies configured")
		}

		data, err := fs.ReadFile(configFS, filePath)
		if err != nil {
			return zero, fmt.Errorf("config: read %s: %w", filePath, err)
		}

		entryURI := &url.URL{
			Scheme: "embed",
			Path:   filepath.Join("/", "config", filepath.Base(filepath.ToSlash(filePath))),
		}

		// Build the root deps.json by merging all dependencies' remote deps
		// and adding each as a local dep entry.
		rootDepsJSON, err := buildRootDepsJSONMulti(o.deps)
		if err != nil {
			return zero, fmt.Errorf("config: %w", err)
		}

		vfs := overlayFS{
			{prefix: "config", inner: staticFS{name: "PklProject.deps.json", content: rootDepsJSON}},
			{prefix: "config", inner: configFS},
		}

		rawDeps := make(map[string]any, len(o.deps))
		for _, dep := range o.deps {
			vfs = append(vfs, prefixFS{prefix: dep.Name, inner: dep.FS})

			baseUri, version := splitPackageURI(dep.PackageURI)
			rawDeps[dep.Name] = &pkl.Project{
				ProjectFileUri: "embed:///" + dep.Name + "/PklProject",
				Package: &pkl.ProjectPackage{
					Name:    dep.Name,
					BaseUri: baseUri,
					Version: version,
					Uri:     dep.PackageURI,
				},
			}
		}

		rootProject := &pkl.Project{
			ProjectFileUri:  "embed:///config/PklProject",
			RawDependencies: rawDeps,
		}

		evaluator, err := pkl.NewEvaluator(ctx,
			pkl.PreconfiguredOptions,
			pkl.WithFs(vfs, "embed"),
			pkl.WithProject(rootProject),
		)
		if err != nil {
			return zero, fmt.Errorf("config: create embedded pkl evaluator: %w", err)
		}
		defer evaluator.Close()

		return load(ctx, evaluator, &pkl.ModuleSource{Uri: entryURI, Contents: string(data)})
	}
}

// EmbeddedPklTextLoader is like EmbeddedPklLoader but renders the module as
// text using the specified output format instead of decoding into a Go struct.
// Use WithOutputFormat to set the format (defaults to "pcf" if not set).
//
// This is useful for rendering config files to JSON, YAML, or other text
// formats without needing generated Go types.
func EmbeddedPklTextLoader(configFS fs.FS, opts ...Option) func(context.Context, string) (string, error) {
	var o options
	for _, fn := range opts {
		fn(&o)
	}

	outputFormat := o.outputFormat
	if outputFormat == "" {
		outputFormat = "pcf"
	}

	for i := range o.deps {
		if o.deps[i].PackageURI == "" {
			o.deps[i].PackageURI = "package://localhost/" + o.deps[i].Name + "@0.0.0"
		}
	}

	return func(ctx context.Context, filePath string) (string, error) {
		if len(o.deps) == 0 {
			return "", fmt.Errorf("config: no schema dependencies configured")
		}

		data, err := fs.ReadFile(configFS, filePath)
		if err != nil {
			return "", fmt.Errorf("config: read %s: %w", filePath, err)
		}

		entryURI := &url.URL{
			Scheme: "embed",
			Path:   filepath.Join("/", "config", filepath.Base(filepath.ToSlash(filePath))),
		}

		rootDepsJSON, err := buildRootDepsJSONMulti(o.deps)
		if err != nil {
			return "", fmt.Errorf("config: %w", err)
		}

		vfs := overlayFS{
			{prefix: "config", inner: staticFS{name: "PklProject.deps.json", content: rootDepsJSON}},
			{prefix: "config", inner: configFS},
		}

		rawDeps := make(map[string]any, len(o.deps))
		for _, dep := range o.deps {
			vfs = append(vfs, prefixFS{prefix: dep.Name, inner: dep.FS})

			baseUri, version := splitPackageURI(dep.PackageURI)
			rawDeps[dep.Name] = &pkl.Project{
				ProjectFileUri: "embed:///" + dep.Name + "/PklProject",
				Package: &pkl.ProjectPackage{
					Name:    dep.Name,
					BaseUri: baseUri,
					Version: version,
					Uri:     dep.PackageURI,
				},
			}
		}

		rootProject := &pkl.Project{
			ProjectFileUri:  "embed:///config/PklProject",
			RawDependencies: rawDeps,
		}

		evaluator, err := pkl.NewEvaluator(ctx,
			pkl.PreconfiguredOptions,
			pkl.WithFs(vfs, "embed"),
			pkl.WithProject(rootProject),
			func(opts *pkl.EvaluatorOptions) {
				opts.OutputFormat = outputFormat
			},
		)
		if err != nil {
			return "", fmt.Errorf("config: create embedded pkl evaluator: %w", err)
		}
		defer evaluator.Close()

		return evaluator.EvaluateOutputText(ctx, &pkl.ModuleSource{Uri: entryURI, Contents: string(data)})
	}
}

// loadPklBin evaluates a pre-compiled .pklbin file using a plain evaluator.
// No project dependencies or embedded schema FS are needed because pklbin
// files are fully resolved at build time.
func loadPklBin[T any](ctx context.Context, filePath string) (T, error) {
	var zero T
	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		return zero, fmt.Errorf("config: create pkl evaluator for pklbin: %w", err)
	}
	defer evaluator.Close()

	var cfg T
	if err := evaluator.EvaluateModule(ctx, pkl.FileSource(filePath), &cfg); err != nil {
		return zero, fmt.Errorf("config: evaluate pklbin %s: %w", filePath, err)
	}
	return cfg, nil
}

func loadJSON[T any](path string) (*T, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("config: open %s: %w", path, err)
	}
	defer f.Close()

	var cfg T
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("config: decode JSON %s: %w", path, err)
	}
	return &cfg, nil
}

func loadYAML[T any](path string) (*T, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("config: open %s: %w", path, err)
	}
	defer f.Close()

	var cfg T
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("config: decode YAML %s: %w", path, err)
	}
	return &cfg, nil
}

// buildRootDepsJSONMulti produces a root PklProject.deps.json that merges all
// dependencies' remote deps and adds each dependency as a local dep entry.
func buildRootDepsJSONMulti(deps []Dependency) ([]byte, error) {
	resolved := make(map[string]json.RawMessage)

	for _, dep := range deps {
		data, err := fs.ReadFile(dep.FS, "PklProject.deps.json")
		if err != nil {
			return nil, fmt.Errorf("read PklProject.deps.json from %s: %w", dep.Name, err)
		}

		var depFile struct {
			ResolvedDependencies map[string]json.RawMessage `json:"resolvedDependencies"`
		}
		if err := json.Unmarshal(data, &depFile); err != nil {
			return nil, fmt.Errorf("decode PklProject.deps.json from %s: %w", dep.Name, err)
		}

		// Merge remote deps.
		for key, val := range depFile.ResolvedDependencies {
			resolved[key] = val
		}

		// Add local dep entry for this dependency.
		depKey := majorVersionURI(dep.PackageURI)
		localDep, err := json.Marshal(struct {
			Type string `json:"type"`
			URI  string `json:"uri"`
			Path string `json:"path"`
		}{
			Type: "local",
			URI:  strings.Replace(dep.PackageURI, "package://", "projectpackage://", 1),
			Path: "../" + dep.Name,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal local dep %s: %w", dep.Name, err)
		}
		resolved[depKey] = localDep
	}

	return json.Marshal(struct {
		SchemaVersion        int                            `json:"schemaVersion"`
		ResolvedDependencies map[string]json.RawMessage     `json:"resolvedDependencies"`
	}{
		SchemaVersion:        1,
		ResolvedDependencies: resolved,
	})
}

// splitPackageURI splits "package://example.com/foo@1.2.3" into
// baseUri "package://example.com/foo" and version "1.2.3".
func splitPackageURI(uri string) (baseUri, version string) {
	atIdx := strings.LastIndex(uri, "@")
	if atIdx == -1 {
		return uri, ""
	}
	return uri[:atIdx], uri[atIdx+1:]
}

// majorVersionURI converts a full package URI like
// "package://example.com/foo@1.2.3" to its major-version form
// "package://example.com/foo@1".
func majorVersionURI(uri string) string {
	atIdx := strings.LastIndex(uri, "@")
	if atIdx == -1 {
		return uri
	}
	version := uri[atIdx+1:]
	major, _, _ := strings.Cut(version, ".")
	return uri[:atIdx+1] + major
}
