package github

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/go-github/v84/github"
	"github.com/grokify/gogithub/pathutil"
	omnistorage "github.com/plexusone/omnistorage-core/object"
)

// BatchOperation represents a single operation in a batch.
type BatchOperation struct {
	Type    BatchOperationType
	Path    string
	Content []byte
}

// BatchOperationType indicates the type of batch operation.
type BatchOperationType int

const (
	// BatchOpWrite represents a write (create/update) operation.
	BatchOpWrite BatchOperationType = iota
	// BatchOpDelete represents a delete operation.
	BatchOpDelete
)

// Batch accumulates multiple file operations to be committed atomically.
// Use NewBatch to create a batch, then call Write/Delete to queue operations,
// and finally Commit to apply all changes in a single commit.
type Batch struct {
	backend    *Backend
	ctx        context.Context
	message    string
	operations []BatchOperation
	committed  bool
	mu         sync.Mutex
}

// NewBatch creates a new batch for accumulating file operations.
// The message is used as the commit message when Commit is called.
func (b *Backend) NewBatch(ctx context.Context, message string) (*Batch, error) {
	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if message == "" {
		message = "Batch update via omnistorage"
	}

	return &Batch{
		backend:    b,
		ctx:        ctx,
		message:    message,
		operations: make([]BatchOperation, 0),
	}, nil
}

// Write queues a file write operation.
// The file will be created or updated when Commit is called.
func (batch *Batch) Write(filePath string, content []byte) error {
	batch.mu.Lock()
	defer batch.mu.Unlock()

	if batch.committed {
		return fmt.Errorf("github: batch already committed")
	}

	if err := pathutil.Validate(filePath); err != nil {
		return translatePathError(err)
	}

	if filePath == "" {
		return omnistorage.ErrInvalidPath
	}

	batch.operations = append(batch.operations, BatchOperation{
		Type:    BatchOpWrite,
		Path:    pathutil.Normalize(filePath),
		Content: content,
	})

	return nil
}

// Delete queues a file delete operation.
// The file will be deleted when Commit is called.
// If the file doesn't exist at commit time, it is ignored (no error).
func (batch *Batch) Delete(filePath string) error {
	batch.mu.Lock()
	defer batch.mu.Unlock()

	if batch.committed {
		return fmt.Errorf("github: batch already committed")
	}

	if err := pathutil.Validate(filePath); err != nil {
		return translatePathError(err)
	}

	if filePath == "" {
		return omnistorage.ErrInvalidPath
	}

	batch.operations = append(batch.operations, BatchOperation{
		Type: BatchOpDelete,
		Path: pathutil.Normalize(filePath),
	})

	return nil
}

// Len returns the number of operations in the batch.
func (batch *Batch) Len() int {
	batch.mu.Lock()
	defer batch.mu.Unlock()
	return len(batch.operations)
}

// Commit applies all queued operations in a single commit.
// This uses the Git Data API (Trees and Commits) to create an atomic commit.
//
// The process:
// 1. Get the current commit SHA from the branch reference
// 2. Get the current tree SHA from that commit
// 3. Create blobs for all new/updated file contents
// 4. Create a new tree with the changes
// 5. Create a new commit pointing to the new tree
// 6. Update the branch reference to the new commit
func (batch *Batch) Commit() error {
	batch.mu.Lock()
	defer batch.mu.Unlock()

	if batch.committed {
		return fmt.Errorf("github: batch already committed")
	}

	if err := batch.backend.checkClosed(); err != nil {
		return err
	}

	if err := batch.ctx.Err(); err != nil {
		return err
	}

	if len(batch.operations) == 0 {
		batch.committed = true
		return nil // Nothing to commit
	}

	// Step 1: Get the current branch reference
	ref, resp, err := batch.backend.client.Git.GetRef(
		batch.ctx,
		batch.backend.config.Owner,
		batch.backend.config.Repo,
		"refs/heads/"+batch.backend.config.Branch,
	)
	if err != nil {
		return batch.backend.translateError(err, resp)
	}

	currentCommitSHA := ref.Object.GetSHA()

	// Step 2: Get the current commit to find the tree SHA
	currentCommit, resp, err := batch.backend.client.Git.GetCommit(
		batch.ctx,
		batch.backend.config.Owner,
		batch.backend.config.Repo,
		currentCommitSHA,
	)
	if err != nil {
		return batch.backend.translateError(err, resp)
	}

	baseTreeSHA := currentCommit.Tree.GetSHA()

	// Step 3: Build tree entries for all operations
	treeEntries, err := batch.buildTreeEntries()
	if err != nil {
		return err
	}

	// Step 4: Create the new tree
	newTree, resp, err := batch.backend.client.Git.CreateTree(
		batch.ctx,
		batch.backend.config.Owner,
		batch.backend.config.Repo,
		baseTreeSHA,
		treeEntries,
	)
	if err != nil {
		return batch.backend.translateError(err, resp)
	}

	// Step 5: Create the new commit
	commitOpts := github.Commit{
		Message: github.Ptr(batch.message),
		Tree:    newTree,
		Parents: []*github.Commit{{SHA: github.Ptr(currentCommitSHA)}},
	}

	// Set commit author if configured
	if batch.backend.config.CommitAuthor != nil {
		commitOpts.Author = &github.CommitAuthor{
			Name:  github.Ptr(batch.backend.config.CommitAuthor.Name),
			Email: github.Ptr(batch.backend.config.CommitAuthor.Email),
		}
	}

	newCommit, resp, err := batch.backend.client.Git.CreateCommit(
		batch.ctx,
		batch.backend.config.Owner,
		batch.backend.config.Repo,
		commitOpts,
		nil, // CreateCommitOptions
	)
	if err != nil {
		return batch.backend.translateError(err, resp)
	}

	// Step 6: Update the branch reference
	newCommitSHA := newCommit.GetSHA()
	updateRef := github.UpdateRef{
		SHA:   newCommitSHA,
		Force: github.Ptr(false),
	}

	_, resp, err = batch.backend.client.Git.UpdateRef(
		batch.ctx,
		batch.backend.config.Owner,
		batch.backend.config.Repo,
		ref.GetRef(),
		updateRef,
	)
	if err != nil {
		return batch.backend.translateError(err, resp)
	}

	batch.committed = true
	return nil
}

// buildTreeEntries creates GitHub tree entries for all operations.
func (batch *Batch) buildTreeEntries() ([]*github.TreeEntry, error) {
	entries := make([]*github.TreeEntry, 0, len(batch.operations))

	for _, op := range batch.operations {
		switch op.Type {
		case BatchOpWrite:
			// Create a blob for the content
			blob, resp, err := batch.backend.client.Git.CreateBlob(
				batch.ctx,
				batch.backend.config.Owner,
				batch.backend.config.Repo,
				github.Blob{
					Content:  github.Ptr(string(op.Content)),
					Encoding: github.Ptr("utf-8"),
				},
			)
			if err != nil {
				return nil, batch.backend.translateError(err, resp)
			}

			entries = append(entries, &github.TreeEntry{
				Path: github.Ptr(op.Path),
				Mode: github.Ptr("100644"), // Regular file
				Type: github.Ptr("blob"),
				SHA:  blob.SHA,
			})

		case BatchOpDelete:
			// To delete a file, we set SHA to nil (or omit it) with the path
			// The GitHub API interprets this as a deletion when creating a tree
			// We need to check if the file exists first
			exists, err := batch.backend.Exists(batch.ctx, op.Path)
			if err != nil {
				return nil, err
			}
			if exists {
				// Use a nil SHA to indicate deletion
				entries = append(entries, &github.TreeEntry{
					Path: github.Ptr(op.Path),
					Mode: github.Ptr("100644"),
					Type: github.Ptr("blob"),
					SHA:  nil, // nil SHA means delete
				})
			}
			// If file doesn't exist, skip it (idempotent)
		}
	}

	return entries, nil
}
