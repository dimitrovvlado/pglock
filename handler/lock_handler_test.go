package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var endpoint = "http://localhost:8080/v1/lock"

func TestLockingCases(t *testing.T) {
	handler := testHandler(t)

	tests := []tableTest{
		{
			description: "The same lock for the same device ID is always successful",
			steps: []step{
				{method: "POST", json: `{"profileId":"1", "deviceId":"1"}`, expectedStatus: 200},
				{method: "POST", json: `{"profileId":"1", "deviceId":"1"}`, expectedStatus: 200},
				{method: "POST", json: `{"profileId":"1", "deviceId":"1"}`, expectedStatus: 200},
			},
		},
		{
			description: "Locking multiple devices returns 400 until unlocked",
			steps: []step{
				{method: "POST", json: `{"profileId":"2", "deviceId":"2"}`, expectedStatus: 200},
				{method: "POST", json: `{"profileId":"2", "deviceId":"3"}`, expectedStatus: 400},
				{method: "DELETE", json: `{"profileId":"2"}`, expectedStatus: 200},
				{method: "POST", json: `{"profileId":"2", "deviceId":"3"}`, expectedStatus: 200},
			},
		},
		{
			description: "Unlock a non-existing lock results in 404",
			steps: []step{
				{method: "DELETE", json: `{"profileId":"0"}`, expectedStatus: 404},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			for _, step := range test.steps {
				assertHTTPStatusCode(t, handler, step.method, endpoint, step.expectedStatus, strings.NewReader(step.json))
			}
		})
	}
}

func TestLockExpiry(t *testing.T) {
	handler := testHandler(t)
	assertHTTPStatusCode(t, handler, "POST", endpoint, 200, strings.NewReader(`{"profileId":"10", "deviceId":"10"}`))
	assertHTTPStatusCode(t, handler, "POST", endpoint, 400, strings.NewReader(`{"profileId":"10", "deviceId":"11"}`))
	//We're setting the default TTL to 5 seconds
	time.Sleep(8 * time.Second)
	assertHTTPStatusCode(t, handler, "POST", endpoint, 200, strings.NewReader(`{"profileId":"10", "deviceId":"11"}`))
}

type tableTest struct {
	description string
	steps       []step
}

type step struct {
	method         string
	json           string
	expectedStatus int
}

func assertHTTPStatusCode(t *testing.T, handler http.HandlerFunc, method, url string, statuscode int, body io.Reader) bool {
	// code, err := httpCode(handler, method, url, body)
	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Errorf("Failed to build test request, got error: %s", err)
		t.Fail()
	}
	handler(w, req)
	successful := w.Code == statuscode
	if !successful {
		t.Errorf(fmt.Sprintf("Expected HTTP status code %d but received %d", statuscode, w.Code))
		t.Fail()
	}

	return successful
}

func httpCode(handler http.HandlerFunc, method, url string, body io.Reader) (int, error) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return -1, err
	}
	handler(w, req)
	return w.Code, nil
}

// testHandler is a glue function for the assert library
// it's goal is to execute a request and write the response into the http.ResponseWriter
func testHandler(t *testing.T) http.HandlerFunc {
	client := &http.Client{Timeout: time.Duration(2 * time.Second)}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.Do(r)
		if err != nil {
			t.Errorf("Failed to connect to server: %v", err)
			t.FailNow()
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		resp.Body.Close()
	})
}
