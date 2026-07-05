# Storage Provider Overview

The `omnistorage/` package implements the [omnistorage-core](https://github.com/plexusone/omnistorage-core) `Backend` interface, allowing you to use GitHub repositories as a storage backend.

## Features

- 📄 Read and write files to any branch
- ⚡ Batch multiple file operations into a single atomic commit
- 📂 List files in directories with prefix filtering
- ℹ️ Get file metadata (size, SHA1 hash)
- 🗑️ Delete files from the repository
- ⚙️ Configurable commit messages and author
- 🏢 GitHub Enterprise support
- 🔗 Automatic registration with OmniStorage registry

## Basic Usage

```go
import "github.com/plexusone/omni-github/omnistorage"

backend, err := omnistorage.New(omnistorage.Config{
    Owner:  "myorg",
    Repo:   "myrepo",
    Branch: "main",
    Token:  os.Getenv("GITHUB_TOKEN"),
})
if err != nil {
    log.Fatal(err)
}
defer backend.Close()
```

## Supported Operations

| Operation | Supported | Notes |
|-----------|-----------|-------|
| `NewReader` | ✅ | Reads file content via Contents API |
| `NewWriter` | ✅ | Creates/updates files (each write = 1 commit) |
| `NewBatch` | ✅ | Atomic multi-file commits via Git Trees API |
| `Exists` | ✅ | Checks if file/directory exists |
| `Delete` | ✅ | Deletes files (each delete = 1 commit) |
| `List` | ✅ | Lists files via Trees API |
| `Stat` | ✅ | Returns size and SHA1 hash |
| `Copy` | ❌ | Returns `ErrNotSupported` |
| `Move` | ❌ | Returns `ErrNotSupported` |
| `Mkdir` | ❌ | Directories are implicit in Git |
| `Rmdir` | ❌ | Directories are implicit in Git |

## Use Cases

### Configuration Storage

Store application configuration in a Git repository for version control and audit trails:

```go
// Write config
w, _ := backend.NewWriter(ctx, "config/app.yaml")
w.Write(configData)
w.Close()

// Read config
r, _ := backend.NewReader(ctx, "config/app.yaml")
data, _ := io.ReadAll(r)
r.Close()
```

### Document Management

Use GitHub as a document store with full version history:

```go
// List all documents
files, _ := backend.List(ctx, "documents/")
for _, f := range files {
    fmt.Printf("%s (%d bytes)\n", f.Name, f.Size)
}
```

### Multi-Environment Configs

Use branches to manage environment-specific configurations:

```go
// Production config
prodBackend, _ := omnistorage.New(omnistorage.Config{
    Owner:  "myorg",
    Repo:   "config",
    Branch: "production",
    Token:  token,
})

// Staging config
stagingBackend, _ := omnistorage.New(omnistorage.Config{
    Owner:  "myorg",
    Repo:   "config",
    Branch: "staging",
    Token:  token,
})
```

## Limitations

- **File size**: GitHub Contents API supports files up to 1MB
- **Rate limits**: 5,000 requests/hour for authenticated users
- **Commits**: Each write creates a commit; use batch for bulk operations

## Next Steps

- [Configuration](configuration.md) - Detailed configuration options
- [Batch Operations](batch-operations.md) - Efficient multi-file commits
