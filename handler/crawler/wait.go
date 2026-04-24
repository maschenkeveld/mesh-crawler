// This file implements the /wait endpoint handler.
// It waits for a specified time (from header), then returns environment info and headers.

package handler

import (
	"fmt"                      // For printing to stdout
	"mesh-crawler/core/errors" // Custom error definitions
	"net/http"                 // HTTP status codes
	"os"                       // For reading environment variables
	"strconv"                  // For converting string to int
	"time"                     // For sleeping

	"github.com/labstack/echo/v4" // Echo web framework
)

// Wait handles GET /wait requests.
// It optionally sleeps for a number of seconds specified in the "Sleep-Time" header.
// Returns service info and incoming headers in the response.
func (m *module) Wait(context echo.Context) error {
	// Create the response struct and initialize the headers map
	waitResponse := &WaitResponse{
		IncomingHeaders: map[string]string{},
	}

	// Retrieve and validate environment variables
	name, found := os.LookupEnv("SERVICE_NAME")
	if !found {
		waitResponse.Reason = errors.ErrNoNameEnvSet.Error()
		waitResponse.StatusCode = http.StatusInternalServerError
		return echo.NewHTTPError(waitResponse.StatusCode, waitResponse)
	}

	hostname, found := os.LookupEnv("SERVICE_HOSTNAME")
	if !found {
		waitResponse.Reason = errors.ErrNoNameEnvSet.Error()
		waitResponse.StatusCode = http.StatusInternalServerError
		return echo.NewHTTPError(waitResponse.StatusCode, waitResponse)
	}

	zone, found := os.LookupEnv("MESH_ZONE")
	if !found {
		waitResponse.Reason = errors.ErrNoZoneEnvSet.Error()
		waitResponse.StatusCode = http.StatusInternalServerError
		return echo.NewHTTPError(waitResponse.StatusCode, waitResponse)
	}

	// Populate response fields
	waitResponse.Name = name
	waitResponse.Hostname = hostname
	waitResponse.Zone = zone

	// Capture incoming headers
	headers := context.Request().Header
	for key, values := range headers {
		for _, value := range values {
			waitResponse.IncomingHeaders[key] = value
		}
	}

	// Check for "Sleep-Time" header and parse it as seconds
	sleepTimeStr := context.Request().Header.Get("Sleep-Time")
	if sleepTimeStr != "" {
		sleepTimeSeconds, err := strconv.Atoi(sleepTimeStr)
		if err != nil || sleepTimeSeconds < 0 {
			waitResponse.Reason = "Invalid Sleep-Time header value"
			waitResponse.StatusCode = http.StatusBadRequest
			return echo.NewHTTPError(waitResponse.StatusCode, waitResponse)
		}

		// Print to stdout and sleep for the requested time
		fmt.Println("Starting sleep for", sleepTimeSeconds, "seconds...")
		time.Sleep(time.Duration(sleepTimeSeconds) * time.Second)
		fmt.Println("Finished sleeping!")
	}

	// Return response as JSON
	return context.JSON(http.StatusOK, waitResponse)
}
