package management

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthCheckResponse represents the health status of the server and services
type HealthCheckResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Runtime   RuntimeInfo       `json:"runtime"`
}

// RuntimeInfo contains runtime statistics
type RuntimeInfo struct {
	Goroutines int    `json:"goroutines"`
	Memory     string `json:"memory"`
	Version    string `json:"version"`
}

// HealthHandler handles GET /v1/health
func HealthHandler(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Format memory in human-readable format
	memoryMB := float64(m.Alloc) / 1024 / 1024

	response := HealthCheckResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Services: map[string]string{
			"router": "running",
		},
		Runtime: RuntimeInfo{
			Goroutines: runtime.NumGoroutine(),
			Memory:     formatBytes(m.Alloc),
			Version:    "1.0.0", // TODO: Get version from build flags
		},
	}

	// Check if services are healthy (would be extended in real implementation)
	// For now, just return the router status

	c.JSON(http.StatusOK, response)
}

// DeepHealthHandler handles GET /v1/health/deep (more detailed check)
func DeepHealthHandler(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"checks": map[string]interface{}{
			"router": map[string]interface{}{
				"status": "ok",
				"details": map[string]interface{}{
					"goroutines": runtime.NumGoroutine(),
					"memory":     formatBytes(m.Alloc),
					"heap":       formatBytes(m.HeapAlloc),
					"sys":        formatBytes(m.Sys),
				},
			},
			"config": map[string]interface{}{
				"status": "ok",
			},
			"auth": map[string]interface{}{
				"status": "ok",
			},
		},
	}

	c.JSON(http.StatusOK, response)
}

// formatBytes converts bytes to a human-readable string
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	sizes := []string{"B", "KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.2f %s", float64(b)/float64(div), sizes[exp])
}
