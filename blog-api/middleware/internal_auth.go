package middleware

import (
	"awesomeProject1/setting"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const InternalTokenHeader = "X-Internal-Token"

// InternalAuthMiddleware authenticates calls made by trusted internal
// services. It fails closed when the server secret is not configured.
func InternalAuthMiddleware(c *gin.Context) {
	expected := strings.TrimSpace(setting.InternalToolSetting.Token)
	provided := strings.TrimSpace(c.GetHeader(InternalTokenHeader))
	if expected == "" || provided == "" || !constantTimeEqual(expected, provided) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "unauthorized",
				"message": "invalid internal token",
			},
		})
		return
	}
	c.Next()
}

func constantTimeEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}
