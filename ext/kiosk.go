// Package ext exposes the kiosk's data-source lookups and Mercure
// interactions to PHP as native functions, via FrankenPHP's
// "PHP extensions written in Go" bridge. The Slim 4 routes and the poller
// call these instead of talking to the upstream services themselves, so API
// keys stay inside the Go layer.
//
// Errors surface as PHP exceptions: the exported functions return the
// payload alongside an error message (nil on success), and the PHP glue in
// kiosk.c — hand-maintained, since the extension generator only supports
// single-value returns — throws a RuntimeException and propagates it to the
// caller.
package ext

// #cgo linux CFLAGS: -D_GNU_SOURCE
// #include <Zend/zend_types.h>
// #include <stdlib.h>
import "C"

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/dunglas/frankenphp"

	"ian.im/weather-and-tolls/apiclients"
)

const (
	envOpenWeatherKey = "OPENWEATHER_API_KEY"
	envHubURL         = "MERCURE_INTERNAL_URL"
	envPublisherKey   = "MERCURE_PUBLISHER_JWT_KEY"
	envSubscriberKey  = "MERCURE_SUBSCRIBER_JWT_KEY"
	internalHubURL    = "http://127.0.0.1:80/.well-known/mercure"

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

func init() {
	// Links the extension into the engine; kiosk.c defines the module entry
	// and its PHP_FUNCTION glue.
	frankenphp.RegisterExtension(unsafe.Pointer(&C.kiosk_module_entry))
}

func bootstrap() {
	clientsOnce.Do(func() {
		weather = apiclients.NewWeatherClient(strings.TrimSpace(os.Getenv(envOpenWeatherKey)), nil)
		tolls = apiclients.NewTollClient(nil)
		hub = NewMercureHub(
			envOrDefault(envHubURL, internalHubURL),
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

//export go_kiosk_resolve_coords
func go_kiosk_resolve_coords(location *C.zend_string) (unsafe.Pointer, *C.char) {
	bootstrap()

	name := frankenphp.GoString(unsafe.Pointer(location))

	geoMu.Lock()
	cached, ok := geoByID[name]
	geoMu.Unlock()
	if ok {
		return frankenphp.PHPMap(map[string]any{
			"lat": cached.Lat,
			"lon": cached.Lon,
		}), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	coords, err := weather.ResolveCoords(ctx, name)
	if err != nil {
		return nil, C.CString(err.Error())
	}

	geoMu.Lock()
	if len(geoByID) >= maxGeoCacheEntries {
		geoByID = make(map[string]apiclients.GeoCoords)
	}
	geoByID[name] = coords
	geoMu.Unlock()

	return frankenphp.PHPMap(map[string]any{
		"lat": coords.Lat,
		"lon": coords.Lon,
	}), nil
}

//export go_kiosk_fetch_weather
func go_kiosk_fetch_weather(lat float64, lon float64) (unsafe.Pointer, *C.char) {
	bootstrap()

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	data, err := weather.FetchWeather(ctx, apiclients.GeoCoords{Lat: lat, Lon: lon})
	if err != nil {
		return nil, C.CString(err.Error())
	}

	return frankenphp.PHPMap(map[string]any{
		"location":    data.Location,
		"temperature": data.Temperature,
		"humidity":    int64(data.Humidity),
		"dailyHigh":   data.DailyHigh,
		"dailyLow":    data.DailyLow,
		"units":       data.Units,
		"observedAt":  data.ObservedAt.UTC().Format(time.RFC3339),
	}), nil
}

//export go_kiosk_fetch_tolls
func go_kiosk_fetch_tolls() (unsafe.Pointer, *C.char) {
	bootstrap()

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	data, err := tolls.FetchTolls(ctx)
	if err != nil {
		return nil, C.CString(err.Error())
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
	}), nil
}

//export go_kiosk_mercure_subscriptions
func go_kiosk_mercure_subscriptions() (unsafe.Pointer, *C.char) {
	bootstrap()

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	entries, err := hub.Subscriptions(ctx)
	if err != nil {
		return nil, C.CString(err.Error())
	}

	topics := make([]any, 0, len(entries))
	for _, entry := range entries {
		topics = append(topics, entry.Topic)
	}

	return frankenphp.PHPMap(map[string]any{
		"topics": topics,
	}), nil
}

//export go_kiosk_mercure_publish
func go_kiosk_mercure_publish(topic *C.zend_string, data *C.zend_string, typ *C.zend_string, id *C.zend_string) (unsafe.Pointer, *C.char) {
	bootstrap()

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	updateID, err := hub.Publish(
		ctx,
		frankenphp.GoString(unsafe.Pointer(topic)),
		[]byte(frankenphp.GoString(unsafe.Pointer(data))),
		frankenphp.GoString(unsafe.Pointer(typ)),
		frankenphp.GoString(unsafe.Pointer(id)),
	)
	if err != nil {
		return nil, C.CString(err.Error())
	}

	return frankenphp.PHPMap(map[string]any{
		"id": updateID,
	}), nil
}
