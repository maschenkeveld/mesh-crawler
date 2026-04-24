// This file implements the /health endpoint handler.
// It checks the HEALTH environment variable and returns service health status.

package handler

import (
	"log"      // For logging health check failures
	"net/http" // For HTTP status codes
	"os"       // For reading environment variables

	"github.com/labstack/echo/v4" // Echo web framework
)

// Health handles GET /health requests.
// It returns "ready" if the HEALTH env var is not "unhealthy", otherwise "not ready".
func (m *module) Health(context echo.Context) error {
	// Read the HEALTH environment variable
	health, found := os.LookupEnv("HEALTH")

	// If HEALTH is set to "unhealthy", return a 500 status and log the failure
	if found && health == "unhealthy" {
		log.Printf("Health check failed: %s", health) // Log unhealthy status
		healthResponse := &HealthResponse{
			Status: "not ready",
		}
		return context.JSON(http.StatusInternalServerError, healthResponse)
	}

	// Otherwise, return "ready" with a 200 status
	healthResponse := &HealthResponse{
		Status: "ready",
	}
	return context.JSON(http.StatusOK, healthResponse)
}
