package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"data-platform/internal/config"
	assetH "data-platform/internal/handlers/asset"
	deliveryH "data-platform/internal/handlers/delivery"
	mcapH "data-platform/internal/handlers/mcap"
	"data-platform/internal/middleware"
)

// RegisterAll wires up all API domains in a single process.
// This is the default local/prod runtime mode for the current project.
func RegisterAll(
	r *gin.Engine,
	cfg *config.Config,
	assetHandler *assetH.Handler,
	mcapHandler *mcapH.Handler,
	deliveryHandler *deliveryH.Handler,
) {
	r.Use(middleware.RequestID())
	r.GET("/healthz", healthz("backend"))

	auth := middleware.StaticTokenAuth(cfg.GraceToken)

	api := r.Group("/api/v1", auth)
	{
		assets := api.Group("/assets")
		assets.GET("", assetHandler.List)
		assets.POST("", assetHandler.Create)
		assets.GET("/:id", assetHandler.Get)
		assets.PATCH("/:id", assetHandler.Update)
		assets.DELETE("/:id", assetHandler.Delete)
		assets.GET("/:id/deliveries", assetHandler.ListDeliveries)

		api.POST("/mcap/upload/finalize", mcapHandler.FinalizeUpload)
		api.GET("/mcap/:id/messages", mcapHandler.IterMessages)

		mcapFiles := api.Group("/mcap-files")
		mcapFiles.GET("", mcapHandler.ListFiles)
		mcapFiles.GET("/:id", mcapHandler.GetFile)

		api.POST("/deliveries", deliveryHandler.Commit)
		api.GET("/deliveries/:id", deliveryHandler.Get)
		api.GET("/customers/:customer_id/deliveries", deliveryHandler.ListByCustomer)
	}

	// Internal (service-to-service, no external auth required in Phase 0)
	r.POST("/internal/commit-segments", assetHandler.CommitSegments)
}

func healthz(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": service})
	}
}
