package router

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/alphastyle/logger"
)

// mux is the multiplexer struct
type mux struct {
	*http.ServeMux
	middle     map[string][]handlerFunc
	middleHTTP map[string][]http.HandlerFunc
}

// Context is a custome ResponseWriter and Request
type Context struct {
	http.ResponseWriter
	*http.Request
}

type handlerFunc func(Context)

// Gzip Compression
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

// Gzip Write
func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// New will create a new mux
func New() *mux {
	return &mux{
		ServeMux:   http.NewServeMux(),
		middle:     make(map[string][]handlerFunc),
		middleHTTP: make(map[string][]http.HandlerFunc),
	}
}

// HandlerFunc is a custom http.HandlerFunc that only allow GET requests
func (m *mux) HandlerFunc(pattern string, h http.HandlerFunc) {
	handler := m.handleRequestHTTP(h, "ALL")
	m.Handle(pattern, http.HandlerFunc(handler))
}

// GET is a custom http.HandlerFunc that only allow GET requests
func (m *mux) GET(pattern string, h handlerFunc) {
	handler := m.handleRequest(h, "GET")
	m.Handle(pattern, http.HandlerFunc(handler))
}

// POST is a custom http.HandlerFunc that only allow POST requests
func (m *mux) POST(pattern string, h handlerFunc) {
	handler := m.handleRequest(h, "POST")
	m.Handle(pattern, http.HandlerFunc(handler))
}

// handleRequestHTTP will handle http.HandlerFunc requests
// and handle middleware
func (m *mux) handleRequestHTTP(h http.HandlerFunc, method string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == method && method == "ALL" {
			m.handleMiddlewareHTTP(w, r)
			h.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
	}
}

func (m handlerFunc) ServeHTTP(c Context) {
	m(c)
}

// handleRequest will check the request method
// and handle middleware
func (m *mux) handleRequest(h handlerFunc, method string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == method {
			mrw := Context{w, r}
			m.handleMiddleware(mrw)
			h.ServeHTTP(mrw)
		} else {
			http.NotFound(w, r)
		}
	}
}

// handleMiddlewareHTTP will serve the correct middleware for the request
// meant for http.HandlerFunc
func (m *mux) handleMiddlewareHTTP(w http.ResponseWriter, r *http.Request) {
	// Global Middleware
	for _, v := range m.middleHTTP["GLOBAL"] {
		v.ServeHTTP(w, r)
	}

	// Group Middleware
	// TODO make group middleware work for http.HandlerFunc (router.HandlerFunc)
	path := r.URL.Path

	for k := range m.middle {
		matched := strings.HasPrefix(path, k)
		if matched {
			for _, v := range m.middleHTTP[k] {
				v.ServeHTTP(w, r)
			}
		}
	}
}

// handleMiddleware will serve the correct middleware for the request
func (m *mux) handleMiddleware(c Context) {
	// Global Middleware
	for _, v := range m.middle["GLOBAL"] {
		v.ServeHTTP(c)
	}

	// Group Middleware
	path := c.Request.URL.Path

	for k := range m.middle {
		matched := strings.HasPrefix(path, k)
		if matched {
			for _, v := range m.middle[k] {
				v.ServeHTTP(c)
			}
		}
	}
}

// Use is to make custome global middleware
func (m *mux) Use(h ...handlerFunc) {
	for _, v := range h {
		m.middle["GLOBAL"] = append(m.middle["GLOBAL"], v)
	}
}

// Use is to make custome global middleware
func (m *mux) UseHTTP(h ...http.HandlerFunc) {
	for _, v := range h {
		m.middleHTTP["GLOBAL"] = append(m.middleHTTP["GLOBAL"], v)
	}
}

// GroupMiddleware is to make custome group middleware
func (m *mux) GroupMiddleware(pattern string, h ...handlerFunc) {
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
func (m *mux) ServeFiles(urlPath string, dirPath string, prefix string) {
	m.Handle(urlPath, Gzip(http.StripPrefix(prefix, http.FileServer(http.Dir(dirPath)))))
}

// ServeFavicon will serve the favicon you choose
func (m *mux) ServeFavicon(filePath string) {
	m.HandlerFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filePath)
	})
}

// JSON for json handling
func (ctx Context) JSON(data interface{}) {
	Data, err := json.Marshal(data)
	if err != nil {
		logger.Error(err, "JSON Marshal error")
	}

	ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
	ctx.ResponseWriter.Write(Data)
}

func (ctx Context) Write(str string) {
	ctx.ResponseWriter.Write([]byte(str))
}

// Listen will start the server (http.ListenAndServe)
func (m *mux) Listen(serve string) error {
	// ---- listening @ :PORT ----
	logger.Info("listening @" + serve)

	// start the server
	return http.ListenAndServe(serve, m)
}
