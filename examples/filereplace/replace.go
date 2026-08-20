// Package filereplace demonstrates a staged file-replacement protocol.
//
// Replace uses os.Rename as its commit point. The Go documentation only
// guarantees atomic Rename behavior on Unix, so callers must decide whether
// this protocol is sufficient for their operating system and file system.
package filereplace

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type replaceStage string

const (
	stageTemporaryFileSynced replaceStage = "temporary-file-synced"
	stageTargetReplaced      replaceStage = "target-replaced"
)

// Replace writes data to a temporary file beside path, synchronizes and closes
// it, and then renames it over path. Creating the temporary file beside path
// keeps the rename on the same file system.
//
// On Unix, readers opening path during the rename see either the old file or
// the new file. The standard library does not promise that property on
// non-Unix systems. Replace also does not synchronize the parent directory, so
// a successful return is not a portable power-loss durability guarantee.
func Replace(path string, data []byte, perm fs.FileMode) error {
	return replace(path, data, perm, nil)
}

// replace accepts an observer so tests can stop a subprocess at exact protocol
// boundaries. Production callers use Replace and cannot inject behavior.
func replace(path string, data []byte, perm fs.FileMode, observe func(replaceStage)) (err error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	temporary, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %q: %w", path, err)
	}
	temporaryPath := temporary.Name()
	closed := false
	committed := false
	defer func() {
		if !closed {
			_ = temporary.Close()
		}
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := temporary.Chmod(perm.Perm()); err != nil {
		return fmt.Errorf("set temporary file mode for %q: %w", path, err)
	}
	if n, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write temporary file for %q: %w", path, err)
	} else if n != len(data) {
		return fmt.Errorf("write temporary file for %q: %w", path, io.ErrShortWrite)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary file for %q: %w", path, err)
	}
	if observe != nil {
		observe(stageTemporaryFileSynced)
	}
	if err := temporary.Close(); err != nil {
		closed = true
		return fmt.Errorf("close temporary file for %q: %w", path, err)
	}
	closed = true

	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace %q: %w", path, err)
	}
	committed = true
	if observe != nil {
		observe(stageTargetReplaced)
	}
	return nil
}
