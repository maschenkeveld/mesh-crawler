// This file defines the main data structures used for requests and responses in the mesh-crawler API.
// Each struct represents a different type of response or payload for the various endpoints.

package handler

// CrawlResponse is used for the /crawl endpoint and recursive upstream calls.
// It contains service info, headers, upstream responses, and status.
type CrawlResponse struct {
	StatusCode        int               `json:"statusCode"`         // HTTP status code for the response
	Name              string            `json:"name"`               // Service name
	Version           string            `json:"version"`            // Service version
	Hostname          string            `json:"hostname"`           // Hostname of the service
	Zone              string            `json:"zone"`               // Zone/environment
	Reason            string            `json:"reason,omitempty"`   // Reason for error or status
	IncomingHeaders   map[string]string `json:"incomingHeaders"`    // Headers received in the request
	UpstreamResponses []*CrawlResponse  `json:"upstream,omitempty"` // Responses from upstream services
	FullPath          string            `json:"fullPath,omitempty"` // Request path
}

// IdentifyResponse is used for the /identify endpoint.
type IdentifyResponse struct {
	StatusCode      int               `json:"statusCode"`         // HTTP status code
	Name            string            `json:"name"`               // Service name
	Version         string            `json:"version"`            // Service version
	Hostname        string            `json:"hostname"`           // Hostname
	Zone            string            `json:"zone"`               // Zone/environment
	Reason          string            `json:"reason,omitempty"`   // Reason for error or status
	IncomingHeaders map[string]string `json:"incomingHeaders"`    // Request headers
	FullPath        string            `json:"fullPath,omitempty"` // Request path
}

// HealthResponse is used for the /health endpoint.
type HealthResponse struct {
	Status string `json:"status"` // Health status string
}

// UnhealthResponse is used for unhealthy status responses.
type UnhealthResponse struct {
	Status string `json:"status"` // Health status string
}

// WaitResponse is used for the /wait endpoint.
type WaitResponse struct {
	StatusCode      int               `json:"statusCode"`
	Name            string            `json:"name"`
	Hostname        string            `json:"hostname"`
	Zone            string            `json:"zone"`
	Reason          string            `json:"reason,omitempty"`
	IncomingHeaders map[string]string `json:"incomingHeaders"`
}

// LogResponse is a generic response for logging endpoints.
type LogResponse struct {
	StatusCode      int               `json:"statusCode"`
	Name            string            `json:"name"`
	Hostname        string            `json:"hostname"`
	Zone            string            `json:"zone"`
	Reason          string            `json:"reason,omitempty"`
	IncomingHeaders map[string]string `json:"incomingHeaders"`
}

// LoadTestResponse is used for the /load-test endpoint.
type LoadTestResponse struct {
	StatusCode      int               `json:"statusCode"`       // HTTP status code
	Name            string            `json:"name"`             // Service name
	Hostname        string            `json:"hostname"`         // Hostname
	Zone            string            `json:"zone"`             // Zone/environment
	Reason          string            `json:"reason,omitempty"` // Reason for error or status
	CalculationTime string            `json:"calculationTime"`  // Time taken for load calculation
	IncomingHeaders map[string]string `json:"incomingHeaders"`  // Request headers
}

// Upstream represents an upstream service in the payload.
type Upstream struct {
	Host      string      `json:"host" yaml:"host"`           // Hostname of the upstream service
	Upstreams []*Upstream `json:"upstreams" yaml:"upstreams"` // Nested upstreams for recursive crawling
}

// Payload is used for incoming requests to /crawl and other endpoints.
type Payload struct {
	Upstreams []*Upstream `json:"upstreams" yaml:"upstreams"` // List of upstreams to crawl
}
