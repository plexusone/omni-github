package github

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grokify/gogithub/pathutil"
	omnistorage "github.com/plexusone/omnistorage-core/object"
)

// skipIfNoToken skips the test if GITHUB_TOKEN is not set.
func skipIfNoToken(t *testing.T) {
	t.Helper()
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN not set, skipping integration test")
	}
}

// testBackend creates a backend for testing against grokify/omnistorage.
func testBackend(t *testing.T) *Backend {
	t.Helper()
	skipIfNoToken(t)

	backend, err := New(Config{
		Owner:  "grokify",
		Repo:   "omnistorage",
		Branch: "master",
		Token:  os.Getenv("GITHUB_TOKEN"),
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	return backend
}

func TestNewReader(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	// Read README.md (should exist in the repo)
	r, err := backend.NewReader(ctx, "README.md")
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer func() { _ = r.Close() }()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty content")
	}

	// Check that it contains expected content
	if !strings.Contains(string(data), "Omnistorage") {
		t.Error("Expected README to contain 'Omnistorage'")
	}
}

func TestNewReaderWithOffset(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	// Read with offset
	r, err := backend.NewReader(ctx, "README.md", omnistorage.WithOffset(2))
	if err != nil {
		t.Fatalf("NewReader with offset failed: %v", err)
	}
	defer func() { _ = r.Close() }()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	// Should not start with "# " since we skipped first 2 bytes
	if strings.HasPrefix(string(data), "# ") {
		t.Error("Expected content to not start with '# ' after offset")
	}
}

func TestNewReaderWithLimit(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	// Read with limit
	r, err := backend.NewReader(ctx, "README.md", omnistorage.WithLimit(10))
	if err != nil {
		t.Fatalf("NewReader with limit failed: %v", err)
	}
	defer func() { _ = r.Close() }()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(data) != 10 {
		t.Errorf("Expected 10 bytes, got %d", len(data))
	}
}

func TestNewReaderNotFound(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	_, err := backend.NewReader(ctx, "nonexistent-file-12345.txt")
	if err != omnistorage.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestExists(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	// Test existing file
	exists, err := backend.Exists(ctx, "README.md")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Expected README.md to exist")
	}

	// Test non-existent file
	exists, err = backend.Exists(ctx, "nonexistent-12345.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Expected nonexistent file to not exist")
	}
}

func TestList(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	paths, err := backend.List(ctx, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(paths) == 0 {
		t.Error("Expected non-empty list")
	}

	// Check that README.md is in the list
	found := false
	for _, p := range paths {
		if p == "README.md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected README.md in list")
	}
}

func TestListWithPrefix(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	// List files in backend directory
	paths, err := backend.List(ctx, "backend")
	if err != nil {
		t.Fatalf("List with prefix failed: %v", err)
	}

	// All paths should start with "backend/"
	for _, p := range paths {
		if !strings.HasPrefix(p, "backend/") && p != "backend" {
			t.Errorf("Path %q does not start with 'backend/'", p)
		}
	}
}

func TestStat(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	info, err := backend.Stat(ctx, "README.md")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info.Path() != "README.md" {
		t.Errorf("Path = %q, want %q", info.Path(), "README.md")
	}

	if info.Size() == 0 {
		t.Error("Expected non-zero size")
	}

	if info.IsDir() {
		t.Error("Expected IsDir to be false")
	}

	// Check that SHA1 hash is present
	sha := info.Hash(omnistorage.HashSHA1)
	if sha == "" {
		t.Error("Expected SHA1 hash to be present")
	}
}

func TestStatDirectory(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	info, err := backend.Stat(ctx, "backend")
	if err != nil {
		t.Fatalf("Stat directory failed: %v", err)
	}

	if !info.IsDir() {
		t.Error("Expected IsDir to be true for directory")
	}
}

func TestStatNotFound(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	_, err := backend.Stat(ctx, "nonexistent-12345.txt")
	if err != omnistorage.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

// skipIfNoWriteRepo skips the test if write testing is not enabled.
// Write tests modify the repository, so they require explicit opt-in.
func skipIfNoWriteRepo(t *testing.T) {
	t.Helper()
	if os.Getenv("OMNISTORAGE_GITHUB_TEST_WRITE") != "true" {
		t.Skip("OMNISTORAGE_GITHUB_TEST_WRITE not set to 'true', skipping write test")
	}
}

// writeTestBackend creates a backend for write testing.
// Uses OMNISTORAGE_GITHUB_TEST_WRITE_REPO or falls back to the read test repo.
func writeTestBackend(t *testing.T) *Backend {
	t.Helper()
	skipIfNoToken(t)
	skipIfNoWriteRepo(t)

	repo := os.Getenv("OMNISTORAGE_GITHUB_TEST_WRITE_REPO")
	if repo == "" {
		repo = "omnistorage"
	}

	backend, err := New(Config{
		Owner:  "grokify",
		Repo:   repo,
		Branch: "master",
		Token:  os.Getenv("GITHUB_TOKEN"),
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	return backend
}

func TestNewWriter(t *testing.T) {
	backend := writeTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	testPath := fmt.Sprintf("test/omnistorage-test-%d.txt", time.Now().UnixNano())

	// Create a new file
	w, err := backend.NewWriter(ctx, testPath)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	testContent := []byte("Hello from omnistorage-github test!")
	_, err = w.Write(testContent)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	err = w.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify the file was created
	r, err := backend.NewReader(ctx, testPath)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer func() { _ = r.Close() }()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if string(data) != string(testContent) {
		t.Errorf("Content mismatch: got %q, want %q", string(data), string(testContent))
	}

	// Clean up - delete the test file
	err = backend.Delete(ctx, testPath)
	if err != nil {
		t.Logf("Warning: failed to clean up test file %s: %v", testPath, err)
	}
}

func TestNewWriterUpdate(t *testing.T) {
	backend := writeTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	testPath := fmt.Sprintf("test/omnistorage-test-%d.txt", time.Now().UnixNano())

	// Create initial file
	w1, err := backend.NewWriter(ctx, testPath)
	if err != nil {
		t.Fatalf("NewWriter (create) failed: %v", err)
	}
	_, _ = w1.Write([]byte("initial content"))
	if err := w1.Close(); err != nil {
		t.Fatalf("Close (create) failed: %v", err)
	}

	// Update the file
	w2, err := backend.NewWriter(ctx, testPath)
	if err != nil {
		t.Fatalf("NewWriter (update) failed: %v", err)
	}
	updatedContent := []byte("updated content")
	_, _ = w2.Write(updatedContent)
	if err := w2.Close(); err != nil {
		t.Fatalf("Close (update) failed: %v", err)
	}

	// Verify the update
	r, err := backend.NewReader(ctx, testPath)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer func() { _ = r.Close() }()

	data, _ := io.ReadAll(r)
	if string(data) != string(updatedContent) {
		t.Errorf("Content mismatch after update: got %q, want %q", string(data), string(updatedContent))
	}

	// Clean up
	_ = backend.Delete(ctx, testPath)
}

func TestDelete(t *testing.T) {
	backend := writeTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	testPath := fmt.Sprintf("test/omnistorage-delete-test-%d.txt", time.Now().UnixNano())

	// Create a file to delete
	w, err := backend.NewWriter(ctx, testPath)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	_, _ = w.Write([]byte("file to delete"))
	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify file exists
	exists, err := backend.Exists(ctx, testPath)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Fatal("Expected file to exist before delete")
	}

	// Delete the file
	err = backend.Delete(ctx, testPath)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify file no longer exists
	exists, err = backend.Exists(ctx, testPath)
	if err != nil {
		t.Fatalf("Exists after delete failed: %v", err)
	}
	if exists {
		t.Error("Expected file to not exist after delete")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	// Delete a file that doesn't exist - should be idempotent
	err := backend.Delete(ctx, "nonexistent-file-12345.txt")
	if err != nil {
		t.Errorf("Delete of non-existent file should not error, got: %v", err)
	}
}

func TestNewWriterInvalidPath(t *testing.T) {
	// This test doesn't need API access - it tests path validation
	backend, err := New(Config{
		Owner: "test",
		Repo:  "test",
		Token: "dummy-token",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	// Empty path should error
	_, err = backend.NewWriter(ctx, "")
	if err != omnistorage.ErrInvalidPath {
		t.Errorf("Expected ErrInvalidPath for empty path, got: %v", err)
	}

	// Path traversal should error
	_, err = backend.NewWriter(ctx, "../escape.txt")
	if err != omnistorage.ErrInvalidPath {
		t.Errorf("Expected ErrInvalidPath for path traversal, got: %v", err)
	}
}

func TestCopyNotSupported(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	err := backend.Copy(ctx, "src.txt", "dst.txt")
	if err != omnistorage.ErrNotSupported {
		t.Errorf("Expected ErrNotSupported, got: %v", err)
	}
}

func TestMoveNotSupported(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	err := backend.Move(ctx, "src.txt", "dst.txt")
	if err != omnistorage.ErrNotSupported {
		t.Errorf("Expected ErrNotSupported, got: %v", err)
	}
}

func TestMkdirNotSupported(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	err := backend.Mkdir(ctx, "newdir")
	if err != omnistorage.ErrNotSupported {
		t.Errorf("Expected ErrNotSupported, got: %v", err)
	}
}

func TestRmdirNotSupported(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	err := backend.Rmdir(ctx, "somedir")
	if err != omnistorage.ErrNotSupported {
		t.Errorf("Expected ErrNotSupported, got: %v", err)
	}
}

func TestClose(t *testing.T) {
	backend := testBackend(t)

	if err := backend.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	ctx := context.Background()

	// All operations should fail after close
	_, err := backend.NewReader(ctx, "README.md")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("Expected ErrBackendClosed after close, got: %v", err)
	}

	_, err = backend.Exists(ctx, "README.md")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("Expected ErrBackendClosed for Exists after close, got: %v", err)
	}

	_, err = backend.List(ctx, "")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("Expected ErrBackendClosed for List after close, got: %v", err)
	}
}

func TestFeatures(t *testing.T) {
	backend := testBackend(t)
	defer func() { _ = backend.Close() }()

	features := backend.Features()

	if features.Copy {
		t.Error("Expected Copy to be false")
	}
	if features.Move {
		t.Error("Expected Move to be false")
	}
	if features.Mkdir {
		t.Error("Expected Mkdir to be false")
	}
	if !features.Stat {
		t.Error("Expected Stat to be true")
	}
	if !features.Versioning {
		t.Error("Expected Versioning to be true")
	}
	if !features.RangeRead {
		t.Error("Expected RangeRead to be true")
	}
	if !features.ListPrefix {
		t.Error("Expected ListPrefix to be true")
	}
	if !features.SupportsHash(omnistorage.HashSHA1) {
		t.Error("Expected SHA1 hash to be supported")
	}
}

// Unit tests that don't require API access

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name:    "missing owner",
			config:  Config{Repo: "test", Token: "token"},
			wantErr: ErrOwnerRequired,
		},
		{
			name:    "missing repo",
			config:  Config{Owner: "owner", Token: "token"},
			wantErr: ErrRepoRequired,
		},
		{
			name:    "missing token",
			config:  Config{Owner: "owner", Repo: "repo"},
			wantErr: ErrTokenRequired,
		},
		{
			name:    "valid config",
			config:  Config{Owner: "owner", Repo: "repo", Token: "token"},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigFromMap(t *testing.T) {
	m := map[string]string{
		"owner":      "grokify",
		"repo":       "omnistorage",
		"branch":     "develop",
		"token":      "test-token",
		"base_url":   "https://github.example.com/api/v3/",
		"upload_url": "https://github.example.com/uploads/",
	}

	cfg := ConfigFromMap(m)

	if cfg.Owner != "grokify" {
		t.Errorf("Owner = %q, want %q", cfg.Owner, "grokify")
	}
	if cfg.Repo != "omnistorage" {
		t.Errorf("Repo = %q, want %q", cfg.Repo, "omnistorage")
	}
	if cfg.Branch != "develop" {
		t.Errorf("Branch = %q, want %q", cfg.Branch, "develop")
	}
	if cfg.Token != "test-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "test-token")
	}
	if cfg.BaseURL != "https://github.example.com/api/v3/" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://github.example.com/api/v3/")
	}
	if cfg.UploadURL != "https://github.example.com/uploads/" {
		t.Errorf("UploadURL = %q, want %q", cfg.UploadURL, "https://github.example.com/uploads/")
	}
}

func TestConfigFromMapDefaults(t *testing.T) {
	m := map[string]string{
		"owner": "grokify",
		"repo":  "omnistorage",
		"token": "test-token",
	}

	cfg := ConfigFromMap(m)

	// Check defaults are applied
	if cfg.Branch != "main" {
		t.Errorf("Branch = %q, want default %q", cfg.Branch, "main")
	}
	if cfg.BaseURL != "https://api.github.com/" {
		t.Errorf("BaseURL = %q, want default %q", cfg.BaseURL, "https://api.github.com/")
	}
	if cfg.UploadURL != "https://uploads.github.com/" {
		t.Errorf("UploadURL = %q, want default %q", cfg.UploadURL, "https://uploads.github.com/")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Branch != "main" {
		t.Errorf("DefaultConfig Branch = %q, want %q", cfg.Branch, "main")
	}
	if cfg.BaseURL != "https://api.github.com/" {
		t.Errorf("DefaultConfig BaseURL = %q, want %q", cfg.BaseURL, "https://api.github.com/")
	}
	if cfg.UploadURL != "https://uploads.github.com/" {
		t.Errorf("DefaultConfig UploadURL = %q, want %q", cfg.UploadURL, "https://uploads.github.com/")
	}
	if cfg.CommitMessage != "Update {path} via omnistorage" {
		t.Errorf("DefaultConfig CommitMessage = %q, want %q", cfg.CommitMessage, "Update {path} via omnistorage")
	}
}

func TestFormatCommitMessage(t *testing.T) {
	cfg := Config{CommitMessage: "Custom: {path}"}
	got := cfg.FormatCommitMessage("test/file.txt")
	want := "Custom: test/file.txt"
	if got != want {
		t.Errorf("FormatCommitMessage = %q, want %q", got, want)
	}

	// Test with empty message (should use default)
	cfg2 := Config{}
	got2 := cfg2.FormatCommitMessage("test.txt")
	want2 := "Update test.txt via omnistorage"
	if got2 != want2 {
		t.Errorf("FormatCommitMessage with empty = %q, want %q", got2, want2)
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{"", false},               // empty path is valid (root)
		{"file.txt", false},       // simple file
		{"dir/file.txt", false},   // nested file
		{"../escape.txt", true},   // path traversal
		{"dir/../file.txt", true}, // path traversal
		{"a/b/c/d.txt", false},    // deeply nested
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := pathutil.Validate(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("pathutil.Validate(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"file.txt", "file.txt"},
		{"/file.txt", "file.txt"},
		{"./file.txt", "file.txt"},
		{"dir/file.txt", "dir/file.txt"},
		{"/dir/file.txt", "dir/file.txt"},
		{".", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := pathutil.Normalize(tt.input)
			if got != tt.want {
				t.Errorf("pathutil.Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Batch operation tests

func TestBatchWrite(t *testing.T) {
	backend := writeTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	timestamp := time.Now().UnixNano()
	testPath1 := fmt.Sprintf("test/batch-test-%d-1.txt", timestamp)
	testPath2 := fmt.Sprintf("test/batch-test-%d-2.txt", timestamp)

	// Create a batch
	batch, err := backend.NewBatch(ctx, "Batch test commit")
	if err != nil {
		t.Fatalf("NewBatch failed: %v", err)
	}

	// Add write operations
	content1 := []byte("Content for file 1")
	content2 := []byte("Content for file 2")
	if err := batch.Write(testPath1, content1); err != nil {
		t.Fatalf("batch.Write(1) failed: %v", err)
	}
	if err := batch.Write(testPath2, content2); err != nil {
		t.Fatalf("batch.Write(2) failed: %v", err)
	}

	if batch.Len() != 2 {
		t.Errorf("batch.Len() = %d, want 2", batch.Len())
	}

	// Commit the batch
	if err := batch.Commit(); err != nil {
		t.Fatalf("batch.Commit failed: %v", err)
	}

	// Verify both files were created
	r1, err := backend.NewReader(ctx, testPath1)
	if err != nil {
		t.Fatalf("NewReader(1) failed: %v", err)
	}
	data1, _ := io.ReadAll(r1)
	_ = r1.Close()
	if string(data1) != string(content1) {
		t.Errorf("Content1 mismatch: got %q, want %q", string(data1), string(content1))
	}

	r2, err := backend.NewReader(ctx, testPath2)
	if err != nil {
		t.Fatalf("NewReader(2) failed: %v", err)
	}
	data2, _ := io.ReadAll(r2)
	_ = r2.Close()
	if string(data2) != string(content2) {
		t.Errorf("Content2 mismatch: got %q, want %q", string(data2), string(content2))
	}

	// Clean up
	_ = backend.Delete(ctx, testPath1)
	_ = backend.Delete(ctx, testPath2)
}

func TestBatchWriteAndDelete(t *testing.T) {
	backend := writeTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	timestamp := time.Now().UnixNano()
	testPath1 := fmt.Sprintf("test/batch-mixed-%d-1.txt", timestamp)
	testPath2 := fmt.Sprintf("test/batch-mixed-%d-2.txt", timestamp)

	// First create a file to delete
	w, err := backend.NewWriter(ctx, testPath2)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	_, _ = w.Write([]byte("file to delete"))
	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Now create a batch that writes one file and deletes another
	batch, err := backend.NewBatch(ctx, "Mixed batch operation")
	if err != nil {
		t.Fatalf("NewBatch failed: %v", err)
	}

	content1 := []byte("New file content")
	if err := batch.Write(testPath1, content1); err != nil {
		t.Fatalf("batch.Write failed: %v", err)
	}
	if err := batch.Delete(testPath2); err != nil {
		t.Fatalf("batch.Delete failed: %v", err)
	}

	// Commit
	if err := batch.Commit(); err != nil {
		t.Fatalf("batch.Commit failed: %v", err)
	}

	// Verify file1 was created
	exists1, err := backend.Exists(ctx, testPath1)
	if err != nil {
		t.Fatalf("Exists(1) failed: %v", err)
	}
	if !exists1 {
		t.Error("Expected testPath1 to exist after batch commit")
	}

	// Verify file2 was deleted
	exists2, err := backend.Exists(ctx, testPath2)
	if err != nil {
		t.Fatalf("Exists(2) failed: %v", err)
	}
	if exists2 {
		t.Error("Expected testPath2 to not exist after batch commit")
	}

	// Clean up
	_ = backend.Delete(ctx, testPath1)
}

func TestBatchEmptyCommit(t *testing.T) {
	backend := writeTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	// Create and commit an empty batch
	batch, err := backend.NewBatch(ctx, "Empty batch")
	if err != nil {
		t.Fatalf("NewBatch failed: %v", err)
	}

	// Commit should succeed with no operations
	if err := batch.Commit(); err != nil {
		t.Errorf("Empty batch.Commit should not fail: %v", err)
	}
}

func TestBatchDoubleCommit(t *testing.T) {
	backend := writeTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	batch, err := backend.NewBatch(ctx, "Double commit test")
	if err != nil {
		t.Fatalf("NewBatch failed: %v", err)
	}

	// First commit should succeed
	if err := batch.Commit(); err != nil {
		t.Fatalf("First Commit failed: %v", err)
	}

	// Second commit should fail
	if err := batch.Commit(); err == nil {
		t.Error("Expected error on second Commit")
	}
}

func TestBatchWriteAfterCommit(t *testing.T) {
	backend := writeTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	batch, err := backend.NewBatch(ctx, "Write after commit test")
	if err != nil {
		t.Fatalf("NewBatch failed: %v", err)
	}

	if err := batch.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Write after commit should fail
	if err := batch.Write("test.txt", []byte("content")); err == nil {
		t.Error("Expected error on Write after Commit")
	}
}

func TestBatchDeleteAfterCommit(t *testing.T) {
	backend := writeTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	batch, err := backend.NewBatch(ctx, "Delete after commit test")
	if err != nil {
		t.Fatalf("NewBatch failed: %v", err)
	}

	if err := batch.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Delete after commit should fail
	if err := batch.Delete("test.txt"); err == nil {
		t.Error("Expected error on Delete after Commit")
	}
}

func TestBatchInvalidPath(t *testing.T) {
	backend, err := New(Config{
		Owner: "test",
		Repo:  "test",
		Token: "dummy-token",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = backend.Close() }()

	ctx := context.Background()

	batch, err := backend.NewBatch(ctx, "Invalid path test")
	if err != nil {
		t.Fatalf("NewBatch failed: %v", err)
	}

	// Empty path should error
	if err := batch.Write("", []byte("content")); err != omnistorage.ErrInvalidPath {
		t.Errorf("Expected ErrInvalidPath for empty path, got: %v", err)
	}

	// Path traversal should error
	if err := batch.Write("../escape.txt", []byte("content")); err != omnistorage.ErrInvalidPath {
		t.Errorf("Expected ErrInvalidPath for path traversal, got: %v", err)
	}

	// Same for Delete
	if err := batch.Delete(""); err != omnistorage.ErrInvalidPath {
		t.Errorf("Expected ErrInvalidPath for empty delete path, got: %v", err)
	}

	if err := batch.Delete("../escape.txt"); err != omnistorage.ErrInvalidPath {
		t.Errorf("Expected ErrInvalidPath for path traversal delete, got: %v", err)
	}
}

func TestBatchDeleteNonExistent(t *testing.T) {
	backend := writeTestBackend(t)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	timestamp := time.Now().UnixNano()
	testPath := fmt.Sprintf("test/batch-new-%d.txt", timestamp)

	// Create a batch that creates one file and tries to delete a non-existent file
	batch, err := backend.NewBatch(ctx, "Delete non-existent test")
	if err != nil {
		t.Fatalf("NewBatch failed: %v", err)
	}

	if err := batch.Write(testPath, []byte("new content")); err != nil {
		t.Fatalf("batch.Write failed: %v", err)
	}

	// Delete a file that doesn't exist - should be silently ignored
	if err := batch.Delete("nonexistent-file-12345.txt"); err != nil {
		t.Fatalf("batch.Delete should not fail for queue: %v", err)
	}

	// Commit should succeed - the non-existent delete is just skipped
	if err := batch.Commit(); err != nil {
		t.Fatalf("batch.Commit failed: %v", err)
	}

	// Verify the new file was created
	exists, err := backend.Exists(ctx, testPath)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Expected new file to exist after batch commit")
	}

	// Clean up
	_ = backend.Delete(ctx, testPath)
}

func TestNewBatchAfterClose(t *testing.T) {
	backend := testBackend(t)

	if err := backend.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	ctx := context.Background()

	_, err := backend.NewBatch(ctx, "Test")
	if err != omnistorage.ErrBackendClosed {
		t.Errorf("Expected ErrBackendClosed, got: %v", err)
	}
}
