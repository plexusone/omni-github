package github

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/grokify/gogithub"
	"github.com/grokify/gogithub/clientv1"
	gherrors "github.com/grokify/gogithub/errors"
	"github.com/grokify/gogithub/pathutil"
	omnistorage "github.com/plexusone/omnistorage-core/object"
)

const backendName = "github"

func init() {
	omnistorage.Register(backendName, func(config map[string]string) (omnistorage.Backend, error) {
		cfg := ConfigFromMap(config)
		return New(cfg)
	})
}

// Backend implements omnistorage.ExtendedBackend for GitHub repositories.
type Backend struct {
	client clientv1.Client
	config Config
	closed bool
	mu     sync.RWMutex
}

// writer is a buffered writer that commits content to GitHub on Close.
type writer struct {
	backend  *Backend
	ctx      context.Context
	filePath string
	buffer   *bytes.Buffer
	closed   bool
	mu       sync.Mutex
}

// New creates a new GitHub backend.
func New(cfg Config) (*Backend, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.github.com/"
	}
	if cfg.UploadURL == "" {
		cfg.UploadURL = "https://uploads.github.com/"
	}

	// Create clientv1.Client using options for GitHub Enterprise support
	var client clientv1.Client
	var err error

	if cfg.BaseURL != "https://api.github.com/" {
		// GitHub Enterprise
		client, err = clientv1.NewClientWithOptions(context.Background(), clientv1.ClientOptions{
			Token:     cfg.Token,
			BaseURL:   cfg.BaseURL,
			UploadURL: cfg.UploadURL,
		})
	} else {
		client, err = clientv1.NewClient(context.Background(), cfg.Token)
	}
	if err != nil {
		return nil, fmt.Errorf("github: creating client: %w", err)
	}

	return &Backend{
		client: client,
		config: cfg,
	}, nil
}

// NewWriter creates a writer for the given path.
// The content is buffered and committed to GitHub when Close() is called.
// Each Close() creates a new commit in the repository.
func (b *Backend) NewWriter(ctx context.Context, filePath string, opts ...omnistorage.WriterOption) (io.WriteCloser, error) {
	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := pathutil.Validate(filePath); err != nil {
		return nil, translatePathError(err)
	}

	if filePath == "" {
		return nil, omnistorage.ErrInvalidPath
	}

	return &writer{
		backend:  b,
		ctx:      ctx,
		filePath: pathutil.Normalize(filePath),
		buffer:   &bytes.Buffer{},
	}, nil
}

// Write writes data to the buffer.
func (w *writer) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, omnistorage.ErrWriterClosed
	}

	return w.buffer.Write(p)
}

// Close commits the buffered content to GitHub.
func (w *writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true

	// Check if backend is closed
	if err := w.backend.checkClosed(); err != nil {
		return err
	}

	// Check context
	if err := w.ctx.Err(); err != nil {
		return err
	}

	// Get existing file SHA if it exists (required for updates). The error is
	// ignored here: a lookup failure (e.g. file not found) is not fatal, it
	// just means existingSHA stays empty and we fall through to file creation.
	contentOpts := &gogithub.ContentOptions{Ref: w.backend.config.Branch}
	_, existingSHA, _ := w.backend.client.GetFileContentWithSHA(w.ctx, w.backend.config.Owner, w.backend.config.Repo, w.filePath, contentOpts)

	// Prepare commit options
	commitMessage := w.backend.config.FormatCommitMessage(w.filePath)

	var err error
	if existingSHA != "" {
		// Update existing file
		updateOpts := &clientv1.UpdateFileOptions{
			Content: w.buffer.Bytes(),
			SHA:     existingSHA,
			Message: commitMessage,
			Branch:  w.backend.config.Branch,
		}
		if w.backend.config.CommitAuthor != nil {
			updateOpts.Author = &clientv1.CommitAuthor{
				Name:  w.backend.config.CommitAuthor.Name,
				Email: w.backend.config.CommitAuthor.Email,
			}
		}
		_, err = w.backend.client.UpdateFile(w.ctx, w.backend.config.Owner, w.backend.config.Repo, w.filePath, updateOpts)
	} else {
		// Create new file
		createOpts := &clientv1.CreateFileOptions{
			Content: w.buffer.Bytes(),
			Message: commitMessage,
			Branch:  w.backend.config.Branch,
		}
		if w.backend.config.CommitAuthor != nil {
			createOpts.Author = &clientv1.CommitAuthor{
				Name:  w.backend.config.CommitAuthor.Name,
				Email: w.backend.config.CommitAuthor.Email,
			}
		}
		_, err = w.backend.client.CreateFile(w.ctx, w.backend.config.Owner, w.backend.config.Repo, w.filePath, createOpts)
	}

	if err != nil {
		return w.backend.translateError(err)
	}

	return nil
}

// NewReader creates a reader for the given path.
// Uses GitHub Contents API: GET /repos/{owner}/{repo}/contents/{path}?ref={branch}
func (b *Backend) NewReader(ctx context.Context, filePath string, opts ...omnistorage.ReaderOption) (io.ReadCloser, error) {
	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := pathutil.Validate(filePath); err != nil {
		return nil, translatePathError(err)
	}

	normalPath := pathutil.Normalize(filePath)

	// Get file content from GitHub
	contentOpts := &gogithub.ContentOptions{Ref: b.config.Branch}
	data, err := b.client.GetFileContent(ctx, b.config.Owner, b.config.Repo, normalPath, contentOpts)
	if err != nil {
		return nil, b.translateError(err)
	}

	// Apply reader options
	cfg := omnistorage.ApplyReaderOptions(opts...)

	// Handle offset
	if cfg.Offset > 0 {
		if cfg.Offset >= int64(len(data)) {
			data = []byte{}
		} else {
			data = data[cfg.Offset:]
		}
	}

	// Handle limit
	if cfg.Limit > 0 && int64(len(data)) > cfg.Limit {
		data = data[:cfg.Limit]
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

// Exists checks if a path exists.
func (b *Backend) Exists(ctx context.Context, filePath string) (bool, error) {
	if err := b.checkClosed(); err != nil {
		return false, err
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}

	if err := pathutil.Validate(filePath); err != nil {
		return false, translatePathError(err)
	}

	normalPath := pathutil.Normalize(filePath)
	contentOpts := &gogithub.ContentOptions{Ref: b.config.Branch}

	exists, err := b.client.FileExists(ctx, b.config.Owner, b.config.Repo, normalPath, contentOpts)
	if err != nil {
		return false, b.translateError(err)
	}

	return exists, nil
}

// Delete removes a file from the repository.
// This creates a new commit that deletes the file.
// Returns nil if the file does not exist (idempotent).
func (b *Backend) Delete(ctx context.Context, filePath string) error {
	if err := b.checkClosed(); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := pathutil.Validate(filePath); err != nil {
		return translatePathError(err)
	}

	if filePath == "" {
		return omnistorage.ErrInvalidPath
	}

	normalPath := pathutil.Normalize(filePath)
	contentOpts := &gogithub.ContentOptions{Ref: b.config.Branch}

	// Get existing file SHA (required for delete)
	_, sha, err := b.client.GetFileContentWithSHA(ctx, b.config.Owner, b.config.Repo, normalPath, contentOpts)
	if err != nil {
		// If file doesn't exist, return nil (idempotent)
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return b.translateError(err)
	}

	// Prepare delete options
	commitMessage := fmt.Sprintf("Delete %s via omnistorage", normalPath)
	if b.config.CommitMessage != "" {
		commitMessage = strings.ReplaceAll(b.config.CommitMessage, "{path}", normalPath)
		commitMessage = strings.ReplaceAll(commitMessage, "Update", "Delete")
	}

	deleteOpts := &clientv1.DeleteFileOptions{
		Branch: b.config.Branch,
	}
	if b.config.CommitAuthor != nil {
		deleteOpts.Author = &clientv1.CommitAuthor{
			Name:  b.config.CommitAuthor.Name,
			Email: b.config.CommitAuthor.Email,
		}
	}

	// Delete the file
	_, err = b.client.DeleteFile(ctx, b.config.Owner, b.config.Repo, normalPath, sha, commitMessage, deleteOpts)
	if err != nil {
		return b.translateError(err)
	}

	return nil
}

// List lists paths with the given prefix.
// Uses GitHub Trees API via clientv1.GetTree
func (b *Backend) List(ctx context.Context, prefix string) ([]string, error) {
	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	normalPrefix := pathutil.Normalize(prefix)

	// Get branch SHA first
	branchSHA, err := b.client.GetBranchSHA(ctx, b.config.Owner, b.config.Repo, b.config.Branch)
	if err != nil {
		return nil, b.translateError(err)
	}

	// Get the tree recursively
	entries, err := b.client.GetTree(ctx, b.config.Owner, b.config.Repo, branchSHA, true)
	if err != nil {
		return nil, b.translateError(err)
	}

	var paths []string
	for _, entry := range entries {
		// Check context on each iteration
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Skip directories (type "tree"), only include files (type "blob")
		if entry.Type != "blob" {
			continue
		}

		entryPath := entry.Path

		// Filter by prefix
		if normalPrefix != "" {
			if !strings.HasPrefix(entryPath, normalPrefix) &&
				!strings.HasPrefix(entryPath, normalPrefix+"/") {
				continue
			}
		}

		paths = append(paths, entryPath)
	}

	return paths, nil
}

// Close releases any resources held by the backend.
func (b *Backend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}

// Stat returns metadata about an object.
func (b *Backend) Stat(ctx context.Context, filePath string) (omnistorage.ObjectInfo, error) {
	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := pathutil.Validate(filePath); err != nil {
		return nil, translatePathError(err)
	}

	normalPath := pathutil.Normalize(filePath)
	contentOpts := &gogithub.ContentOptions{Ref: b.config.Branch}

	// Try to get file content first
	content, sha, err := b.client.GetFileContentWithSHA(ctx, b.config.Owner, b.config.Repo, normalPath, contentOpts)
	if err != nil {
		// Check if it's a directory by trying to list it
		entries, listErr := b.client.ListDirectory(ctx, b.config.Owner, b.config.Repo, normalPath, contentOpts)
		if listErr == nil && len(entries) > 0 {
			return &omnistorage.BasicObjectInfo{
				ObjectPath:  normalPath,
				ObjectSize:  0,
				ObjectIsDir: true,
			}, nil
		}
		return nil, b.translateError(err)
	}

	// It's a file
	return &omnistorage.BasicObjectInfo{
		ObjectPath:  normalPath,
		ObjectSize:  int64(len(content)),
		ObjectIsDir: false,
		ObjectHashes: map[omnistorage.HashType]string{
			omnistorage.HashSHA1: sha,
		},
	}, nil
}

// Mkdir returns ErrNotSupported (read-only backend).
func (b *Backend) Mkdir(ctx context.Context, filePath string) error {
	if err := b.checkClosed(); err != nil {
		return err
	}
	return omnistorage.ErrNotSupported
}

// Rmdir returns ErrNotSupported (read-only backend).
func (b *Backend) Rmdir(ctx context.Context, filePath string) error {
	if err := b.checkClosed(); err != nil {
		return err
	}
	return omnistorage.ErrNotSupported
}

// Copy returns ErrNotSupported (read-only backend).
func (b *Backend) Copy(ctx context.Context, src, dst string) error {
	if err := b.checkClosed(); err != nil {
		return err
	}
	return omnistorage.ErrNotSupported
}

// Move returns ErrNotSupported (read-only backend).
func (b *Backend) Move(ctx context.Context, src, dst string) error {
	if err := b.checkClosed(); err != nil {
		return err
	}
	return omnistorage.ErrNotSupported
}

// Features returns the capabilities of the GitHub backend.
func (b *Backend) Features() omnistorage.Features {
	return omnistorage.Features{
		Copy:                 false,
		Move:                 false,
		Mkdir:                false,
		Rmdir:                false,
		Stat:                 true,
		Hashes:               []omnistorage.HashType{omnistorage.HashSHA1},
		CanStream:            false, // Must buffer entire file
		ServerSideEncryption: false,
		Versioning:           true, // Git provides versioning via commits
		RangeRead:            true, // Implemented client-side
		ListPrefix:           true,
	}
}

// checkClosed returns an error if the backend is closed.
func (b *Backend) checkClosed() error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return omnistorage.ErrBackendClosed
	}
	return nil
}

// translateError converts errors to omnistorage errors.
func (b *Backend) translateError(err error) error {
	if err == nil {
		return nil
	}

	// Use gogithub's error translation
	ghErr := gherrors.Translate(err, nil)

	// Map gogithub errors to omnistorage errors
	if gherrors.IsNotFound(ghErr) {
		return omnistorage.ErrNotFound
	}
	if gherrors.IsPermissionDenied(ghErr) {
		return omnistorage.ErrPermissionDenied
	}

	return fmt.Errorf("github: %w", err)
}

// translatePathError converts pathutil errors to omnistorage errors.
func translatePathError(err error) error {
	if err == nil {
		return nil
	}
	if err == pathutil.ErrPathTraversal || err == pathutil.ErrInvalidPath {
		return omnistorage.ErrInvalidPath
	}
	return err
}

// Ensure Backend implements interfaces.
var (
	_ omnistorage.Backend         = (*Backend)(nil)
	_ omnistorage.ExtendedBackend = (*Backend)(nil)
)
