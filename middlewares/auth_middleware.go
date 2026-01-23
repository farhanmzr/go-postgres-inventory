package middlewares

import (
	"net/http"
	"strings"

	"go-postgres-inventory/utils"

	"github.com/gin-gonic/gin"
)

func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token admin diperlukan"})
			c.Abort()
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		claims, err := utils.VerifyAdminToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		c.Set("admin_id", claims.AdminID)
		c.Set("user_id", claims.AdminID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func UserAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token user diperlukan"})
			c.Abort()
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		claims, err := utils.VerifyUserToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("perms", claims.Permissions)
		c.Next()
	}
}

// RequirePerm digunakan di rute User App (BUKAN admin)
func RequirePerm(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := c.Get("perms")
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hak akses tidak ditemukan"})
			c.Abort()
			return
		}
		perms := raw.([]string)
		for _, p := range perms {
			if p == code {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak memiliki permission: " + code})
		c.Abort()
	}
}
