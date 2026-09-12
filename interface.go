package imaging

import (
	"io"
	"os"
	"path/filepath"
)

var fs fileSystem = localFS{}

// fileSystem is an interface for file system operations.
// Mainly used for testing purposes.
type fileSystem interface {
	// Create creates or truncates the named file.
	Create(string) (io.WriteCloser, error)
	// Open opens the named file for reading.
	Open(string) (io.ReadCloser, error)
}

// localFS implements fileSystem interface using local file system.
// It's used by default and same as os package.
type localFS struct{}

// Create implements fileSystem interface. Same as os.Create.
func (localFS) Create(name string) (io.WriteCloser, error) { return os.Create(filepath.Clean(name)) }

// Open implements fileSystem interface. Same as os.Open.
func (localFS) Open(name string) (io.ReadCloser, error) { return os.Open(filepath.Clean(name)) }
