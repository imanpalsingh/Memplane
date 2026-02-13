package httpserver

import (
	"sync"

	"github.com/gin-gonic/gin"
)

var strictJSONDecodingOnce sync.Once

func EnableStrictJSONDecoding() {
	strictJSONDecodingOnce.Do(func() {
		gin.EnableJsonDecoderDisallowUnknownFields()
	})
}
