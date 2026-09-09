package ext

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const mercureJWTTTL = time.Hour

// MercureHub publishes updates to, and introspects the subscribers of, the
// FrankenPHP-embedded Mercure hub over the loopback interface. It is used by
// the PHP side through the kiosk_mercure_* bridge functions.
type MercureHub struct {
	// baseURL is the hub endpoint, e.g. http://127.0.0.1:8080/.well-known/mercure.
	baseURL       string
	publisherKey  string
	subscriberKey string
	client        *http.Client
	nowFn         func() time.Time
}

func NewMercureHub(baseURL, publisherKey, subscriberKey string, client *http.Client) *MercureHub {
	if client == nil {
		client = http.DefaultClient
	}
	return &MercureHub{
		baseURL:       strings.TrimRight(baseURL, "/"),
		publisherKey:  publisherKey,
		subscriberKey: subscriberKey,
		client:        client,
		nowFn:         time.Now,
	}
}

// Publish dispatches an update to the hub and returns the update ID assigned
// by the hub.
func (h *MercureHub) Publish(ctx context.Context, topic string, data []byte, typ string, id string) (string, error) {
	form := url.Values{
		"topic": {topic},
		"data":  {string(data)},
		"type":  {typ},
	}
	if id != "" {
		form.Set("id", id)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create mercure publish request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+PublishJWT(h.publisherKey, mercureJWTTTL, h.nowFn()))

	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("perform mercure publish: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected mercure publish status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return strings.TrimSpace(string(body)), nil
}

// SubscriptionEntry describes one active subscriber connection, mirroring the
// Mercure subscription API response.
type SubscriptionEntry struct {
	ID     string `json:"id"`
	Topic  string `json:"topic"`
	Active bool   `json:"active"`
}

// Subscriptions returns every active subscription known to the hub.
func (h *MercureHub) Subscriptions(ctx context.Context) ([]SubscriptionEntry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.baseURL+"/subscriptions", nil)
	if err != nil {
		return nil, fmt.Errorf("create mercure subscriptions request: %w", err)
	}

	req.Header.Set("Accept", "application/ld+json")
	req.Header.Set("Authorization", "Bearer "+SubscriptionsJWT(h.subscriberKey, mercureJWTTTL, h.nowFn()))

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform mercure subscriptions request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("unexpected mercure subscriptions status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Subscriptions []SubscriptionEntry `json:"subscriptions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode mercure subscriptions response: %w", err)
	}

	return payload.Subscriptions, nil
}
