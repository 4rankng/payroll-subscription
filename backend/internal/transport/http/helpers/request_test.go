package helpers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestContext(method, url string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}

	c.Request, _ = http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	return c, w
}

func TestBindJSON_Failure(t *testing.T) {
	type TestRequest struct {
		Name string `json:"name" binding:"required"`
	}

	// Invalid JSON
	c, w := setupTestContext("POST", "/test", nil)
	c.Request.Body = http.NoBody

	var req TestRequest
	result := BindJSON(c, &req)

	if result {
		t.Error("BindJSON() should return false for invalid request")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Response code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
