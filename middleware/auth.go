package middleware

import (
	"net/http"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// RequireAuth rejects requests without a valid JWT cookie. Every mutating
// route needs it: article, category and tag writes were previously open to
// anyone who could reach the API.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("jwt")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid token"})
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return config.JWTSecret(), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		if id, ok := claims["user_id"].(float64); ok {
			c.Set("user_id", uint(id))
		}
		if name, ok := claims["username"].(string); ok {
			c.Set("username", name)
		}
		c.Next()
	}
}
