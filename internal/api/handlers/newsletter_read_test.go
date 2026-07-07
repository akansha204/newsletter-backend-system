package handlers

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/akansh204/newsletter-backend-system/internal/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type fakeNewsletterRepo struct {
	sends []domain.NewsletterSend
}

func (r *fakeNewsletterRepo) Create(n *domain.NewsletterSend) error       { return nil }
func (r *fakeNewsletterRepo) UpdateStatus(id string, status string) error { return nil }
func (r *fakeNewsletterRepo) IncrementSentCount(id string) error          { return nil }
func (r *fakeNewsletterRepo) IncrementFailCount(id string) error          { return nil }

func (r *fakeNewsletterRepo) FindByID(id string) (*domain.NewsletterSend, error) {
	for i := range r.sends {
		if r.sends[i].ID == id {
			return &r.sends[i], nil
		}
	}
	return nil, nil
}

func (r *fakeNewsletterRepo) List(limit, offset int) ([]domain.NewsletterSend, error) {
	if offset >= len(r.sends) {
		return nil, nil
	}
	end := offset + limit
	if end > len(r.sends) {
		end = len(r.sends)
	}
	return r.sends[offset:end], nil
}

func (r *fakeNewsletterRepo) CountAll() (int, error) {
	return len(r.sends), nil
}

func (r *fakeNewsletterRepo) SumCounts() (int, int, error) {
	sent, failed := 0, 0
	for _, s := range r.sends {
		sent += s.SentCount
		failed += s.FailCount
	}
	return sent, failed, nil
}

type fakeSubscriberRepo struct {
	total     int
	confirmed int
}

func (r *fakeSubscriberRepo) Create(s *domain.Subscriber) error { return nil }
func (r *fakeSubscriberRepo) FindByEmail(email string) (*domain.Subscriber, error) {
	return nil, nil
}
func (r *fakeSubscriberRepo) FindByToken(token string) (*domain.Subscriber, error) {
	return nil, nil
}
func (r *fakeSubscriberRepo) Confirm(id string) error { return nil }
func (r *fakeSubscriberRepo) FindAllConfirmed() ([]domain.Subscriber, error) {
	return nil, nil
}
func (r *fakeSubscriberRepo) CountAll() (int, error)       { return r.total, nil }
func (r *fakeSubscriberRepo) CountConfirmed() (int, error) { return r.confirmed, nil }

func newTestSends(count int) []domain.NewsletterSend {
	now := time.Now()
	sends := make([]domain.NewsletterSend, count)
	for i := range sends {
		sends[i] = domain.NewsletterSend{
			ID:        uuid.New().String(),
			Subject:   "subject",
			Body:      "body",
			Status:    domain.StatusSending,
			SentCount: 10,
			FailCount: 1,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	return sends
}

func decodeBody(t *testing.T, resp io.Reader) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(resp).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return body
}

func TestHandleListPagination(t *testing.T) {
	repo := &fakeNewsletterRepo{sends: newTestSends(30)}
	handler := NewNewsletterHandler(&fakeSubscriberRepo{}, repo, nil)

	app := fiber.New()
	app.Get("/sends", handler.HandleList)

	tests := []struct {
		name       string
		url        string
		wantItems  int
		wantLimit  float64
		wantOffset float64
	}{
		{name: "default limit", url: "/sends", wantItems: 20, wantLimit: 20, wantOffset: 0},
		{name: "explicit limit", url: "/sends?limit=5&offset=2", wantItems: 5, wantLimit: 5, wantOffset: 2},
		{name: "limit capped at max", url: "/sends?limit=500", wantItems: 30, wantLimit: 100, wantOffset: 0},
		{name: "negative values normalized", url: "/sends?limit=-1&offset=-3", wantItems: 20, wantLimit: 20, wantOffset: 0},
		{name: "offset past end", url: "/sends?offset=100", wantItems: 0, wantLimit: 20, wantOffset: 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest("GET", tc.url, nil))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if resp.StatusCode != fiber.StatusOK {
				t.Fatalf("expected 200, got %d", resp.StatusCode)
			}

			body := decodeBody(t, resp.Body)
			items, ok := body["items"].([]any)
			if !ok {
				t.Fatalf("expected items array, got %T", body["items"])
			}
			if len(items) != tc.wantItems {
				t.Errorf("expected %d items, got %d", tc.wantItems, len(items))
			}
			if body["total"] != float64(30) {
				t.Errorf("expected total 30, got %v", body["total"])
			}
			if body["limit"] != tc.wantLimit {
				t.Errorf("expected limit %v, got %v", tc.wantLimit, body["limit"])
			}
			if body["offset"] != tc.wantOffset {
				t.Errorf("expected offset %v, got %v", tc.wantOffset, body["offset"])
			}
		})
	}
}

func TestHandleGet(t *testing.T) {
	sends := newTestSends(1)
	repo := &fakeNewsletterRepo{sends: sends}
	handler := NewNewsletterHandler(&fakeSubscriberRepo{}, repo, nil)

	app := fiber.New()
	app.Get("/sends/:id", handler.HandleGet)

	t.Run("found", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest("GET", "/sends/"+sends[0].ID, nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		body := decodeBody(t, resp.Body)
		if body["id"] != sends[0].ID {
			t.Errorf("expected id %s, got %v", sends[0].ID, body["id"])
		}
		if body["sent_count"] != float64(10) {
			t.Errorf("expected sent_count 10, got %v", body["sent_count"])
		}
	})

	t.Run("unknown id returns 404", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest("GET", "/sends/"+uuid.New().String(), nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})

	t.Run("malformed id returns 404 not 500", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest("GET", "/sends/not-a-uuid", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})
}

func TestStatsHandler(t *testing.T) {
	repo := &fakeNewsletterRepo{sends: newTestSends(3)}
	subRepo := &fakeSubscriberRepo{total: 42, confirmed: 40}
	handler := NewStatsHandler(subRepo, repo)

	app := fiber.New()
	app.Get("/stats", handler.Handle)

	resp, err := app.Test(httptest.NewRequest("GET", "/stats", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := decodeBody(t, resp.Body)

	subscribers, ok := body["subscribers"].(map[string]any)
	if !ok {
		t.Fatalf("expected subscribers object, got %T", body["subscribers"])
	}
	if subscribers["total"] != float64(42) || subscribers["confirmed"] != float64(40) {
		t.Errorf("unexpected subscriber counts: %v", subscribers)
	}

	newsletters, ok := body["newsletters"].(map[string]any)
	if !ok {
		t.Fatalf("expected newsletters object, got %T", body["newsletters"])
	}
	if newsletters["total"] != float64(3) {
		t.Errorf("expected newsletters total 3, got %v", newsletters["total"])
	}
	if newsletters["sent_total"] != float64(30) {
		t.Errorf("expected sent_total 30, got %v", newsletters["sent_total"])
	}
	if newsletters["fail_total"] != float64(3) {
		t.Errorf("expected fail_total 3, got %v", newsletters["fail_total"])
	}
	if _, hasLast := newsletters["last_send"]; !hasLast {
		t.Error("expected last_send to be present")
	}
}
