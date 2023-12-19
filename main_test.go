package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetrieveServiceName(t *testing.T) {
	// Test with a valid ECS_CONTAINER_METADATA_URI_V4 environment variable set (should return test-instance-id)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test-instance-id"))
	}))
	defer server.Close()

	// Replace the infoEndpoint with the test server URL
	os.Setenv("ECS_CONTAINER_METADATA_URI_V4", server.URL)

	// Call the retrieveInstanceId function
	instanceId := retrieveServiceName()

	// Verify the result
	expectedInstanceId := "test-instance-id"
	assert.Equal(t, expectedInstanceId, instanceId, "Expected instance ID: %s, but got: %s", expectedInstanceId, instanceId)
}

func TestRetrieveServiceNameNoServer(t *testing.T) {
	// Test with no ECS_CONTAINER_METADATA_URI_V4 environment variable set (should return NOT_SET)
	os.Unsetenv("ECS_CONTAINER_METADATA_URI_V4")
	instanceId := retrieveServiceName()
	assert.Equal(t, "NOT_SET", instanceId, "Expected instance ID: NOT_SET, but got: %s", instanceId)
}

func TestRetrieveServiceNameBadServer(t *testing.T) {
	// Test with a bad ECS_CONTAINER_METADATA_URI_V4 environment variable set (should return a 500 error)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	os.Setenv("ECS_CONTAINER_METADATA_URI_V4", server.URL)
	instanceId := retrieveServiceName()
	assert.Equal(t, "NOT_SET", instanceId, "Expected instance ID: NOT_SET, but got: %s", instanceId)
}
