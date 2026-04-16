package testdata

func ginRoutes() {
	r := gin.Default()

	r.GET("/ping", pingHandler)
	r.POST("/items", createItem)
	r.PUT("/items/:id", updateItem)
	r.DELETE("/items/:id", deleteItem)

	api := r.Group("/api/v1")
	{
		api.GET("/health", healthCheck)
		api.POST("/deploy", deployHandler)
	}

	// Nested groups: api.Group("/v2") should compose to /api/v1/v2
	v2 := api.Group("/v2")
	v2.GET("/status", v2StatusHandler)
}

func pingHandler()    {}
func createItem()     {}
func updateItem()     {}
func deleteItem()     {}
func healthCheck()    {}
func deployHandler()  {}
func v2StatusHandler() {}
