package apiclients

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeatherClientFetchWeatherWithMockedOpenWeatherResponses(t *testing.T) {

	now := time.Now()
	firstForecastAt := now.Add(1 * time.Hour).Unix()
	secondForecastAt := now.Add(4 * time.Hour).Unix()

	var weatherRequestSeen bool
	var forecastRequestSeen bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		assert.Equal(t, "test-api-key", query.Get("appid"))
		assert.Equal(t, "imperial", query.Get("units"))
		assert.NotEmpty(t, query.Get("lat"), "lat query parameter was empty, want non-empty")
		assert.NotEmpty(t, query.Get("lon"), "lon query parameter was empty, want non-empty")

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/data/2.5/weather":
			weatherRequestSeen = true
			fmt.Fprintf(w, `{
				"name": "Austin",
				"dt": %d,
				"sys": {
					"country": "US"
				},
				"main": {
					"temp": 81.6,
					"humidity": 58,
					"temp_min": 76.0,
					"temp_max": 88.0
				}
			}`, now.Unix())

		case "/data/2.5/forecast":
			forecastRequestSeen = true
			fmt.Fprintf(w, `{
				"list": [
					{
						"dt": %d,
						"main": {
							"temp_min": 74.5,
							"temp_max": 89.1
						}
					},
					{
						"dt": %d,
						"main": {
							"temp_min": 72.4,
							"temp_max": 93.8
						}
					}
				]
			}`, firstForecastAt, secondForecastAt)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv(EnvOpenWeatherBaseURL, server.URL)

	client := NewWeatherClient("test-api-key", server.Client())

	got, err := client.FetchWeather(t.Context(), GeoCoords{Lat: 30.35, Lon: -97.72})
	require.NoError(t, err)

	require.True(t, weatherRequestSeen, "mock weather endpoint was not called")
	require.True(t, forecastRequestSeen, "mock forecast endpoint was not called")

	assert.Equal(t, "Austin, US", got.Location)
	assert.Equal(t, 81.6, got.Temperature)
	assert.Equal(t, 58, got.Humidity)
	assert.Equal(t, 72.4, got.DailyLow)
	assert.Equal(t, 93.8, got.DailyHigh)
	assert.Equal(t, "imperial", got.Units)
}

func TestWeatherClientResolveCoordsForZIPCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/geo/1.0/zip":
			query := r.URL.Query()
			assert.Equal(t, "78757,us", query.Get("zip"))
			assert.Equal(t, "test-api-key", query.Get("appid"))
			fmt.Fprint(w, `{"zip":"78757","name":"Austin","lat":30.3518,"lon":-97.7236,"country":"US"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv(EnvOpenWeatherBaseURL, server.URL)

	client := NewWeatherClient("test-api-key", server.Client())

	got, err := client.ResolveCoords(t.Context(), "78757")
	require.NoError(t, err)

	assert.Equal(t, 30.3518, got.Lat)
	assert.Equal(t, -97.7236, got.Lon)
}

func TestWeatherClientResolveCoordsForCityName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/geo/1.0/direct":
			query := r.URL.Query()
			assert.Equal(t, "Austin, TX", query.Get("q"))
			assert.Equal(t, "1", query.Get("limit"))
			assert.Equal(t, "test-api-key", query.Get("appid"))
			fmt.Fprint(w, `[{"name":"Austin","lat":30.2711,"lon":-97.7437,"country":"US","state":"Texas"}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv(EnvOpenWeatherBaseURL, server.URL)

	client := NewWeatherClient("test-api-key", server.Client())

	got, err := client.ResolveCoords(t.Context(), "Austin, TX")
	require.NoError(t, err)

	assert.Equal(t, 30.2711, got.Lat)
	assert.Equal(t, -97.7437, got.Lon)
}

func TestWeatherClientFetchWeatherRequiresAPIKey(t *testing.T) {

	client := NewWeatherClient("", nil)

	_, err := client.FetchWeather(t.Context(), GeoCoords{})
	require.Error(t, err)
	require.ErrorContains(t, err, "API key")
}

func TestWeatherClientFetchWeatherReturnsHTTPError(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid api key", http.StatusUnauthorized)
	}))
	defer server.Close()

	t.Setenv(EnvOpenWeatherBaseURL, server.URL)

	client := NewWeatherClient("bad-key", server.Client())

	_, err := client.FetchWeather(t.Context(), GeoCoords{Lat: 30.35, Lon: -97.72})
	require.Error(t, err)
	require.ErrorContains(t, err, "unexpected status 401")
}

func TestOpenWeatherSamplePayloadShape(t *testing.T) {

	const samplePayload = `{
		"name": "Austin",
		"dt": 1766600000,
		"sys": {
			"country": "US"
		},
		"main": {
			"temp": 81.6,
			"humidity": 58,
			"temp_min": 76.0,
			"temp_max": 88.0
		}
	}`

	var decoded openWeatherCurrentResponse
	err := json.Unmarshal([]byte(samplePayload), &decoded)
	require.NoError(t, err, "sample OpenWeather payload failed to decode")

	assert.Equal(t, "Austin", decoded.Name)
	assert.Equal(t, 81.6, decoded.Main.Temperature)
	assert.Equal(t, 58, decoded.Main.Humidity)
}
