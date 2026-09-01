package routes

import (
	"net/http"
	"strings"
	"time"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Dashboard is a protected endpoint that returns user info if JWT is valid
func Dashboard(c *gin.Context) {
	tokenString, err := c.Cookie("jwt")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid token"})
		return
	}

	secret := config.JWTSecret()
	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	userID, ok := claims["user_id"].(float64)
	username, ok2 := claims["username"].(string)
	if !ok || !ok2 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to the dashboard!",
		"user": gin.H{
			"id":       int(userID),
			"username": username,
		},
	})
}

// LoginRequest struct untuk binding JSON request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// POST /login
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var user models.User
	result := config.DB.Where("username = ?", req.Username).First(&user)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if !checkPassword(&user, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, err := generateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Set JWT as HTTP-only cookie
	// Secure in production. SameSite defaults to Lax because the panel is served
	// from the same origin as the API, under /admin on the marketing domain.
	c.SetSameSite(config.CookieSameSite())
	c.SetCookie("jwt", token, 3600*24, "/", config.CookieDomain(), config.IsProduction(), true)

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
	// Example for protected endpoints:
	// token, err := c.Cookie("jwt")
	// Then validate the token as usual
}

// generateJWT creates a JWT token for the authenticated user
func generateJWT(user models.User) (string, error) {
	secret := config.JWTSecret()
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// Me handles the /me route
func Me(c *gin.Context) {
	tokenString, err := c.Cookie("jwt")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid token"})
		return
	}

	secret := config.JWTSecret()
	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	userID, ok := claims["user_id"].(float64)
	username, ok2 := claims["username"].(string)
	if !ok || !ok2 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       int(userID),
			"username": username,
		},
	})
}

// checkPassword verifies req against the stored value. Passwords used to be
// stored and compared in plain text; anything that is not a bcrypt hash is
// treated as a legacy plaintext record, verified once and then rehashed in
// place so no existing account is locked out.
//
// TODO: drop the legacy branch once every row has been migrated.
func checkPassword(user *models.User, candidate string) bool {
	if strings.HasPrefix(user.Password, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(candidate)) == nil
	}
	if user.Password != candidate {
		return false
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(candidate), bcrypt.DefaultCost)
	if err != nil {
		return true
	}
	config.DB.Model(user).Update("password", string(hashed))
	return true
}
