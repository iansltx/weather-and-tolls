// Package ext exposes the kiosk's data-source lookups and Mercure
// interactions to PHP as methods of the namespaced Kiosk\Bridge class, via
// FrankenPHP's "PHP extensions written in Go" bridge. The Slim 4 routes and
// the poller call these instead of talking to the upstream services
// themselves, so API keys stay inside the Go layer.
package ext

// #cgo linux CFLAGS: -D_GNU_SOURCE
// #include <Zend/zend_types.h>
import "C"

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/dunglas/frankenphp"

	"ian.im/weather-and-tolls/apiclients"
)

// export_php:namespace Kiosk

const (
	envOpenWeatherKey   = "OPENWEATHER_API_KEY"
	envHubURL           = "MERCURE_INTERNAL_URL"
	envPublisherKey     = "MERCURE_PUBLISHER_JWT_KEY"
	envSubscriberKey    = "MERCURE_SUBSCRIBER_JWT_KEY"
	envInternalHubURL   = "http://127.0.0.1:80/.well-known/mercure"
	defaultOpenWeatherK = ""

	fetchTimeout = 20 * time.Second

	maxGeoCacheEntries = 64
)

var (
	clientsOnce sync.Once
	weather     *apiclients.WeatherClient
	tolls       *apiclients.TollClient
	hub         *MercureHub

	geoMu   sync.Mutex
	geoByID = make(map[string]apiclients.GeoCoords)
)

func bootstrap() {
	clientsOnce.Do(func() {
		weather = apiclients.NewWeatherClient(strings.TrimSpace(os.Getenv(envOpenWeatherKey)), nil)
		tolls = apiclients.NewTollClient(nil)
		hub = NewMercureHub(
			envOrDefault(envHubURL, envInternalHubURL),
			os.Getenv(envPublisherKey),
			os.Getenv(envSubscriberKey),
			nil,
		)
	})
}

func envOrDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

// errorResult builds the bridge's uniform error payload: every method
// returns an array carrying an "error" key, empty on success.
func errorResult(err error) unsafe.Pointer {
	return frankenphp.PHPMap(map[string]any{
		"error": err.Error(),
	})
}

// export_php:class Bridge
type Bridge struct{}

// export_php:method Bridge::resolveCoords(string $location): array
func (b *Bridge) ResolveCoords(location *C.zend_string) unsafe.Pointer {
	bootstrap()

	name := frankenphp.GoString(unsafe.Pointer(location))

	geoMu.Lock()
	cached, ok := geoByID[name]
	geoMu.Unlock()
	if ok {
		return frankenphp.PHPMap(map[string]any{
			"lat":   cached.Lat,
			"lon":   cached.Lon,
			"error": "",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	coords, err := weather.ResolveCoords(ctx, name)
	if err != nil {
		return errorResult(err)
	}

	geoMu.Lock()
	if len(geoByID) >= maxGeoCacheEntries {
		geoByID = make(map[string]apiclients.GeoCoords)
	}
	geoByID[name] = coords
	geoMu.Unlock()

	return frankenphp.PHPMap(map[string]any{
		"lat":   coords.Lat,
		"lon":   coords.Lon,
		"error": "",
	})
}

// export_php:method Bridge::fetchWeather(float $lat, float $lon): array
func (b *Bridge) FetchWeather(lat float64, lon float64) unsafe.Pointer {
	bootstrap()

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	data, err := weather.FetchWeather(ctx, apiclients.GeoCoords{Lat: lat, Lon: lon})
	if err != nil {
		return errorResult(err)
	}

	return frankenphp.PHPMap(map[string]any{
		"location":    data.Location,
		"temperature": data.Temperature,
		"humidity":    int64(data.Humidity),
		"dailyHigh":   data.DailyHigh,
		"dailyLow":    data.DailyLow,
		"units":       data.Units,
		"observedAt":  data.ObservedAt.UTC().Format(time.RFC3339),
		"error":       "",
	})
}

// export_php:method Bridge::fetchTolls(): array
func (b *Bridge) FetchTolls() unsafe.Pointer {
	bootstrap()

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	data, err := tolls.FetchTolls(ctx)
	if err != nil {
		return errorResult(err)
	}

	rates := make([]any, 0, len(data.Rates))
	for _, rate := range data.Rates {
		rates = append(rates, map[string]any{
			"name":          rate.Name,
			"tollTagRate":   rate.TollTagRate,
			"payByMailRate": rate.PayByMailRate,
			"closed":        rate.Closed,
		})
	}

	return frankenphp.PHPMap(map[string]any{
		"requestedAt": data.RequestedAt.UTC().Format(time.RFC3339Nano),
		"rates":       rates,
		"error":       "",
	})
}

// export_php:method Bridge::mercureSubscriptions(): array
func (b *Bridge) MercureSubscriptions() unsafe.Pointer {
	bootstrap()

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	entries, err := hub.Subscriptions(ctx)
	if err != nil {
		return errorResult(err)
	}

	topics := make([]any, 0, len(entries))
	for _, entry := range entries {
		topics = append(topics, entry.Topic)
	}

	return frankenphp.PHPMap(map[string]any{
		"topics": topics,
		"error":  "",
	})
}

// export_php:method Bridge::mercurePublish(string $topic, string $data, string $updateType, string $id): array
func (b *Bridge) MercurePublish(topic *C.zend_string, data *C.zend_string, updateType *C.zend_string, id *C.zend_string) unsafe.Pointer {
	bootstrap()

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	updateID, err := hub.Publish(
		ctx,
		frankenphp.GoString(unsafe.Pointer(topic)),
		[]byte(frankenphp.GoString(unsafe.Pointer(data))),
		frankenphp.GoString(unsafe.Pointer(updateType)),
		frankenphp.GoString(unsafe.Pointer(id)),
	)
	if err != nil {
		return errorResult(err)
	}

	return frankenphp.PHPMap(map[string]any{
		"id":    updateID,
		"error": "",
	})
}

var _ = fmt.Sprintf // retained import for future debug helpers
