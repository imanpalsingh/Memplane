package httpserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(environment string) (*gin.Engine, error) {
	gin.SetMode(ginMode(environment))
	router := gin.New()
	router.Use(gin.Recovery())
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("set trusted proxies: %w", err)
	}
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
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
