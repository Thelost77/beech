package storage

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrConflict means that another process changed the file after Beech loaded it.
var ErrConflict = errors.New("file changed outside Beech")

// Fingerprint identifies the file content used by the current document.
type Fingerprint struct {
	Exists bool
	Hash   [sha256.Size]byte
}

// LoadedFile is the result of loading a document path.
type LoadedFile struct {
	Path        string
	Data        []byte
	Fingerprint Fingerprint
	Mode        fs.FileMode
}

// Load reads path and resolves a symlink so atomic saves do not replace it.
func Load(path string) (LoadedFile, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return LoadedFile{}, err
	}
	resolved := absolute
	if target, err := filepath.EvalSymlinks(absolute); err == nil {
		resolved = target
	} else if !errors.Is(err, os.ErrNotExist) {
		return LoadedFile{}, err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return LoadedFile{}, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return LoadedFile{}, err
	}
	return LoadedFile{
		Path:        resolved,
		Data:        data,
		Fingerprint: fingerprint(data),
		Mode:        info.Mode().Perm(),
	}, nil
}

// NewPath resolves a path for a document that does not exist yet.
func NewPath(path string) (string, error) {
	return filepath.Abs(path)
}

// Save atomically writes data if the target still matches expected.
func Save(path string, data []byte, expected Fingerprint, mode fs.FileMode) (Fingerprint, error) {
	if path == "" {
		return Fingerprint{}, errors.New("save path is empty")
	}
	current, err := currentFingerprint(path)
	if err != nil {
		return Fingerprint{}, err
	}
	if current != expected {
		return Fingerprint{}, ErrConflict
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Fingerprint{}, fmt.Errorf("create parent directory: %w", err)
	}
	if mode == 0 {
		mode = 0o644
	}
	tmp, err := os.CreateTemp(dir, ".beech-save-*")
	if err != nil {
		return Fingerprint{}, fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return Fingerprint{}, fmt.Errorf("set temporary file permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return Fingerprint{}, fmt.Errorf("write temporary file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return Fingerprint{}, fmt.Errorf("flush temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return Fingerprint{}, fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return Fingerprint{}, fmt.Errorf("replace document: %w", err)
	}
	removeTemp = false
	return fingerprint(data), nil
}

func currentFingerprint(path string) (Fingerprint, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Fingerprint{}, nil
	}
	if err != nil {
		return Fingerprint{}, fmt.Errorf("read current document: %w", err)
	}
	return fingerprint(data), nil
}

func fingerprint(data []byte) Fingerprint {
	return Fingerprint{Exists: true, Hash: sha256.Sum256(data)}
}
