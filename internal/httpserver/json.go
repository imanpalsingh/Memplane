package httpserver

import "github.com/gin-gonic/gin"

func EnableStrictJSONDecoding() {
	gin.EnableJsonDecoderDisallowUnknownFields()
}
