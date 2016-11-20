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
	middle     []handlerFunc
	middleHTTP []http.HandlerFunc
}

// group is to divide request middleware
type group struct {
	*mux
	middleware []handlerFunc
	prefix     string
}

// Context is a custome ResponseWriter and Request
type Context struct {
	http.ResponseWriter
	*http.Request
}

// handlerFunc is custome http.HandleFunc type
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
func New() *group {
	m := &mux{
		ServeMux: http.NewServeMux(),
	}

	return &group{
		mux: m,
	}
}

func (h handlerFunc) ServeHTTP(c Context) {
	h(c)
}

// HandlerFunc is a custom http.HandlerFunc
func (g *group) HandlerFunc(pattern string, h http.HandlerFunc) {
	handler := g.handleRequestHTTP(h, "ALL")
	g.Handle(g.prefix+pattern, http.HandlerFunc(handler))
}

// GET is a custom http.HandlerFunc that only allow GET requests
func (g *group) GET(pattern string, h handlerFunc) {
	handler := g.handleRequest(h, "GET")
	g.Handle(g.prefix+pattern, http.HandlerFunc(handler))
}

// POST is a custom http.HandlerFunc that only allow POST requests
func (g *group) POST(pattern string, h handlerFunc) {
	handler := g.handleRequest(h, "POST")
	g.Handle(g.prefix+pattern, http.HandlerFunc(handler))
}

// handleRequestHTTP will handle http.HandlerFunc requests
// and handle middleware
func (g *group) handleRequestHTTP(h http.HandlerFunc, method string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if method == "ALL" {
			g.handleMiddlewareHTTP(w, r)
			h.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
	}
}

// handleRequest will check the request method and handle middleware
func (g *group) handleRequest(h handlerFunc, method string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == method {
			mrw := Context{w, r}
			g.handleMiddleware(mrw)
			h.ServeHTTP(mrw)
		} else {
			http.NotFound(w, r)
		}
	}
}

// handleMiddlewareHTTP will serve the correct middleware for the request
// meant for http.HandlerFunc
func (g *group) handleMiddlewareHTTP(w http.ResponseWriter, r *http.Request) {
	// Global Middleware
	for _, v := range g.middleHTTP {
		v.ServeHTTP(w, r)
	}

	// Group Middleware
	// TODO make middleware work for HandlerFunc (http.HanderFunc)
	// for _, v := range g.middleware {
	// 	v.ServeHTTP(w, r)
	// }
}

// handleMiddleware will serve the correct middleware for the request
func (g *group) handleMiddleware(c Context) {
	// Global Middleware
	for _, v := range g.middle {
		v.ServeHTTP(c)
	}

	// Group Middleware
	for _, v := range g.middleware {
		v.ServeHTTP(c)
	}
}

// Use is to make custome global middleware
func (g *group) Use(h ...handlerFunc) {
	if g.prefix == "" {
		for _, v := range h {
			g.middle = append(g.middle, v)
		}
	}
}

// Use is to make custome global middleware
func (g *group) UseHTTP(h ...http.HandlerFunc) {
	if g.prefix == "" {
		for _, v := range h {
			g.middleHTTP = append(g.middleHTTP, v)
		}
	}
}

// Group is to make custome group middleware
func (g *group) Group(pattern string, h ...handlerFunc) *group {
	newGroup := &group{
		mux: g.mux,
	}

	if pattern != "" && strings.HasPrefix(pattern, "/") {
		newGroup.prefix = pattern
		for _, v := range h {
			newGroup.middleware = append(newGroup.middleware, v)
		}
	} else {
		err := errors.New("Url pattern can't be empty and has to start with / (slash)!")
		logger.Error(err, "Group error")
	}

	return newGroup
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
func (g *group) ServeFiles(urlPath string, dirPath string, prefix string) {
	g.Handle(urlPath, Gzip(http.StripPrefix(prefix, http.FileServer(http.Dir(dirPath)))))
}

// ServeFavicon will serve the favicon you choose
func (g *group) ServeFavicon(filePath string) {
	g.HandlerFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filePath)
	})
}

// JSON for json handling
func (c Context) JSON(data interface{}) {
	Data, err := json.Marshal(data)
	if err != nil {
		logger.Error(err, "JSON Marshal error")
	}

	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.Write(Data)
}

func (c Context) Write(str string) {
	c.ResponseWriter.Write([]byte(str))
}

// Listen will start the server (http.ListenAndServe)
func (g *group) Listen(serve string) error {
	// listening @ :PORT
	logger.Info("listening @" + serve)

	// start listening
	return http.ListenAndServe(serve, g)
}
