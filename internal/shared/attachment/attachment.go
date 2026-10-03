// Package attachment holds the rules shared by every module that accepts an
// uploaded file: how big it may be, what kind it may be, and how it is named
// in object storage.
package attachment

import (
	"context"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/storage"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// MaxBytes is the largest accepted upload.
const MaxBytes = 10 << 20

var allowed = map[string]string{
	".pdf": "application/pdf", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".doc": "application/msword", ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls": "application/vnd.ms-excel", ".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".csv": "text/csv", ".txt": "text/plain",
}

var unsafe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// File is an accepted upload.
type File struct {
	Name        string // display name, path stripped
	ContentType string // derived from the extension, never trusted from the client
	Size        int64
}

// Check validates an upload and returns the normalised file. The content type
// comes from the extension so a client cannot label a script as a PDF.
func Check(filename string, content []byte) (File, error) {
	name := filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	name = strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r == '"' {
			return -1
		}
		return r
	}, name))
	if name == "" || name == "." || name == "/" {
		return File{}, apperrors.NewBadRequest("A file name is required")
	}
	if len(content) == 0 {
		return File{}, apperrors.NewBadRequest("The file is empty")
	}
	if len(content) > MaxBytes {
		return File{}, apperrors.NewBadRequest("The file is larger than 10 MB")
	}
	ct, ok := allowed[strings.ToLower(filepath.Ext(name))]
	if !ok {
		return File{}, apperrors.NewBadRequest("File type not allowed; use PDF, image, Word, Excel, CSV or text")
	}
	return File{Name: name, ContentType: ct, Size: int64(len(content))}, nil
}

// Key is the object-storage key for a file: a random id keeps uploads with the
// same name apart and makes keys unguessable.
func Key(prefix, name string) string {
	return prefix + "/" + uuid.NewString() + "/" + unsafe.ReplaceAllString(name, "_")
}

// Stored is what a module keeps on its record after a file is saved.
type Stored struct {
	File
	Key string
}

// Save validates and writes an upload to object storage.
func Save(ctx context.Context, store storage.Storage, prefix, filename string, content []byte) (*Stored, error) {
	if store == nil || !store.Enabled() {
		return nil, apperrors.NewServiceUnavailable("File storage is not configured. Set R2 credentials or LOCAL_STORAGE_DIR in the backend environment.")
	}
	f, err := Check(filename, content)
	if err != nil {
		return nil, err
	}
	key := Key(prefix, f.Name)
	if _, err := store.Upload(ctx, key, content, f.ContentType); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to store the file")
	}
	return &Stored{File: f, Key: key}, nil
}

// Load reads a stored file back.
func Load(ctx context.Context, store storage.Storage, key string) ([]byte, error) {
	if store == nil || !store.Enabled() {
		return nil, apperrors.NewServiceUnavailable("File storage is not configured.")
	}
	data, err := store.Download(ctx, key)
	if err != nil {
		return nil, apperrors.NewNotFound("The stored file could not be found")
	}
	return data, nil
}

// Discard removes a stored file; failures are logged, not returned, because the
// record it belonged to is already gone.
func Discard(ctx context.Context, store storage.Storage, key string) {
	if key == "" || store == nil || !store.Enabled() {
		return
	}
	if err := store.Delete(ctx, key); err != nil {
		slog.Warn("attachment: failed to delete stored file", "key", key, "error", err)
	}
}
