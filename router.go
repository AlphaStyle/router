package router

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alphastyle/logger"
	uuid "github.com/satori/go.uuid"
)

// mux is the multiplexer struct
type mux struct {
	*http.ServeMux
	middle []handlerFunc
}

// Group is to divide request middleware
type Group struct {
	*mux
	middleware []handlerFunc
	prefix     string
}

// Context is a custom ResponseWriter and Request
type Context struct {
	http.ResponseWriter
	*http.Request
}

// handlerFunc is custom http.HandleFunc type
type handlerFunc func(*Context)

// Gzip Compression
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

// Gzip Write
func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// New will create a new group
func New() *Group {
	return &Group{
		mux: &mux{
			ServeMux: http.NewServeMux(),
		},
	}
}

func (h handlerFunc) ServeHTTP(c *Context) {
	h(c)
}

// GET is a custom http.HandlerFunc that only allow GET requests
func (g *Group) GET(pattern string, h handlerFunc) {
	handler := g.handleRequest(h, "GET")
	g.Handle(g.prefix+pattern, http.HandlerFunc(handler))
}

// POST is a custom http.HandlerFunc that only allow POST requests
func (g *Group) POST(pattern string, h handlerFunc) {
	handler := g.handleRequest(h, "POST")
	g.Handle(g.prefix+pattern, http.HandlerFunc(handler))
}

// handleRequest will check the request method and handle middleware
func (g *Group) handleRequest(h handlerFunc, method string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == method {
			// w.Header().Set("Content-Type", "text/HTML")
			mrw := &Context{w, r}
			g.handleMiddleware(mrw)
			h.ServeHTTP(mrw)
		} else {
			http.NotFound(w, r)
		}
	}
}

// handleMiddleware will serve the correct middleware for the request
func (g *Group) handleMiddleware(c *Context) {
	// Global Middleware
	for _, v := range g.middle {
		v.ServeHTTP(c)
	}

	// Group Middleware
	for _, v := range g.middleware {
		v.ServeHTTP(c)
	}
}

// Use is to make custom global middleware
// or group middleware
func (g *Group) Use(h ...handlerFunc) {
	if g.prefix == "" {
		// Global Middleware
		for _, v := range h {
			g.middle = append(g.middle, v)
		}
	} else {
		// Group Middleware
		for _, v := range h {
			g.middleware = append(g.middleware, v)
		}
	}
}

// Group makes it possible to have custom group middleware
func (g *Group) Group(pattern string, h ...handlerFunc) *Group {
	// Initialize new group
	newGroup := &Group{
		mux: g.mux,
	}

	if pattern != "" && strings.HasPrefix(pattern, "/") {
		newGroup.prefix = pattern
		// Appending middleware to the new group
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
func (g *Group) ServeFiles(urlPath string, dirPath string, prefix string) {
	g.Handle(urlPath, Gzip(http.StripPrefix(prefix, http.FileServer(http.Dir(dirPath)))))
}

// ServeFavicon will serve the favicon you choose
func (g *Group) ServeFavicon(filePath string) {
	g.GET("/favicon.ico", func(c *Context) {
		http.ServeFile(c.ResponseWriter, c.Request, filePath)
	})
}

// JSON for json handling
func (c *Context) JSON(data interface{}) {
	Data, err := json.Marshal(data)
	if err != nil {
		logger.Error(err, "JSON Marshal error")
	}

	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.Write(Data)
}

func (c *Context) Write(str string) {
	c.ResponseWriter.Write([]byte(str))
}

// NewContext creates and return the request with context
func (c *Context) NewContext(key, value interface{}) {
	ctx := c.Context()
	ctx = context.WithValue(ctx, key, value)
	c.Request = c.WithContext(ctx)
}

// GetContext will get the context from the specific request
func (c *Context) GetContext(key string) interface{} {
	val := c.Context().Value(key)
	return val
}

// Got this from Stackoverflow (Copy / Paste)
// Will create a random string with the length n
// func randomValue(n int, src rand.Source) string {
// 	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
// 	letterIdxBits := uint(6)              // 6 bits to represent a letter index
// 	letterIdxMask := 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
// 	letterIdxMax := 63 / letterIdxBits    // # of letter indices fitting in 63 bits

// 	b := make([]byte, n)
// 	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
// 	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
// 		if remain == 0 {
// 			cache, remain = src.Int63(), letterIdxMax
// 		}
// 		if idx := int(cache & int64(letterIdxMask)); idx < len(letterBytes) {
// 			b[i] = letterBytes[idx]
// 			i--
// 		}
// 		cache >>= letterIdxBits
// 		remain--
// 	}

// 	return string(b)
// }

func randomValue() uuid.UUID {
	return uuid.NewV4()
}

// NewSession will create a new cookie session
func (c *Context) NewSession(name string) {
	// value := randomValue(40, rand.NewSource(time.Now().UnixNano()))
	value := randomValue()
	expiration := time.Now().Add(30 * time.Minute) // TODO make time a config setting

	cookie := &http.Cookie{
		Name:    name,
		Value:   value.String(),
		Expires: expiration,
		Path:    "/",
	}

	http.SetCookie(c.ResponseWriter, cookie)
}

// DeleteSession will delete the cookie session
func (c *Context) DeleteSession(name string) {
	cookie := &http.Cookie{
		Name:    name,
		Value:   "deleted",
		Expires: time.Now(),
		MaxAge:  -1,
		Path:    "/",
	}

	http.SetCookie(c.ResponseWriter, cookie)
}

// GetSession will get the cookie session
func (c *Context) GetSession(name string) (*http.Cookie, error) {
	cookie, err := c.Cookie(name)
	return cookie, err
}

// Listen will start the server (http.ListenAndServe)
func (g *Group) Listen(serve string) error {
	// listening @ :PORT
	logger.Info("listening @" + serve)

	// start listening
	return http.ListenAndServe(serve, g)
}
