package action

import (
	"net/http"
	"strings"
)

// ServeGzipedFS serves pre-gziped files.
func ServeGzipedFS(prefix string, fs http.FileSystem) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var sb strings.Builder
		sb.WriteString(prefix)
		sb.WriteString(r.URL.Path)

		fp := sb.String()
		f, err := fs.Open(fp)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer f.Close() //nolint:errcheck

		stat, _ := f.Stat()
		w.Header().Add("Content-Encoding", "gzip")
		http.ServeContent(w, r, fp, stat.ModTime(), f)
	})
}
