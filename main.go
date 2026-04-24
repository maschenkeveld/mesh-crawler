package main

// Import the routers package, which contains the HTTP server and route setup.
import "mesh-crawler/routers"

// main is the entry point of the Go application.
// When you run `go run .` or build and execute the binary, this function is called first.
func main() {
	// Start the API server by calling routers.Api().
	// This sets up all HTTP routes and starts listening for requests.
	routers.Api()
}
