package comfig

import (
	"context"
	"fmt"
	"io/fs"
	"os"
)

type Source interface {
	Configuration(ctx context.Context, environment, extension string) ([]byte, error)
}

func NewFileSystemSource(fs fs.FS) Source {
	return fileSystemSource{fs: fs}
}

func NewFileSystemSourceByDirectory(dir string) Source {
	return NewFileSystemSource(os.DirFS(dir))
}

type fileSystemSource struct {
	fs fs.FS
}

func (s fileSystemSource) Configuration(_ context.Context, environment, extension string) ([]byte, error) {
	name := fmt.Sprintf("%s.%s", environment, extension)
	data, err := fs.ReadFile(s.fs, name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	return data, nil
}
