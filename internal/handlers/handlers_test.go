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
}

func TestRequestRejectsExtraJSONAndOversizedBody(t *testing.T) {
	for _, body := range []string{
		`{"user_id":"one","payload":{}} {"user_id":"two"}`,
		`{"user_id":"one","payload":{},"unexpected":true}`,
		`{"user_id":"` + strings.Repeat("x", 1<<20) + `","payload":{}}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/request", strings.NewReader(body))
		res := httptest.NewRecorder()
		testHandler().HandleRequest(res, req)
		if res.Code != http.StatusBadRequest {
			t.Errorf("status = %d for rejected request body, want %d", res.Code, http.StatusBadRequest)
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

	listed := call(h, http.MethodGet, "/products?limit=1", "")
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"sku":"LAMP-001"`) {
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
}

func call(h *Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	res := httptest.NewRecorder()
	if strings.HasPrefix(path, "/products/") {
		h.HandleProductByID(res, req)
	} else {
		h.HandleProducts(res, req)
	}
	return res
}
