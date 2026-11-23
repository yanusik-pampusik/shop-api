package app

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "shopapi/docs"
	"shopapi/internal/config"
	"shopapi/internal/handler"
	"shopapi/internal/middleware"
	authpb "shopapi/proto/auth"
)

func Router(cfg *config.Config, auth authpb.AuthServiceClient, authHandler *handler.AuthHandler,
	clientHandler *handler.ClientHandler, supplierHandler *handler.SupplierHandler,
	productHandler *handler.ProductHandler, imageHandler *handler.ImageHandler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	// Connections come directly from clients unless a trusted proxy is explicitly configured.
	_ = router.SetTrustedProxies(nil)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.POST("/register", authHandler.Register)
	router.POST("/auth", authHandler.Login)
	router.POST("/reset", authHandler.Reset)
	router.POST("/change-password", middleware.RequireAuth(auth, cfg.AuthTimeout), authHandler.ChangePassword)
	// Группируем API под /api/v1
	api := router.Group("/api/v1", middleware.RequireAuth(auth, cfg.AuthTimeout))
	{
		// Clients
		clients := api.Group("/clients")
		{
			clients.POST("", clientHandler.CreateClient)
			clients.GET("", clientHandler.GetClients)
			clients.GET("/search", clientHandler.GetClientsByName)
			clients.GET("/:id", clientHandler.GetClientByID)
			clients.PATCH("/:id/address", clientHandler.UpdateClientAddress)
			clients.DELETE("/:id", clientHandler.DeleteClient)
		}

		// Suppliers
		suppliers := api.Group("/suppliers")
		{
			suppliers.POST("", supplierHandler.CreateSupplier)
			suppliers.GET("", supplierHandler.GetSuppliers)
			suppliers.GET("/:id", supplierHandler.GetSupplierByID)
			suppliers.PATCH("/:id/address", supplierHandler.UpdateSupplierAddress)
			suppliers.DELETE("/:id", supplierHandler.DeleteSupplier)
		}

		// Products
		products := api.Group("/products")
		{
			products.POST("", productHandler.CreateProduct)
			products.GET("", productHandler.GetProducts)
			products.GET("/:id", productHandler.GetProductByID)
			products.PATCH("/:id/decrease", productHandler.DecreaseStock)
			products.DELETE("/:id", productHandler.DeleteProduct)
		}

		// Images
		images := api.Group("/images")
		{
			images.POST("", imageHandler.UploadImage)
			images.GET("/:id", imageHandler.GetImageByID)
			images.GET("/product/:product_id", imageHandler.GetImageByProductID)
			images.PUT("/:id", imageHandler.UpdateImage)
			images.DELETE("/:id", imageHandler.DeleteImage)
		}
	}

	return router
}
