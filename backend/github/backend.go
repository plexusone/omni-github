package github

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/google/go-github/v84/github"
	gherrors "github.com/grokify/gogithub/errors"
	"github.com/grokify/gogithub/pathutil"
	omnistorage "github.com/plexusone/omnistorage-core/object"
	"golang.org/x/oauth2"
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
	client *github.Client
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

	// Create OAuth2 token source
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.Token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	// Create GitHub client
	var client *github.Client
	var err error

	if cfg.BaseURL != "https://api.github.com/" {
		// GitHub Enterprise
		client, err = github.NewClient(tc).WithEnterpriseURLs(cfg.BaseURL, cfg.UploadURL)
		if err != nil {
			return nil, fmt.Errorf("github: creating enterprise client: %w", err)
		}
	} else {
		client = github.NewClient(tc)
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

	// Get existing file SHA if it exists (required for updates)
	var existingSHA *string
	fileContent, _, resp, err := w.backend.client.Repositories.GetContents(
		w.ctx,
		w.backend.config.Owner,
		w.backend.config.Repo,
		w.filePath,
		&github.RepositoryContentGetOptions{
			Ref: w.backend.config.Branch,
		},
	)
	if err == nil && fileContent != nil {
		sha := fileContent.GetSHA()
		existingSHA = &sha
	} else if resp != nil && resp.StatusCode != 404 {
		// Only ignore 404 errors (file doesn't exist yet)
		if errResp, ok := err.(*github.ErrorResponse); ok {
			if errResp.Response == nil || errResp.Response.StatusCode != 404 {
				return w.backend.translateError(err, resp)
			}
		} else {
			return w.backend.translateError(err, resp)
		}
	}

	// Prepare commit options
	commitMessage := w.backend.config.FormatCommitMessage(w.filePath)
	opts := &github.RepositoryContentFileOptions{
		Message: &commitMessage,
		Content: w.buffer.Bytes(),
		Branch:  &w.backend.config.Branch,
		SHA:     existingSHA,
	}

	// Set commit author if configured
	if w.backend.config.CommitAuthor != nil {
		opts.Author = &github.CommitAuthor{
			Name:  &w.backend.config.CommitAuthor.Name,
			Email: &w.backend.config.CommitAuthor.Email,
		}
	}

	// Create or update the file
	_, resp, err = w.backend.client.Repositories.CreateFile(
		w.ctx,
		w.backend.config.Owner,
		w.backend.config.Repo,
		w.filePath,
		opts,
	)
	if err != nil {
		return w.backend.translateError(err, resp)
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
	fileContent, _, resp, err := b.client.Repositories.GetContents(
		ctx,
		b.config.Owner,
		b.config.Repo,
		normalPath,
		&github.RepositoryContentGetOptions{
			Ref: b.config.Branch,
		},
	)
	if err != nil {
		return nil, b.translateError(err, resp)
	}

	// Check if it's a file (not a directory)
	if fileContent == nil || fileContent.GetType() != "file" {
		return nil, fmt.Errorf("github: path is a directory: %s", filePath)
	}

	// Decode content using the go-github helper
	content, err := fileContent.GetContent()
	if err != nil {
		return nil, fmt.Errorf("github: decoding content: %w", err)
	}

	data := []byte(content)

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

	_, _, resp, err := b.client.Repositories.GetContents(
		ctx,
		b.config.Owner,
		b.config.Repo,
		normalPath,
		&github.RepositoryContentGetOptions{
			Ref: b.config.Branch,
		},
	)

	if err != nil {
		// Check for 404
		if resp != nil && resp.StatusCode == 404 {
			return false, nil
		}
		// Check error type
		if errResp, ok := err.(*github.ErrorResponse); ok {
			if errResp.Response != nil && errResp.Response.StatusCode == 404 {
				return false, nil
			}
		}
		return false, b.translateError(err, resp)
	}

	return true, nil
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

	// Get existing file SHA (required for delete)
	fileContent, _, resp, err := b.client.Repositories.GetContents(
		ctx,
		b.config.Owner,
		b.config.Repo,
		normalPath,
		&github.RepositoryContentGetOptions{
			Ref: b.config.Branch,
		},
	)
	if err != nil {
		// If file doesn't exist, return nil (idempotent)
		if resp != nil && resp.StatusCode == 404 {
			return nil
		}
		if errResp, ok := err.(*github.ErrorResponse); ok {
			if errResp.Response != nil && errResp.Response.StatusCode == 404 {
				return nil
			}
		}
		return b.translateError(err, resp)
	}

	if fileContent == nil {
		// It's a directory, not a file
		return fmt.Errorf("github: cannot delete directory: %s", filePath)
	}

	// Prepare delete options
	commitMessage := fmt.Sprintf("Delete %s via omnistorage", normalPath)
	if b.config.CommitMessage != "" {
		commitMessage = strings.ReplaceAll(b.config.CommitMessage, "{path}", normalPath)
		commitMessage = strings.ReplaceAll(commitMessage, "Update", "Delete")
	}

	sha := fileContent.GetSHA()
	opts := &github.RepositoryContentFileOptions{
		Message: &commitMessage,
		SHA:     &sha,
		Branch:  &b.config.Branch,
	}

	// Set commit author if configured
	if b.config.CommitAuthor != nil {
		opts.Author = &github.CommitAuthor{
			Name:  &b.config.CommitAuthor.Name,
			Email: &b.config.CommitAuthor.Email,
		}
	}

	// Delete the file
	_, resp, err = b.client.Repositories.DeleteFile(
		ctx,
		b.config.Owner,
		b.config.Repo,
		normalPath,
		opts,
	)
	if err != nil {
		return b.translateError(err, resp)
	}

	return nil
}

// List lists paths with the given prefix.
// Uses GitHub Trees API: GET /repos/{owner}/{repo}/git/trees/{branch}?recursive=1
func (b *Backend) List(ctx context.Context, prefix string) ([]string, error) {
	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	normalPrefix := pathutil.Normalize(prefix)

	// Get the tree recursively
	tree, resp, err := b.client.Git.GetTree(
		ctx,
		b.config.Owner,
		b.config.Repo,
		b.config.Branch,
		true, // recursive
	)
	if err != nil {
		return nil, b.translateError(err, resp)
	}

	var paths []string
	for _, entry := range tree.Entries {
		// Check context on each iteration
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Skip directories (type "tree"), only include files (type "blob")
		if entry.GetType() != "blob" {
			continue
		}

		entryPath := entry.GetPath()

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

	fileContent, dirContents, resp, err := b.client.Repositories.GetContents(
		ctx,
		b.config.Owner,
		b.config.Repo,
		normalPath,
		&github.RepositoryContentGetOptions{
			Ref: b.config.Branch,
		},
	)
	if err != nil {
		return nil, b.translateError(err, resp)
	}

	// Check if it's a directory
	if dirContents != nil {
		return &omnistorage.BasicObjectInfo{
			ObjectPath:  normalPath,
			ObjectSize:  0,
			ObjectIsDir: true,
		}, nil
	}

	// It's a file
	return &omnistorage.BasicObjectInfo{
		ObjectPath:  normalPath,
		ObjectSize:  int64(fileContent.GetSize()),
		ObjectIsDir: false,
		ObjectHashes: map[omnistorage.HashType]string{
			omnistorage.HashSHA1: fileContent.GetSHA(),
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

// translateError converts GitHub API errors to omnistorage errors.
func (b *Backend) translateError(err error, resp *github.Response) error {
	if err == nil {
		return nil
	}

	// Use gogithub's error translation
	ghErr := gherrors.Translate(err, resp)

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
