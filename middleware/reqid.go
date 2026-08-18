// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

package middleware

import (
	"strings"

	ginlogger "github.com/LingByte/ling-base/logger/gin"
	"github.com/LingByte/ling-base/logger"
	"github.com/gin-gonic/gin"
)

// RequestIDMiddleware assigns X-Reqid on every HTTP request (reuse inbound header or generate).
// Must run before LoggerMiddleware and handlers so logger.*Ctx helpers see the id.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := strings.TrimSpace(c.GetHeader(logger.HeaderXReqID))
		if reqID == "" {
			reqID = logger.GenReqID()
		}
		c.Header(logger.HeaderXReqID, reqID)
		c.Set(logger.GinCtxReqIDKey, reqID)
		if c.Request != nil {
			c.Request = c.Request.WithContext(logger.WithRequestID(c.Request.Context(), reqID))
		}
		c.Next()
	}
}

// ReqIDFromGin is a convenience alias for ginlogger.ReqIDFromGin.
func ReqIDFromGin(c *gin.Context) string {
	return ginlogger.ReqIDFromGin(c)
}
