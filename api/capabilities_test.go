package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deposist/s-ui-x/core/capabilities"
	"github.com/gin-gonic/gin"
)

func TestGetCapabilitiesReturnsOfficialBuildView(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/capabilities", (&ApiService{}).GetCapabilities)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/capabilities", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /api/capabilities returned %d", recorder.Code)
	}
	var response struct {
		Success bool                 `json:"success"`
		Obj     capabilities.APIView `json:"obj"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Success {
		t.Fatal("capability response was unsuccessful")
	}
	if _, found := response.Obj.BuildTags["with_wireguard"]; !found {
		t.Fatal("capability response omitted official manifest build tag")
	}
	for _, endpoint := range response.Obj.Endpoints {
		if endpoint.Type == "vpn" {
			t.Fatal("capability response exposed extended-only endpoint")
		}
	}
}
