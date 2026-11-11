package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/YashRaJShaRMaCODE/sourceasia-backend-assignment/internal/models"
	"github.com/YashRaJShaRMaCODE/sourceasia-backend-assignment/internal/services"
)

type Handler struct {
	rateService    *services.RateLimitService
	productService *services.ProductService
}

func NewHandler(rs *services.RateLimitService, ps *services.ProductService) *Handler {
	return &Handler{rateService: rs, productService: ps}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RateLimitRequest
	if err := decodeJSON(w, r, &req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if req.UserID == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "user_id required"})
		return
	}

	if !h.rateService.ProcessRequest(req.UserID) {
		jsonResponse(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]string{"status": "accepted"})
}

func (h *Handler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	jsonResponse(w, http.StatusOK, h.rateService.GetStats())
}

func (h *Handler) HandleProducts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var p models.Product
		if err := decodeJSON(w, r, &p); err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		created, err := h.productService.CreateProduct(&p)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "conflict") {
				status = http.StatusConflict
			}
			jsonResponse(w, status, map[string]string{"error": err.Error()})
			return
		}
		jsonResponse(w, http.StatusCreated, created)

	case http.MethodGet:
		limit, err := queryInt(r, "limit", 20)
		if err != nil || limit <= 0 {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "limit must be a positive integer"})
			return
		}
		offset, err := queryInt(r, "offset", 0)
		if err != nil || offset < 0 {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "offset must be a non-negative integer"})
			return
		}
		if limit <= 0 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}

		items := h.productService.ListProducts(limit, offset)
		jsonResponse(w, http.StatusOK, models.ProductListResponse{
			Items:  items,
			Limit:  limit,
			Offset: offset,
		})
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) HandleProductByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/products/")
	if id == "" {
		h.HandleProducts(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		p, ok := h.productService.GetByID(id)
		if !ok {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		jsonResponse(w, http.StatusOK, p)

	case http.MethodPost:
		if !strings.HasSuffix(r.URL.Path, "/media") {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		id = strings.TrimSuffix(id, "/media")
		var req models.AddMediaRequest
		if err := decodeJSON(w, r, &req); err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := h.productService.AddMedia(id, req); err != nil {
			status := http.StatusBadRequest
			if err.Error() == "not found" {
				status = http.StatusNotFound
			}
			jsonResponse(w, status, map[string]string{"error": err.Error()})
			return
		}
		jsonResponse(w, http.StatusOK, map[string]string{"status": "media added"})
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return errors.New("invalid JSON request body")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func queryInt(r *http.Request, key string, fallback int) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func jsonResponse(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}
