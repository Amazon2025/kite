package api

import "net/http"

// Storage is how much of a project a site keeps on disk.
type Storage struct {
	// Content is the size in bytes of the files under content/: the markdown
	// and every file a page keeps beside it.
	Content int64 `json:"content"`
	// Files counts those files.
	Files int `json:"files"`
}

// handleStorage says how much the project holds.
//
// The count is made by the site, not here: where a project keeps its files is
// the store's business, and a handler that walked the disk itself would be a
// second answer to a question that already has one.
func (s *Server) handleStorage(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if view.Storage == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot measure the project")
		return
	}

	used, err := view.Storage(r.Context())
	if err != nil {
		s.failErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, used)
}
