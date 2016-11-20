# Router

## This package is for learning purposes.

### Example

```go
// GET Example Handler
func indexHandler(c router.Context) {
  c.Write("This is index Page")
}

// POST Example Handler
func registerHandler(c router.Context) {
  c.Write("This is Register Page")
  c.JSON(structData)
}

// HandlerFunc Example
func aboutHandler(w http.ResponseWriter, r *http.Request) {
  w.Write([]byte("This is About Page"))
}

// Middleware Example
func gMiddleware1(c router.Context) {
  fmt.Println(c.URL.Path)
  fmt.Println("This is Global Middleware1")
}

// Create New multiplexor / router
r := router.New()

// You can add as many middlewares as you like,
// they will load in the order they are added.

// Handle Global Middleware for every requests
r.Use(gMiddleware1, gMiddleware2)

// Handle Group Middleware for the specific request
r.Group("/admin", aMiddleware1, aMiddleware2)

// GET requests
r.GET("/", indexHandler)
r.GET("/admin", adminHandler)

// POST requests
r.POST("/register", registerHandler)
r.POST("/login", loginHandler)
r.POST("/logout". logoutHandler)

// HandlerFunc request (like http.HandlerFunc)
r.HandlerFunc("/" aboutHandler)

// Serve Static Files (Gzip and cache)
r.ServeFiles("urlPath", "dirPath", "prefix")

// Serve Favicon
r.ServeFavicon("Relative/Path/To/Favicon")

// ListenAndServe
r.Listen(":8000")
```
