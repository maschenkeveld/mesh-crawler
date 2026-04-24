package handler

import (
	"crypto/rand"
	"log"
	"math"
	"mesh-crawler/core/errors"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

func generateCPULoad(complexity int) string {
	start := time.Now()

	log.Printf("Going to generate CPU load")

	iterationComplexity := complexity
	subiterationComplexity := complexity

	for i := 0; i < iterationComplexity; i++ {
		result := 0.0
		for j := 0; j < subiterationComplexity; j++ {
			result += math.Pow(float64(j), float64(i))
		}
		log.Printf("Iteration %d", i)
	}

	elapsed := time.Since(start)
	return elapsed.String()
}

func generateMemoryLoad() {
	var memory [][]byte
	for {
		// Generate memory load by allocating memory
		for i := 0; i < 1000; i++ {
			// Allocate 10MB of memory
			mem := make([]byte, 10*1024*1024)
			// Fill memory with random data
			rand.Read(mem)
			memory = append(memory, mem)
		}
		// Sleep for a short duration to control the rate of memory load generation
		time.Sleep(500 * time.Millisecond)
		// Clear memory to avoid memory leak
		memory = nil
		// Run garbage collection to free up memory
		runtime.GC()
	}
}

func (m *module) LoadTest(context echo.Context) error {
	loadTestResponse := &LoadTestResponse{
		IncomingHeaders: map[string]string{},
	}

	name, found := os.LookupEnv("SERVICE_NAME")
	if !found {
		loadTestResponse.Reason = errors.ErrNoNameEnvSet.Error()
		loadTestResponse.StatusCode = http.StatusInternalServerError
		return echo.NewHTTPError(loadTestResponse.StatusCode, loadTestResponse)
	}

	hostname, found := os.LookupEnv("SERVICE_HOSTNAME")
	if !found {
		loadTestResponse.Reason = errors.ErrNoNameEnvSet.Error()
		loadTestResponse.StatusCode = http.StatusInternalServerError
		return echo.NewHTTPError(loadTestResponse.StatusCode, loadTestResponse)
	}

	zone, found := os.LookupEnv("MESH_ZONE")
	if !found {
		loadTestResponse.Reason = errors.ErrNoZoneEnvSet.Error()
		loadTestResponse.StatusCode = http.StatusInternalServerError
		return echo.NewHTTPError(loadTestResponse.StatusCode, loadTestResponse)
	}

	complexityQueryParam := context.QueryParam("c")

	complexity, err := strconv.Atoi(complexityQueryParam)
	if err != nil {
		return context.String(http.StatusBadRequest, "Invalid parameter for complexity (c)")
	}

	loadTestResponse.Name = name
	loadTestResponse.Hostname = hostname
	loadTestResponse.Zone = zone
	loadTestResponse.CalculationTime = generateCPULoad(complexity) // This is a blocking call

	headers := context.Request().Header
	for key, values := range headers {
		for _, value := range values {
			loadTestResponse.IncomingHeaders[key] = value
		}
	}

	return context.JSON(loadTestResponse.StatusCode, loadTestResponse)

}
