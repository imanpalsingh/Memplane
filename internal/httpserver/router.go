package httpserver

import (
	"errors"
	"fmt"
	"net/http"

	"memplane/internal/memory"

	"github.com/gin-gonic/gin"
)

func NewRouter(environment string, store *memory.Store) (*gin.Engine, error) {
	if store == nil {
		return nil, errors.New("memory store is required")
	}

	EnableStrictJSONDecoding()
	gin.SetMode(ginMode(environment))

	router := gin.New()
	router.Use(gin.Recovery())
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("set trusted proxies: %w", err)
	}
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	eventsHandler := newEventsHandler(store)
	v1 := router.Group("/v1")
	v1.POST("/events", eventsHandler.create)
	v1.GET("/events", eventsHandler.list)
	v1.POST("/segment", eventsHandler.segment)
	v1.POST("/retrieve", eventsHandler.retrieve)

	return router, nil
}

func ginMode(environment string) string {
	switch environment {
	case "development":
		return gin.DebugMode
	case "test":
		return gin.TestMode
	default:
		return gin.ReleaseMode
	}
}
