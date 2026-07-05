# Batch Operations

For multiple file operations in a single commit, use the batch API. This is more efficient than individual writes and creates a cleaner commit history.

## Why Use Batches?

| Approach | Commits | API Calls | Best For |
|----------|---------|-----------|----------|
| Individual writes | 1 per file | 1 per file | Single file updates |
| Batch operations | 1 total | 3-4 total | Multiple file updates |

## Basic Usage

```go
// Create a batch with a commit message
batch, err := backend.NewBatch(ctx, "Update multiple files")
if err != nil {
    log.Fatal(err)
}

// Queue operations
batch.Write("file1.txt", []byte("content for file 1"))
batch.Write("file2.txt", []byte("content for file 2"))
batch.Delete("old-file.txt")

// Commit all changes atomically
if err := batch.Commit(); err != nil {
    log.Fatal(err)
}
```

## How It Works

The batch API uses Git's low-level APIs for efficiency:

1. **Get current tree** - Fetches the current branch's tree SHA
2. **Create blobs** - Creates blobs for new/updated file contents
3. **Create new tree** - Creates a new tree with all changes
4. **Create commit** - Creates a commit pointing to the new tree
5. **Update ref** - Updates the branch to point to the new commit

This results in a single atomic commit with all changes.

## Mixed Operations

Batches support both writes and deletes:

```go
batch, _ := backend.NewBatch(ctx, "Refactor: rename config files")

// Delete old files
batch.Delete("config/old-name.yaml")
batch.Delete("config/deprecated.yaml")

// Write new files
batch.Write("config/new-name.yaml", newConfig)
batch.Write("config/settings.yaml", settings)

batch.Commit()
```

## Error Handling

If any operation in the batch fails, the entire batch is rolled back:

```go
batch, err := backend.NewBatch(ctx, "Update files")
if err != nil {
    // Failed to initialize batch
    log.Fatal(err)
}

batch.Write("file1.txt", data1)
batch.Write("file2.txt", data2)

if err := batch.Commit(); err != nil {
    // Entire batch failed - no partial commits
    log.Printf("Batch failed: %v", err)
}
```

## Best Practices

### Group Related Changes

```go
// Good: Related changes in one batch
batch, _ := backend.NewBatch(ctx, "Add user feature")
batch.Write("models/user.go", userModel)
batch.Write("handlers/user.go", userHandler)
batch.Write("tests/user_test.go", userTest)
batch.Commit()
```

### Use Descriptive Commit Messages

```go
// Good: Clear commit message
batch, _ := backend.NewBatch(ctx, "fix: resolve config parsing issue #123")

// Bad: Generic message
batch, _ := backend.NewBatch(ctx, "update files")
```

### Batch Size Limits

While there's no hard limit on batch size, consider:

- GitHub API rate limits
- Large batches may timeout
- Recommended: < 100 files per batch

```go
// For large updates, split into multiple batches
for i := 0; i < len(files); i += 50 {
    batch, _ := backend.NewBatch(ctx, fmt.Sprintf("Update batch %d", i/50+1))
    for _, f := range files[i:min(i+50, len(files))] {
        batch.Write(f.Path, f.Content)
    }
    batch.Commit()
}
```
