package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/kite-plus/kite/internal/api"
)

// How much a project holds is measured by the site, and the endpoint passes on
// what it says rather than counting anything itself.
func TestStorageSaysHowMuchTheProjectHolds(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newServer(t, root, func(o *api.Options) {
		base := o.Site
		o.Site = func() api.View {
			v := base()
			v.Storage = func(context.Context) (api.Storage, error) {
				return api.Storage{Content: 4096, Files: 3}, nil
			}
			return v
		}
	})

	rec := send(t, h, http.MethodGet, api.Prefix+"/storage", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /storage: %d %s", rec.Code, rec.Body.String())
	}
	if got := decode[api.Storage](t, rec); got.Content != 4096 || got.Files != 3 {
		t.Errorf("got %+v, want 4096 bytes in 3 files", got)
	}
}

// A server that cannot look at the project says so, rather than answering zero
// bytes, which would read as an empty site.
func TestStorageIsNotAnsweredWithoutAMeasure(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newServer(t, root)

	rec := send(t, h, http.MethodGet, api.Prefix+"/storage", nil, nil)
	if rec.Code != http.StatusNotImplemented {
		t.Errorf("a server that cannot measure answered %d %s", rec.Code, rec.Body.String())
	}
}
