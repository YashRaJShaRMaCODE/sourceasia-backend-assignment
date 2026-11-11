package services

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/YashRaJShaRMaCODE/sourceasia-backend-assignment/internal/models"
	"github.com/YashRaJShaRMaCODE/sourceasia-backend-assignment/internal/store"
)

func TestCreateProductTrimsAndRejectsDuplicateSKU(t *testing.T) {
	service := NewProductService(store.NewInMemoryStore())
	first, err := service.CreateProduct(&models.Product{Name: "  Lamp ", SKU: " lamp-1 "})
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "Lamp" || first.SKU != "lamp-1" {
		t.Fatalf("unexpected normalized product: %#v", first)
	}
	if _, err := service.CreateProduct(&models.Product{Name: "Another", SKU: "lamp-1"}); err == nil {
		t.Fatal("expected duplicate SKU to fail")
	}
}

func TestConcurrentCreateAllowsOnlyOneSKU(t *testing.T) {
	service := NewProductService(store.NewInMemoryStore())
	var succeeded atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := service.CreateProduct(&models.Product{Name: "Lamp", SKU: "same"}); err == nil {
				succeeded.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := succeeded.Load(); got != 1 {
		t.Fatalf("successful creates = %d, want 1", got)
	}
}

func TestValidateMediaRequiresAbsoluteHTTPURL(t *testing.T) {
	for _, url := range []string{"//example.com/image.png", "file:///tmp/image.png", "https:///missing-host"} {
		if err := validateMedia([]string{url}, nil); err == nil {
			t.Errorf("validateMedia(%q) succeeded, want error", url)
		}
	}
}
