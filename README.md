# pkl-loader

[![Go Reference](https://pkg.go.dev/badge/github.com/bmurray/pkl-loader.svg)](https://pkg.go.dev/github.com/bmurray/pkl-loader)

A Go library for loading application configuration with automatic format detection. Supports `.pkl`, `.pklbin`, `.json`, `.yaml`, and `.yml` files with a fallback resolution order when no extension is given.

The primary use case is embedding [Pkl](https://pkl-lang.org/) schema packages into Go binaries so that `.pkl` config files can be evaluated at runtime without any on-disk project structure.

## Install

```bash
go get github.com/bmurray/pkl-loader
```

## Quick start

### JSON / YAML

No schema setup required:

```go
type Config struct {
    AppName string `json:"appName" yaml:"appName"`
    Port    int    `json:"port"    yaml:"port"`
}

cfg, err := pklloader.Load[Config](ctx, "config.yaml")
```

### Pre-compiled Pkl (.pklbin)

`.pklbin` files are pre-compiled Pkl modules with all dependencies resolved at build time. They load with a plain evaluator -- no schema FS, no project structure, no options needed.

**Generate a .pklbin:**

```bash
pkl eval -f binary -o app.pklbin app.pkl
```

This resolves all `amends`, `import`, and `package://` references at build time and produces a self-contained binary.

**Load it:**

```go
cfg, err := pklloader.Load[gen.AppConfig](ctx, "app.pklbin")
```

No `WithSchema` or `WithConfigFS` options are required. The `.pklbin` format is ideal for CI/CD pipelines where you want to validate and freeze configuration at build time, then ship a single binary artifact.

You still need generated Go types (`gen.AppConfig`) to decode the result -- see the Pkl section below.

### Pkl with embedded schema

Pkl loading requires **three things**: a schema package (`.pkl` files with a `PklProject`), **generated Go types** from that schema, and config files that amend the schema.

You **must** generate Go structs from your Pkl schema before using this library. The generic type parameter `T` in `Load[T]` needs concrete Go types that match your schema's structure. Without generated types, the evaluator has nothing to decode into.

#### Step 1: Define your schema

Create a Pkl package with a `PklProject` and `PklProject.deps.json`:

```
config/
  PklProject
  PklProject.deps.json
  AppConfig.pkl
  embed.go
```

#### Step 2: Generate Go types

Run the Pkl Go code generator to produce Go structs from your schema:

```bash
pkl run package://pkg.pkl-lang.org/pkl-go/pkl.golang@0.13.2#/gen.pkl \
    -p projectDir=config \
    -p moduleDir=config \
    --output-path . \
    -- config/AppConfig.pkl
```

This produces Go files (e.g. `gen/AppConfig.pkl.go`) with structs like `gen.AppConfig` that mirror your Pkl module's properties. These are the types you pass as `T` to `Load[T]`.

You can also add this as a `go:generate` directive (see [Generating Go types](#generating-go-types) below).

#### Step 3: Embed the schema

```go
package config

import "embed"

//go:embed *.pkl PklProject PklProject.deps.json
var FS embed.FS
```

#### Step 4: Write a config file

```pkl
amends "@schema/AppConfig.pkl"

appName = "my-service"
port = 8080
```

#### Step 5: Load it

The `configFS` is the filesystem where your config `.pkl` files live. You have two options:

**Option A: Config files on disk** (e.g. mounted as a Kubernetes secret, or in a config directory)

```go
// Read config files from a directory on disk.
// The filePath is relative to the directory.
cfg, err := pklloader.Load[gen.AppConfig](ctx, "app.pkl",
    pklloader.WithSchema(config.FS),
    pklloader.WithConfigDir("/etc/myapp/config"),
)
```

If you omit both `WithConfigFS` and `WithConfigDir`, config files are read from disk relative to `filePath`:

```go
// Reads /etc/myapp/config/app.pkl from disk directly.
cfg, err := pklloader.Load[gen.AppConfig](ctx, "/etc/myapp/config/app.pkl",
    pklloader.WithSchema(config.FS),
)
```

**Option B: Config files embedded in the binary** (e.g. for testing, or shipping defaults)

```go
package fixtures

import "embed"

//go:embed all:*.pkl all:**/*.pkl
var FS embed.FS
```

```go
cfg, err := pklloader.Load[gen.AppConfig](ctx, "app.pkl",
    pklloader.WithSchema(config.FS),
    pklloader.WithConfigFS(fixtures.FS),
)
```

This is useful for tests, default configurations, or when you want the entire application to be a single binary with no external files.

Config files reference the schema via `@schema/...` imports. The `@schema` prefix is the default dependency name and can be customized.

## API reference

### Load

```go
func Load[T any](ctx context.Context, filePath string, opts ...Option) (*T, error)
```

Top-level entry point. Detects the file format by extension and dispatches to the appropriate loader. When `filePath` has no extension, it tries each format in order: `.pklbin`, `.pkl`, `.json`, `.yaml`, `.yml`.

For `.pkl` files, at least one schema dependency must be configured via options.

### EmbeddedPklLoader

```go
func EmbeddedPklLoader[T any](configFS fs.FS, opts ...Option) func(context.Context, string) (T, error)
```

Returns a reusable loader function for evaluating `.pkl` config files against embedded schema packages. The returned function reads config files from `configFS` and resolves `@name` imports against the configured dependencies.

Each dependency's `fs.FS` must contain at its root:
- `PklProject` -- the schema package's project file
- `PklProject.deps.json` -- resolved dependencies (run `pkl project resolve`)
- The schema `.pkl` files and any subdirectories

### EmbeddedPklTextLoader

```go
func EmbeddedPklTextLoader(configFS fs.FS, opts ...Option) func(context.Context, string) (string, error)
```

Like `EmbeddedPklLoader` but renders the module as text instead of decoding into a Go struct. Set the format with `WithOutputFormat` (defaults to `"pcf"`). Supported formats: `"json"`, `"jsonnet"`, `"pcf"`, `"properties"`, `"plist"`, `"textproto"`, `"xml"`, `"yaml"`.

This is useful when you need serialized output without generated Go types, or for roundtripping config through text formats.

### PklLoader

```go
func PklLoader[T any](projectDir string) func(context.Context, string) (T, error)
```

Returns a loader that resolves dependencies from an on-disk `PklProject` directory. Useful during development when the config directory is present on disk and you want schema changes reflected without recompiling.

### Dependency

```go
type Dependency struct {
    Name       string  // import prefix in config files (e.g. "schema" for @schema/...)
    FS         fs.FS   // schema files, PklProject, and PklProject.deps.json
    PackageURI string  // optional; synthetic URI generated if empty
}
```

Describes a Pkl schema package. Config files reference it via `@Name` imports.

`PackageURI` is only needed if your config files use full `package://` URIs instead of `@name` imports. In most cases, leave it empty.

## Options

### Schema dependencies

| Option | Description |
|--------|-------------|
| `WithSchema(fs.FS)` | Add a schema dependency with the default name `"schema"`. Config files use `@schema/...` imports. |
| `WithSchemaDir(path)` | Like `WithSchema` but opens the directory as an `os.DirFS`. |
| `WithNamedSchema(name, fs.FS)` | Add a schema dependency with a custom name. Config files use `@name/...` imports. |
| `WithDependency(Dependency)` | Add a dependency with full control over name, FS, and package URI. |

Multiple dependencies can be added. Each is available via its `@name` prefix in config files.

### Output format

| Option | Description |
|--------|-------------|
| `WithOutputFormat(format)` | Set the text output format for `EmbeddedPklTextLoader`. Supported: `"json"`, `"yaml"`, `"pcf"`, `"jsonnet"`, `"properties"`, `"plist"`, `"textproto"`, `"xml"`. |

### Config source

| Option | Description |
|--------|-------------|
| `WithConfigFS(fs.FS)` | Set the filesystem for reading config files. Required for embedded Pkl loading. |
| `WithConfigDir(path)` | Like `WithConfigFS` but opens the directory as an `os.DirFS`. |

If neither is set, config files are read from disk relative to `filePath`.

## Format detection

When `filePath` has no extension, `Load` tries each format in order:

1. `.pklbin` -- pre-compiled Pkl binary
2. `.pkl` -- Pkl source (requires schema dependency)
3. `.json` -- JSON
4. `.yaml` -- YAML
5. `.yml` -- YAML

The first file that exists on disk is used.

## Pkl config patterns

### Basic: amend a schema

```pkl
amends "@schema/AppConfig.pkl"

appName = "my-service"
port = 9090
```

### Import from another config file

Config files can import other config files using relative paths:

```pkl
amends "@schema/AppConfig.pkl"

import "db.pkl"

appName = "my-service"
database = db.database
```

### Inline import

```pkl
amends "@schema/AppConfig.pkl"

appName = "my-service"
database = import("overrides/database.pkl")
```

Both the main config and imported files are resolved from the same `configFS`.

### Subdirectory imports

Imported files can live in subdirectories within `configFS`:

```pkl
direct = import("overrides/direct_config.pkl")
```

The imported file can itself amend a schema:

```pkl
amends "@schema/DirectConfig.pkl"

host = "10.0.0.1"
port = 9090
```

### Multiple schema dependencies

Register multiple schema packages and reference each by name:

```go
cfg, err := pklloader.Load[MyConfig](ctx, "app.pkl",
    pklloader.WithSchema(configSchema.FS),            // @schema
    pklloader.WithNamedSchema("monitoring", monFS),    // @monitoring
    pklloader.WithConfigFS(configFS),
)
```

```pkl
amends "@schema/AppConfig.pkl"

import "@monitoring/Monitoring.pkl" as mon

appName = "my-service"
monitoringEndpoint = mon.endpoint
```

### Custom dependency name

Use `WithNamedSchema` or `WithDependency` to change the `@schema` prefix:

```go
cfg, err := pklloader.Load[MyConfig](ctx, "app.pkl",
    pklloader.WithNamedSchema("my-config", schema.FS),
    pklloader.WithConfigFS(configFS),
)
```

```pkl
amends "@my-config/AppConfig.pkl"

appName = "custom"
```

### Text rendering

Use `EmbeddedPklTextLoader` to render config as JSON, YAML, or other text formats without needing generated Go types:

```go
loader := pklloader.EmbeddedPklTextLoader(configFS,
    pklloader.WithSchema(schema.FS),
    pklloader.WithOutputFormat("json"),
)
jsonText, err := loader(ctx, "app.pkl")
```

The PCF (Pkl Config Format) output omits the `amends` header. To roundtrip a config through text, prepend the header back:

```go
// Render to PCF
loader := pklloader.EmbeddedPklTextLoader(configFS,
    pklloader.WithSchema(schema.FS),
    pklloader.WithOutputFormat("pcf"),
)
pcf, _ := loader(ctx, "app.pkl")

// Roundtrip: prepend amends header, then reload as a typed struct
roundtripped := "amends \"@schema/AppConfig.pkl\"\n\n" + pcf
rtFS := fstest.MapFS{
    "roundtripped.pkl": &fstest.MapFile{Data: []byte(roundtripped)},
}
cfg, _ := pklloader.Load[gen.AppConfig](ctx, "roundtripped.pkl",
    pklloader.WithSchema(schema.FS),
    pklloader.WithConfigFS(rtFS),
)
```

This is useful for snapshotting resolved config (with all defaults applied) or converting between formats.

## Schema package setup

A schema package is a standard Pkl project. Minimal structure:

```
config/
  PklProject
  PklProject.deps.json
  AppConfig.pkl
  embed.go
```

**PklProject:**

```pkl
amends "pkl:Project"

package {
  name = "my-config"
  baseUri = "package://pkg.pkl-lang.org/myorg/my-config"
  version = "0.0.1"
  packageZipUrl = "https://example.com/my-config@\(version).zip"
}

dependencies {
  ["pkl.golang"] {
    uri = "package://pkg.pkl-lang.org/pkl-go/pkl.golang@0.13.2"
  }
}
```

**PklProject.deps.json** (generated by `pkl project resolve`):

```json
{
  "schemaVersion": 1,
  "resolvedDependencies": {
    "package://pkg.pkl-lang.org/pkl-go/pkl.golang@0": {
      "type": "remote",
      "uri": "projectpackage://pkg.pkl-lang.org/pkl-go/pkl.golang@0.13.2",
      "checksums": {
        "sha256": "f9f6caaab341eab8dfe5677c96db3a29af6dd3c9663512bf09cb920023952a41"
      }
    }
  }
}
```

**embed.go:**

```go
package config

import "embed"

//go:embed *.pkl PklProject PklProject.deps.json
var FS embed.FS
```

### Generating Go types

Use `pkl-go`'s code generator to produce Go structs from your schema:

```bash
pkl run package://pkg.pkl-lang.org/pkl-go/pkl.golang@0.13.2#/gen.pkl \
    -p projectDir=config \
    -p moduleDir=config \
    --output-path . \
    -- config/AppConfig.pkl
```

Or as a `go:generate` directive:

```go
//go:generate sh -c "cd .. && pkl run 'package://pkg.pkl-lang.org/pkl-go/pkl.golang@0.13.2#/gen.pkl' -p projectDir=config -p moduleDir=config --output-path . -- config/AppConfig.pkl"
```

## Dependencies

- [github.com/apple/pkl-go](https://github.com/apple/pkl-go) -- Pkl language evaluator
- [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) -- YAML decoding
