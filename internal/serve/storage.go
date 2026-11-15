package serve

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/kite-plus/kite/internal/api"
)

// contentDir is where a file-backed project keeps its content, as
// store/file.Path says. It is repeated here so that measuring a project does
// not make this package depend on the store.
const contentDir = "content"

// diskUsage adds up the regular files under root, which is how the studio says
// how much a project holds. A directory that does not exist yet holds nothing.
func diskUsage(ctx context.Context, root string) (api.Storage, error) {
	var out api.Storage
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		out.Content += info.Size()
		out.Files++
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return api.Storage{}, nil
	}
	if err != nil {
		return api.Storage{}, err
	}
	return out, nil
}
