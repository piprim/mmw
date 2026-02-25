# OGL - OVYA Go Library

[![Go Version](https://img.shields.io/badge/go-1.25-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

**OGL** (OVYA Go Library) is a shared library providing common functionality for Go services within the OVYA monorepo workspace.

## Packages

### Configuration Management (`config`)
Unified configuration loading with TOML files and environment variables.

### File Operations (`file`)
File utilities including existence checks, directory creation, file locking, PID files, and ZIP compression.

### String Utilities (`string`)
String manipulation utilities for accent removal and filename normalization.

## Features

### Configuration Management (`config` package)

- **Dual-source configuration**: Combine TOML files with environment variables
- **Environment-specific overrides**: Load `default.toml` + `<environment>.toml`
- **Type-safe**: Unmarshal directly to Go structs
- **Testable**: Mock filesystem support for unit tests
- **Production-ready**: Uses battle-tested libraries (Viper + go-envconfig)

### File Operations (`file` package)

- **File existence checks**: Check if files/directories exist (filesystem and fs.FS)
- **Directory creation**: Create directories with proper permissions
- **File locking**: Exclusive file locks using syscall.Flock
- **PID file management**: Create and manage process ID files with locking
- **ZIP compression**: Create ZIP archives from files with Deflate compression

### String Utilities (`string` package)

- **Accent removal**: Remove accents from strings (café → cafe)
- **Stream processing**: Process readers with accent removal
- **Filename normalization**: Sanitize and normalize filenames safely

## Installation

Since this is a workspace library, it's referenced via the workspace configuration:

```bash
# In your service directory
go work use ../../libs/ogl
```

Then import in your code:

```go
import (
    "github.com/ovya/ogl/config"
    "github.com/ovya/ogl/file"
    "github.com/ovya/ogl/string"
)
```

## Usage

### Configuration Management

#### Basic Configuration Loading

```go
package main

import (
    "context"
    "embed"
    "log"

    "github.com/ovya/ogl/config"
)

// Define your config structure
type AppConfig struct {
    Database    *DatabaseConfig `mapstructure:"database"`
    Port        string          `mapstructure:"port"`
    Environment string          `env:"APP_ENV, required"`
}

type DatabaseConfig struct {
    Host string `mapstructure:"host"`
    Port string `mapstructure:"port"`
    Name string `mapstructure:"name"`
}

func (c AppConfig) GetAppEnv() string {
    return c.Environment
}

func main() {
    ctx := context.Background()

    // Create config context with embedded filesystem
    //go:embed configs/*.toml
    var configFS embed.FS

    // Load configuration
    cfg := &AppConfig{}
    configCtx := config.NewContext(ctx, configFS, nil)

    if err := configCtx.Fill(cfg); err != nil {
        log.Fatal(err)
    }

    // Use configuration
    log.Printf("Starting on port %s", cfg.Port)
}
```

#### Configuration Files

**Directory Structure:**
```
configs/
├── default.toml      # Base configuration
├── development.toml  # Development overrides
├── staging.toml      # Staging overrides
└── production.toml   # Production overrides
```

**Example `configs/default.toml`:**
```toml
port = "8080"

[database]
host = "localhost"
port = "5432"
name = "myapp"
```

**Example `configs/production.toml`:**
```toml
port = "443"

[database]
host = "prod-db.example.com"
```

#### Environment Variables

Environment variables take precedence over TOML values:

```bash
export APP_ENV=production
export DB_PASSWORD=secret123
```

```go
type Config struct {
    Password string `env:"DB_PASSWORD, required"`
    APIKey   string `env:"API_KEY"`
}
```

### File Operations

#### File Existence Checks

```go
import "github.com/ovya/ogl/file"

// Check if file exists
if file.Exists("/path/to/file.txt") {
    log.Println("File exists")
}

// Check if path is directory
isDir, err := file.IsDir("/path/to/directory")
if err != nil {
    log.Fatal(err)
}

// Check existence in fs.FS
//go:embed static/*
var staticFS embed.FS

if file.ExistsFS(staticFS, "static/index.html") {
    log.Println("Static file exists")
}
```

#### Directory Creation

```go
import "github.com/ovya/ogl/file"

// Create directory if it doesn't exist (with 0775 permissions)
err := file.CreateDirIfNotExists("/path/to/new/directory")
if err != nil {
    log.Fatal(err)
}

// Create parent directory for a file
// If path ends with /, it's treated as directory
// Otherwise, creates parent directory of the file
err = file.CreateTargetDirIfNotExists("/path/to/new/file.txt")
if err != nil {
    log.Fatal(err)
}
```

#### File Locking and PID Files

```go
import (
    "log"
    "os"

    "github.com/ovya/ogl/file"
)

// Create PID file with exclusive lock
lockFile, err := file.CreatePidFile("/var/run/myapp.pid", 0644)
if err != nil {
    log.Fatal("Another instance is already running")
}
defer lockFile.Remove()

// Read PID from existing PID file
pid, err := file.ReadPidFile("/var/run/myapp.pid")
if err != nil {
    log.Fatal(err)
}
log.Printf("Process PID: %d", pid)

// Manual file locking
lockFile, err := file.OpenLockFile("/tmp/my.lock", 0644)
if err != nil {
    log.Fatal(err)
}
defer lockFile.Close()

if err := lockFile.Lock(); err != nil {
    if err == file.ErrWouldBlock {
        log.Fatal("File is locked by another process")
    }
    log.Fatal(err)
}
defer lockFile.Unlock()

// Do critical work while holding lock
```

#### ZIP Compression

```go
import (
    "os"

    "github.com/ovya/ogl/file"
)

// Create ZIP archive to file
f, err := os.Create("archive.zip")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

filePaths := []string{
    "/path/to/file1.txt",
    "/path/to/file2.pdf",
    "/path/to/file3.jpg",
}

err = file.ZipFiles(f, filePaths)
if err != nil {
    log.Fatal(err)
}

// Create ZIP to memory buffer
var buf bytes.Buffer
err = file.ZipFiles(&buf, filePaths)
if err != nil {
    log.Fatal(err)
}
zipData := buf.Bytes()

// Create ZIP for HTTP response
func handleDownload(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/zip")
    w.Header().Set("Content-Disposition", "attachment; filename=archive.zip")

    err := file.ZipFiles(w, filePaths)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
```

### String Utilities

#### Accent Removal

```go
import "github.com/ovya/ogl/string"

// Remove accents from string
result, err := string.UnaccentString("café résumé")
if err != nil {
    log.Fatal(err)
}
fmt.Println(result) // Output: cafe resume

// Remove accents from reader (for streaming)
reader := strings.NewReader("Crème Brûlée")
unaccentedReader := string.UnaccentReader(reader)

var buf bytes.Buffer
io.Copy(&buf, unaccentedReader)
fmt.Println(buf.String()) // Output: Creme Brulee
```

#### Filename Normalization

```go
import "github.com/ovya/ogl/string"

// Normalize filename (removes accents, sanitizes special characters)
normalized, err := string.NormalizeFileName("Café & Thé (2024).pdf")
if err != nil {
    log.Fatal(err)
}
fmt.Println(normalized) // Output: Cafe-The-2024-.pdf

// Handles path traversal attempts
safe, _ := string.NormalizeFileName("../../../etc/passwd")
fmt.Println(safe) // Output: etc-passwd

// Preserves valid characters (alphanumeric, underscore, hyphen)
clean, _ := string.NormalizeFileName("Valid_File-Name123.txt")
fmt.Println(clean) // Output: Valid_File-Name123.txt
```

## Configuration Loading Process

The `config` package follows this loading order (later sources override earlier ones):

1. **Load `configs/default.toml`** (required)
2. **Load `configs/<APP_ENV>.toml`** (optional)
3. **Apply environment variables** (highest priority)

### Struct Tag Reference

| Tag | Purpose | Example |
|-----|---------|---------|
| `mapstructure` | TOML field mapping | `mapstructure:"database"` |
| `env` | Environment variable | `env:"APP_ENV, required"` |

**Note:** The `.golangci.yml` enforces kebab-case for `mapstructure` tags via tagliatelle linter.

## API Reference

### `config` Package

#### `config.Context`

Container for configuration loading context.

```go
func NewContext(ctx context.Context, fs fs.FS, envs map[string]string) *Context
```

**Parameters:**
- `ctx`: Standard Go context
- `fs`: Filesystem containing `configs/*.toml` files (use `embed.FS` in production)
- `envs`: Optional map for environment variables (use `nil` to read from OS)

#### `config.Context.Fill`

Fills a configuration struct from TOML files and environment variables.

```go
func (c *Context) Fill(config Config) error
```

**Parameters:**
- `config`: Pointer to struct implementing `Config` interface

**Returns:**
- `error`: Any error during configuration loading

#### `config.Config` Interface

Configuration structs must implement:

```go
type Config interface {
    GetAppEnv() string
}
```

### `file` Package

#### File Existence

```go
func Exists(path string) bool
func ExistsFS(fs fs.FS, path string) bool
func IsDir(path string) (bool, error)
```

#### Directory Creation

```go
func CreateDirIfNotExists(path string) error
func CreateTargetDirIfNotExists(path string) error
```

**Note:** Creates directories with `0775` permissions.

#### File Locking

```go
type LockFile struct { *os.File }

func NewLockFile(file *os.File) *LockFile
func OpenLockFile(name string, perm os.FileMode) (*LockFile, error)
func (file *LockFile) Lock() error
func (file *LockFile) Unlock() error
func (file *LockFile) Remove() error
```

**Error Values:**
- `file.ErrWouldBlock`: File is locked by another process (equals `syscall.EWOULDBLOCK`)

#### PID File Management

```go
func CreatePidFile(name string, perm os.FileMode) (*LockFile, error)
func SaveCurrentPID(fileName string) error
func ReadPidFile(name string) (pid int, error)
func (file *LockFile) WritePid() error
func (file *LockFile) ReadPid() (int, error)
```

#### ZIP Compression

```go
func ZipFiles(w io.Writer, filePaths []string) error
```

Creates a ZIP archive containing specified files. Only the base name of each file is used in the archive. Files are compressed using Deflate method.

### `string` Package

```go
func UnaccentString(s string) (string, error)
func UnaccentReader(r io.Reader) io.Reader
func NormalizeFileName(name string) (string, error)
```

**NormalizeFileName behavior:**
- Removes accents
- Replaces special characters with hyphens
- Preserves alphanumeric, underscore, and hyphen
- Strips path traversal attempts (`..`, leading `/`)
- Collapses multiple hyphens to single hyphen
- Preserves file extension

## Testing

### Configuration Testing

```go
import (
    "context"
    "testing"
    "testing/fstest"

    "github.com/ovya/ogl/config"
)

func TestConfig(t *testing.T) {
    ctx := context.Background()

    // Create in-memory filesystem
    mockFS := fstest.MapFS{
        "configs/default.toml": &fstest.MapFile{
            Data: []byte(`port = "8080"`),
        },
    }

    // Mock environment variables
    envs := map[string]string{
        "APP_ENV": "testing",
    }

    cfg := &Config{}
    configCtx := config.NewContext(ctx, mockFS, envs)
    err := configCtx.Fill(cfg)

    if err != nil {
        t.Fatal(err)
    }
}
```

### Running Tests

```bash
# Run all tests
cd /workspace/poc/libs/ogl
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./config
go test ./file
go test ./string

# Run with verbose output
go test -v ./...
```

### Using mise

```bash
# Run tests
mise run test

# Run with coverage
mise run test:coverage
```

## Dependencies

### Core Dependencies

- **[spf13/viper](https://github.com/spf13/viper)**: TOML file parsing and merging (config)
- **[sethvargo/go-envconfig](https://github.com/sethvargo/go-envconfig)**: Environment variable processing (config)
- **[golang.org/x/text](https://pkg.go.dev/golang.org/x/text)**: Unicode normalization and transformation (string)

### Testing

- **[stretchr/testify](https://github.com/stretchr/testify)**: Testing utilities
- **Standard library**: `testing/fstest` for filesystem mocking

## Architecture

### Design Principles

1. **Separation of Concerns**: Each package has a single, well-defined responsibility
2. **Testability**: Mockable filesystem for config, comprehensive test coverage
3. **Type Safety**: Compile-time checking via struct tags and interfaces
4. **Fail Fast**: Required fields and validation cause immediate errors
5. **Convention over Configuration**: Standard patterns and defaults
6. **Security**: Path traversal protection, URL encoding, file locking
7. **Performance**: Stream processing for large data, efficient transformations

### Package Organization

```
ogl/
├── config/           # Configuration management
│   ├── config.go     # Core configuration logic
│   ├── config_test.go
│   └── doc.go
├── file/             # File operations
│   ├── file.go       # Basic file operations
│   ├── file_test.go
│   ├── lock.go       # File locking and PID files
│   ├── lock_test.go
│   ├── zip.go        # ZIP compression
│   └── zip_test.go
├── string/           # String utilities
│   ├── string.go     # Accent removal and normalization
│   └── string_test.go
├── go.mod            # Module definition
├── mise.toml         # Task runner configuration
└── README.md         # This file
```

## Development

### Linting

```bash
golangci-lint run ./...
```

### Code Style

- Follow [Uber's Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- Run `golangci-lint` before committing
- Write tests for all new functionality
- Use conventional commit messages

## Contributing

This library is part of the OVYA monorepo workspace. See the main repository's contribution guidelines.

### Adding New Packages

1. Create package directory under `ogl/`
2. Add comprehensive tests
3. Update this README with usage examples
4. Update `go.mod` if adding new dependencies

### Modifying Existing Packages

1. Maintain backward compatibility when possible
2. Update tests to cover new functionality
3. Update documentation and examples
4. Consider impact on all services using the library

## License

MIT License - see [LICENSE](LICENSE) file for details.

Copyright (c) 2026 OVYA

## Support

For issues and questions:
- Open an issue in the main repository
- Contact the OVYA development team

---

**Note:** This library is designed for use within the OVYA workspace monorepo using Go workspaces. External usage may require adjustments to import paths and workspace configuration.
