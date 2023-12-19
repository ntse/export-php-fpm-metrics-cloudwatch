package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetrieveInstanceId(t *testing.T) {
	// Test with a valid METADATA_LINK_LOCAL_ADDRESS environment variable set (should return test-instance-id)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test-instance-id"))
	}))
	defer server.Close()

	// Replace the infoEndpoint with the test server URL
	os.Setenv("METADATA_LINK_LOCAL_ADDRESS", server.URL)

	// Call the retrieveInstanceId function
	instanceId := retrieveInstanceId()

	// Verify the result
	expectedInstanceId := "test-instance-id"
	assert.Equal(t, expectedInstanceId, instanceId, "Expected instance ID: %s, but got: %s", expectedInstanceId, instanceId)
}

func TestRetrieveInstanceIdNoServer(t *testing.T) {
	// Test with no METADATA_LINK_LOCAL_ADDRESS environment variable set (should return NOT_SET)
	os.Unsetenv("METADATA_LINK_LOCAL_ADDRESS")
	instanceId := retrieveInstanceId()
	assert.Equal(t, "NOT_SET", instanceId, "Expected instance ID: NOT_SET, but got: %s", instanceId)
}

func TestRetrieveInstanceIdBadServer(t *testing.T) {
	// Test with a bad METADATA_LINK_LOCAL_ADDRESS environment variable set (should return a 500 error)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	os.Setenv("METADATA_LINK_LOCAL_ADDRESS", server.URL)
	instanceId := retrieveInstanceId()
	assert.Equal(t, "NOT_SET", instanceId, "Expected instance ID: NOT_SET, but got: %s", instanceId)
}
