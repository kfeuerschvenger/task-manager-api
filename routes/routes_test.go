package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSwaggerDocJSONIsServed(t *testing.T) {
	router := SetupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/documentation/doc.json", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &spec))
	assert.Contains(t, resp.Body.String(), `"/tasks/{id}"`)
	assert.Contains(t, spec.Paths["/tasks/{id}"], "get")
}

func TestSwaggerUIIsServed(t *testing.T) {
	router := SetupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/documentation/index.html", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}
