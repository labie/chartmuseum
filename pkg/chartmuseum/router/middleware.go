package router

import (
	"fmt"
	pathutil "path"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"

	"github.com/gin-gonic/gin"
)

// this is needed due to issues with Gin handling wildcard routes:
// https://github.com/gin-gonic/gin/issues/388
func prefixPathMiddleware(engine *gin.Engine, pathPrefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqPath := c.Request.URL.Path
		if strings.HasPrefix(reqPath, pathPrefix) {
			return
		}
		c.Request.URL.Path = pathutil.Join(pathPrefix, reqPath)
		engine.HandleContext(c)
	}
}

func loggingMiddleware(logger *cm_logger.Logger, ignorePrefix string) gin.HandlerFunc {
	var requestCount int64
	return func(c *gin.Context) {
		reqPath := c.Request.URL.Path
		if ignorePrefix != "" && strings.HasPrefix(reqPath, ignorePrefix) {
			return
		}

		reqCount := strconv.FormatInt(atomic.AddInt64(&requestCount, 1), 10)
		c.Set("RequestCount", reqCount)

		logger.Debugc(c, fmt.Sprintf("Incoming request: %s", reqPath))
		start := time.Now()
		c.Next()

		msg := "Request served"
		status := c.Writer.Status()

		meta := []interface{}{
			"path", reqPath,
			"comment", c.Errors.ByType(gin.ErrorTypePrivate).String(),
			"latency", time.Now().Sub(start),
			"clientIP", c.ClientIP(),
			"method", c.Request.Method,
			"statusCode", status,
		}

		switch {
		case status == 200 || status == 201:
			logger.Infoc(c, msg, meta...)
		case status == 404:
			logger.Warnc(c, msg, meta...)
		default:
			logger.Errorc(c, msg, meta...)
		}
	}
}
