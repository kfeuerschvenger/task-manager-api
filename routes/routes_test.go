package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kfeuerschvenger/task-manager-api/middleware"
	"github.com/kfeuerschvenger/task-manager-api/utils"
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

func TestOversizedBodyIsRejectedWith413(t *testing.T) {
	const limit = 1 << 20
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	token, err := utils.GenerateJWT("00000000-0000-0000-0000-000000000001")
	assert.NoError(t, err)

	handler := middleware.LimitBody(limit, SetupRoutes())
	// Valid JSON prefix followed by more than limit bytes, so the decoder hits the cap before finishing.
	oversized := `{"title":"` + strings.Repeat("a", limit+1) + `"}`
	within := `{"title":"x"`

	tests := []struct {
		name   string
		method string
		path   string
		auth   bool
	}{
		{"register", http.MethodPost, "/auth/register", false},
		{"login", http.MethodPost, "/auth/login", false},
		{"create task", http.MethodPost, "/tasks", true},
		{"update task", http.MethodPut, "/tasks/00000000-0000-0000-0000-000000000002", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(oversized))
			if tt.auth {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
			assert.Contains(t, rec.Body.String(), "Request body too large")

			// A malformed body under the limit must stay a 400, not a 413.
			req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(within))
			if tt.auth {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			rec = httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestSwaggerUIIsServed(t *testing.T) {
	router := SetupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/documentation/index.html", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}
