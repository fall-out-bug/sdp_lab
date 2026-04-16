package testdata

func echoRoutes() {
	e := echo.New()

	e.GET("/", homeHandler)
	e.POST("/login", loginHandler)
	e.PUT("/profile", updateProfile)
	e.DELETE("/account", deleteAccount)

	g := e.Group("/api")
	g.GET("/status", statusHandler)
	g.POST("/webhook", webhookHandler)
}

func homeHandler()     {}
func loginHandler()    {}
func updateProfile()   {}
func deleteAccount()   {}
func statusHandler()   {}
func webhookHandler()  {}
