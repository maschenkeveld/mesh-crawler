package errors

// Import the standard errors package to create error values.
import "errors"

// Define common error variables for use throughout the mesh-crawler application.
// These provide consistent error messages for missing environment variables, failed requests, etc.
var (
	ErrNoNameEnvSet     = errors.New("no environment name set")     // Used when SERVICE_NAME env var is missing
	ErrNoHostnameEnvSet = errors.New("no environment hostname set") // Used when HOSTNAME env var is missing
	ErrNoVersionEnvSet  = errors.New("no environment version set")  // Used when SERVICE_VERSION env var is missing
	ErrNoZoneEnvSet     = errors.New("no environment zone set")     // Used when MESH_ZONE env var is missing
	ErrNoRootElement    = errors.New("no root element found")       // Used when a required root element is missing
	ErrNameMissMatch    = errors.New("names do not match")          // Used when names do not match in payloads
	ErrNoNextHop        = errors.New("no hop left")                 // Used when there are no more upstreams to crawl
	ErrRequestFailed    = errors.New("request failed")              // Used for generic request failures
)
