# Router

## This package is for learning purposes.

### Example

```go
// Create New multiplexor / router
r := router.New()

// You can add as many middlewares as you like,
// they will load in the order they are added.

// Handle Global Middleware for every requests
r.GlobalMiddleware(gMiddleware1, gMiddleware2)

// Handle Group Middleware for the specific request
r.GroupMiddleware("/admin", aMiddleware1, aMiddleware2)

// GET requests
r.GET("/", indexHandler)
r.GET("/admin", adminHandler)

// POST requests
r.POST("/register", registerHandler)
r.POST("/login", loginHandler)
r.POST("/logout". logoutHandler)

// Serve Static Files (Gzip and cache)
r.ServeFiles("urlPath", "dirPath", "prefix")
// Serve Favicon
r.ServeFavicon("pathToFavicon")

// ListenAndServe
r.Listen(":8000")
```
