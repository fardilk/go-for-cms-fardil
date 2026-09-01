package routes

import (
	"net/http"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/gin-gonic/gin"
)

// Health reports whether the process can reach the database, so the deploy
// smoke check has something meaningful to poll.
func Health(c *gin.Context) {
	sqlDB, err := config.DB.DB()
	if err != nil || sqlDB.Ping() != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "database": "unreachable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "database": "ok"})
}
