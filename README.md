# Router (mini framework)

## This package is for learning purposes only

### Example

``` go
// GET Example Handler
func indexHandler(c router.Context) {
	c.Write("This is index Page")
}

// POST Example Handler
func registerHandler(c router.Context) {
	c.Write("This is Register Page")
	// WIll post JSON
	c.JSON(structData)
}

// Middleware Example 1
func gMiddleware1(c router.Context) {
	fmt.Println(c.URL.Path)
	fmt.Println("This is Global Middleware1")

	// New Request Context
	c.NewContext("key", "value")

	// Create a cookie session if cookie with name "key" does not exist
	if _, err := c.GetSession("key"); err != nil {
		c.NewSession("key")
	}
}

// Middleware Example 2
func gMiddleware2(c router.Context) {
	// Get a Request Context
	ctxVal := c.GetContext("key")
	fmt.Println(ctxVal)

    // Get a cookie Session
    cookie, err := c.GetSession("ket")
    if err != nil {
    	// handle error
    }
    fmt.Println(cookie.Value) // print cookie value

    // Use c.DeleteSession to delete a cookie
    c.DeleteSession("key")
}

// Create New multiplexor / router
r := router.New()

// You can add as many middlewares as you like,
// they will load in the order they are added.

// Handle Global Middleware for every requests
r.Use(gMiddleware1, gMiddleware2)

// Handle Group Middleware for the specific request
admin := r.Group("/admin", aMiddleware1, aMiddleware2)

//You can also add Middleware by using the Use method to a group
admin.Use(aMiddleware1)

// GET Request With Admin Middleware (localhost:8000/admin/index)
admin.GET("/index", adminHandler)

// GET requests (localhost:8000/)
r.GET("/", indexHandler)

// POST requests (localhost:8000/register | /login | /logout)
r.POST("/register", registerHandler)
r.POST("/login", loginHandler)
r.POST("/logout". logoutHandler)

// Serve Static Files such as css, js (with Gzip and cache)
r.ServeFiles("urlPath", "dirPath", "prefix")

// Serve Favicon
r.ServeFavicon("Relative/Path/To/Favicon")

// ListenAndServe
r.Listen(":8000")
```
