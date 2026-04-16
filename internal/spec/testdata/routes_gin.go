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
}

func pingHandler()    {}
func createItem()     {}
func updateItem()     {}
func deleteItem()     {}
func healthCheck()    {}
func deployHandler()  {}
