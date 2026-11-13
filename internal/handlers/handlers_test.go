package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YashRaJShaRMaCODE/sourceasia-backend-assignment/internal/services"
	"github.com/YashRaJShaRMaCODE/sourceasia-backend-assignment/internal/store"
)

func testHandler() *Handler {
	data := store.NewInMemoryStore()
	return NewHandler(services.NewRateLimitService(data), services.NewProductService(data))
}

func TestProductsRejectsNegativeOffset(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/products?offset=-1", nil)
	res := httptest.NewRecorder()
	testHandler().HandleProducts(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestProductsRejectsUnsupportedMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/products", nil)
	res := httptest.NewRecorder()
	testHandler().HandleProducts(res, req)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusMethodNotAllowed)
	}
	if res.Header().Get("Allow") != "GET, POST" || !strings.Contains(res.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("unexpected method error headers: %#v", res.Header())
	}
}

func TestRequestRejectsExtraAndUnknownJSON(t *testing.T) {
	for _, body := range []string{
		`{"user_id":"one","payload":{}} {"user_id":"two"}`,
		`{"user_id":"one","payload":{},"unexpected":true}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/request", strings.NewReader(body))
		res := httptest.NewRecorder()
		testHandler().HandleRequest(res, req)
		if res.Code != http.StatusBadRequest {
			t.Errorf("status = %d for rejected request body, want %d", res.Code, http.StatusBadRequest)
		}
	}
}

func TestOversizedJSONBodyReturns413(t *testing.T) {
	body := `{"user_id":"` + strings.Repeat("x", 1<<20) + `","payload":{}}`
	req := httptest.NewRequest(http.MethodPost, "/request", strings.NewReader(body))
	res := httptest.NewRecorder()
	testHandler().HandleRequest(res, req)
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(res.Body.String(), "1 MiB") {
		t.Fatalf("unexpected error body: %s", res.Body.String())
	}
}

func TestRequestTrimsAndBoundsUserID(t *testing.T) {
	h := testHandler()
	for i := 0; i < 5; i++ {
		body := `{"user_id":"  demo-user  ","payload":{}}`
		if i > 0 {
			body = `{"user_id":"demo-user","payload":{}}`
		}
		res := call(h, http.MethodPost, "/request", body)
		if res.Code != http.StatusCreated {
			t.Fatalf("request %d status = %d, want %d", i+1, res.Code, http.StatusCreated)
		}
	}
	if got := call(h, http.MethodPost, "/request", `{"user_id":"demo-user","payload":{}}`).Code; got != http.StatusTooManyRequests {
		t.Fatalf("sixth request status = %d, want %d", got, http.StatusTooManyRequests)
	}
	for _, userID := range []string{"   ", strings.Repeat("x", 129)} {
		body, _ := json.Marshal(map[string]any{"user_id": userID, "payload": map[string]any{}})
		if got := call(h, http.MethodPost, "/request", string(body)).Code; got != http.StatusBadRequest {
			t.Errorf("user ID length %d status = %d, want %d", len(userID), got, http.StatusBadRequest)
		}
	}
}

func TestProductCreateListAndAddMedia(t *testing.T) {
	h := testHandler()
	created := call(h, http.MethodPost, "/products", `{"name":"Desk lamp","sku":"LAMP-001"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", created.Code, http.StatusCreated, created.Body.String())
	}
	var product struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &product); err != nil || product.ID == "" {
		t.Fatalf("created product has no ID: %v", err)
	}
	if other := call(h, http.MethodPost, "/products", `{"name":"Desk chair","sku":"CHAIR-001"}`); other.Code != http.StatusCreated {
		t.Fatalf("second create status = %d, want %d: %s", other.Code, http.StatusCreated, other.Body.String())
	}

	listed := call(h, http.MethodGet, "/products?limit=1", "")
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"sku":"LAMP-001"`) || !strings.Contains(listed.Body.String(), `"total":2`) {
		t.Fatalf("unexpected product list response: %d %s", listed.Code, listed.Body.String())
	}

	media := call(h, http.MethodPost, "/products/"+product.ID+"/media", `{"image_urls":["https://example.com/lamp.png"]}`)
	if media.Code != http.StatusOK {
		t.Fatalf("add media status = %d: %s", media.Code, media.Body.String())
	}
	got := call(h, http.MethodGet, "/products/"+product.ID, "")
	if got.Code != http.StatusOK || !strings.Contains(got.Body.String(), "https://example.com/lamp.png") {
		t.Fatalf("unexpected product detail response: %d %s", got.Code, got.Body.String())
	}
	if got.Header().Get("Cache-Control") != "no-store" || got.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing safe response headers: %#v", got.Header())
	}
}

func call(h *Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	res := httptest.NewRecorder()
	switch {
	case path == "/products":
		h.HandleProducts(res, req)
	case strings.HasPrefix(path, "/products/"):
		h.HandleProductByID(res, req)
	case strings.HasPrefix(path, "/request"):
		h.HandleRequest(res, req)
	case strings.HasPrefix(path, "/health"):
		h.HealthCheck(res, req)
	case strings.HasPrefix(path, "/stats"):
		h.HandleStats(res, req)
	default:
		h.HandleProducts(res, req)
	}
	return res
}
