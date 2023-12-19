package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createMockServer() *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{"ServiceName": "test-service-name"}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})

	return httptest.NewServer(handler)
}

func TestRetrieveServiceName(t *testing.T) {
	server := createMockServer()
	defer server.Close()

	os.Setenv("ECS_CONTAINER_METADATA_URI_V4", server.URL)

	serviceName := retrieveServiceName()

	expectedServiceName := "test-service-name"
	assert.Equal(t, expectedServiceName, serviceName, "Expected service name: %s, but got: %s", expectedServiceName, serviceName)
}
