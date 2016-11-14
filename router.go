package router

import (
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/alphastyle/logger"
)

// Mux is the multiplexer struct
type Mux struct {
	*http.ServeMux
	middle map[string][]http.HandlerFunc
}

// Gzip Compression
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

// Gzip Write
func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// New will create a new Mux
func New() *Mux {
	return &Mux{http.NewServeMux(), make(map[string][]http.HandlerFunc)}
}

// GET is a custom http.HandlerFunc that only allow GET requests
func (m *Mux) GET(pattern string, h http.HandlerFunc) {
	handler := m.checkMethod(h, "GET")
	m.Handle(pattern, http.HandlerFunc(handler))
}

// POST is a custom http.HandlerFunc that only allow POST requests
func (m *Mux) POST(pattern string, h http.HandlerFunc) {
	handler := m.checkMethod(h, "POST")
	m.Handle(pattern, http.HandlerFunc(handler))
}

// checkMethod will check the request method
// and handle middleware
func (m *Mux) checkMethod(h http.HandlerFunc, method string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == method {
			m.handleMiddleware(w, r)
			h(w, r)
		} else {
			http.NotFound(w, r)
		}
	}
}

// handleMiddleware will serve the correct middleware for the request
func (m *Mux) handleMiddleware(w http.ResponseWriter, r *http.Request) {
	// Global Middleware
	for _, v := range m.middle["GLOBAL"] {
		v(w, r)
	}

	// Group Middleware
	path := r.URL.Path

	for k := range m.middle {
		matched := strings.HasPrefix(path, k)
		if matched {
			for _, v := range m.middle[k] {
				v(w, r)
			}
		}
	}
}

// GlobalMiddleware is to make custome global middleware
func (m *Mux) GlobalMiddleware(h ...http.HandlerFunc) {
	for _, v := range h {
		m.middle["GLOBAL"] = append(m.middle["GLOBAL"], v)
	}
}

// GroupMiddleware is to make custome group middleware
func (m *Mux) GroupMiddleware(pattern string, h ...http.HandlerFunc) {
	if pattern != "" && strings.HasPrefix(pattern, "/") {
		for _, v := range h {
			m.middle[pattern] = append(m.middle[pattern], v)
		}
	} else {
		err := errors.New("Url pattern can't be empty and has to start with / (slash)!")
		logger.Error(err, "GroupMiddleware error")
	}
}

// Gzip compress all served files
func Gzip(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow the browser to cache content for 1 day (less traffic)
		w.Header().Set("Cache-Control", "max-age:86400")

		// if request does not accept Gzip then return without Gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			handler.ServeHTTP(w, r)
		}

		// Allow gzip
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzw := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		handler.ServeHTTP(gzw, r)
	})
}

// ServeFiles serve static files
func (m *Mux) ServeFiles(urlPath string, dirPath string, prefix string) {
	m.Handle(urlPath, Gzip(http.StripPrefix(prefix, http.FileServer(http.Dir(dirPath)))))
}

// ServeFavicon will serve the favicon you choose
func (m *Mux) ServeFavicon(filePath string) {
	m.GET("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filePath)
	})
}

// Listen will start the server (http.ListenAndServe)
func (m *Mux) Listen(serve ...string) error {
	// if serve parameter is empty then set default values
	if serve == nil {
		port := "8000"
		address := ""

		serve = []string{address, ":", port}
	}

	// join serve slice to make a string
	serveAt := strings.Join(serve, "")

	// ---- print "listening @ :PORT"" ----
	logger.Info("listening @" + serveAt)

	// start the server
	err := http.ListenAndServe(serveAt, m)
	return err
}
