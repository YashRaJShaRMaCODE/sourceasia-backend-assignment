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

var errRequestTooLarge = errors.New("request body exceeds 1 MiB limit")

type Handler struct {
	rateService    *services.RateLimitService
	productService *services.ProductService
}

func NewHandler(rs *services.RateLimitService, ps *services.ProductService) *Handler {
	return &Handler{rateService: rs, productService: ps}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "POST")
		return
	}

	var req models.RateLimitRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	req.UserID = strings.TrimSpace(req.UserID)
	if req.UserID == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "user_id required"})
		return
	}
	if len(req.UserID) > 128 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "user_id must be at most 128 bytes"})
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
		methodNotAllowed(w, "GET")
		return
	}
	jsonResponse(w, http.StatusOK, h.rateService.GetStats())
}

func (h *Handler) HandleProducts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var p models.Product
		if err := decodeJSON(w, r, &p); err != nil {
			writeDecodeError(w, err)
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

		items, total := h.productService.ListProducts(limit, offset)
		jsonResponse(w, http.StatusOK, models.ProductListResponse{
			Items:  items,
			Limit:  limit,
			Offset: offset,
			Total:  total,
		})
	default:
		methodNotAllowed(w, "GET, POST")
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
			jsonResponse(w, http.StatusNotFound, map[string]string{"error": "product not found"})
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
			writeDecodeError(w, err)
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
		methodNotAllowed(w, "GET, POST")
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if isRequestTooLarge(err) {
			return errRequestTooLarge
		}
		return errors.New("invalid JSON request body")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if isRequestTooLarge(err) {
			return errRequestTooLarge
		}
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func isRequestTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func writeDecodeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, errRequestTooLarge) {
		status = http.StatusRequestEntityTooLarge
	}
	jsonResponse(w, status, map[string]string{"error": err.Error()})
}

func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func queryInt(r *http.Request, key string, fallback int) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func jsonResponse(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}
