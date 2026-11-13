package services

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/YashRaJShaRMaCODE/sourceasia-backend-assignment/internal/models"
	"github.com/YashRaJShaRMaCODE/sourceasia-backend-assignment/internal/store"
	"github.com/google/uuid"
)

type ProductService struct {
	store *store.InMemoryStore
}

func NewProductService(s *store.InMemoryStore) *ProductService {
	return &ProductService{store: s}
}

func (s *ProductService) CreateProduct(p *models.Product) (*models.Product, error) {
	p.Name = strings.TrimSpace(p.Name)
	p.SKU = strings.TrimSpace(p.SKU)
	if p.Name == "" || p.SKU == "" {
		return nil, errors.New("name and sku are required")
	}
	if len(p.Name) > 120 || len(p.SKU) > 64 {
		return nil, errors.New("name must be at most 120 characters and sku at most 64")
	}

	if err := validateMedia(p.ImageURLs, p.VideoURLs); err != nil {
		return nil, err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("generate product ID: %w", err)
	}
	p.ID = id.String()
	p.CreatedAt = time.Now()

	if !s.store.AddProduct(p) {
		return nil, fmt.Errorf("conflict: sku %s already exists", p.SKU)
	}
	return p, nil
}

func (s *ProductService) AddMedia(id string, req models.AddMediaRequest) error {
	if len(req.ImageURLs) == 0 && len(req.VideoURLs) == 0 {
		return errors.New("at least one media array required")
	}

	if err := validateMedia(req.ImageURLs, req.VideoURLs); err != nil {
		return err
	}

	if ok := s.store.UpdateProductMedia(id, req.ImageURLs, req.VideoURLs); !ok {
		return errors.New("not found")
	}
	return nil
}

func validateMedia(imgs, vids []string) error {
	if len(imgs) > 20 || len(vids) > 20 {
		return errors.New("max 20 URLs per array")
	}
	for _, list := range [][]string{imgs, vids} {
		for _, u := range list {
			if len(u) > 2048 {
				return errors.New("URL max length 2048 chars")
			}
			parsed, err := url.Parse(u)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
				return fmt.Errorf("invalid URL: %s", u)
			}
		}
	}
	return nil
}

type RateLimitService struct {
	store *store.InMemoryStore
}

func NewRateLimitService(s *store.InMemoryStore) *RateLimitService {
	return &RateLimitService{store: s}
}

func (s *RateLimitService) ProcessRequest(userID string) bool {
	return s.store.AllowRequest(userID)
}

func (s *RateLimitService) GetStats() models.StatsResponse {
	return models.StatsResponse{Users: s.store.GetStats()}
}

func (s *ProductService) ListProducts(limit, offset int) ([]models.ProductSummary, int) {
	return s.store.ListProducts(limit, offset)
}

func (s *ProductService) GetByID(id string) (*models.Product, bool) {
	return s.store.GetProductByID(id)
}
