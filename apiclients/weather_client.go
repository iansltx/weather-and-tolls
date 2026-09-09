// Package apiclients contains the HTTP clients for the two data sources
// displayed by the kiosk: OpenWeather weather data and the Mobility
// Authority's Austin-area toll rates. The kiosk PHP extension (loaded
// through FrankenPHP's Go/PHP bridge) builds on this package.
package apiclients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var zipRe = regexp.MustCompile(`^\d{5}(,|$)`)

const (
	EnvOpenWeatherBaseURL = "OPENWEATHER_BASE_URL"

	defaultOpenWeatherBaseURL = "https://api.openweathermap.org"
)

// GeoCoords holds the resolved latitude and longitude for a location.
type GeoCoords struct {
	Lat float64
	Lon float64
}

type geoDirectResponse struct {
	Name    string  `json:"name"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Country string  `json:"country"`
	State   string  `json:"state"`
}

type geoZipResponse struct {
	Name    string  `json:"name"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Country string  `json:"country"`
}

type WeatherClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type WeatherData struct {
	Location    string
	Temperature float64
	Humidity    int
	DailyHigh   float64
	DailyLow    float64
	Units       string
	ObservedAt  time.Time
}

type openWeatherCurrentResponse struct {
	Name string `json:"name"`
	Main struct {
		Temperature float64 `json:"temp"`
		Humidity    int     `json:"humidity"`
		TempMin     float64 `json:"temp_min"`
		TempMax     float64 `json:"temp_max"`
	} `json:"main"`
	Sys struct {
		Country string `json:"country"`
	} `json:"sys"`
	DT int64 `json:"dt"`
}

type openWeatherForecastResponse struct {
	List []struct {
		DT   int64 `json:"dt"`
		Main struct {
			TempMin float64 `json:"temp_min"`
			TempMax float64 `json:"temp_max"`
		} `json:"main"`
	} `json:"list"`
}

func NewWeatherClient(apiKey string, httpClient *http.Client) *WeatherClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &WeatherClient{
		baseURL:    EnvOrDefault(EnvOpenWeatherBaseURL, defaultOpenWeatherBaseURL),
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: httpClient,
	}
}

// ResolveCoords resolves a location string (city name or US ZIP) to geographic
// coordinates using the OpenWeatherMap Geocoding API. It is intended to be
// called once at app startup.
func (c *WeatherClient) ResolveCoords(ctx context.Context, location string) (GeoCoords, error) {
	if c.apiKey == "" {
		return GeoCoords{}, errors.New("OpenWeather API key is required")
	}
	location = strings.TrimSpace(location)
	if location == "" {
		return GeoCoords{}, errors.New("location is required")
	}

	if zipRe.MatchString(location) {
		return c.resolveCoordsZIP(ctx, location)
	}
	return c.resolveCoordsCity(ctx, location)
}

func (c *WeatherClient) resolveCoordsZIP(ctx context.Context, location string) (GeoCoords, error) {
	endpoint, err := JoinURL(c.baseURL, "/geo/1.0/zip")
	if err != nil {
		return GeoCoords{}, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return GeoCoords{}, fmt.Errorf("parse geo zip URL: %w", err)
	}
	q := parsed.Query()
	q.Set("zip", normalizeUSZIP(location))
	q.Set("appid", c.apiKey)
	parsed.RawQuery = q.Encode()

	var payload geoZipResponse
	if err := c.getJSON(ctx, parsed.String(), &payload); err != nil {
		return GeoCoords{}, fmt.Errorf("resolve zip coords: %w", err)
	}
	return GeoCoords{Lat: payload.Lat, Lon: payload.Lon}, nil
}

func (c *WeatherClient) resolveCoordsCity(ctx context.Context, location string) (GeoCoords, error) {
	endpoint, err := JoinURL(c.baseURL, "/geo/1.0/direct")
	if err != nil {
		return GeoCoords{}, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return GeoCoords{}, fmt.Errorf("parse geo direct URL: %w", err)
	}
	q := parsed.Query()
	q.Set("q", location)
	q.Set("limit", "1")
	q.Set("appid", c.apiKey)
	parsed.RawQuery = q.Encode()

	var payload []geoDirectResponse
	if err := c.getJSON(ctx, parsed.String(), &payload); err != nil {
		return GeoCoords{}, fmt.Errorf("resolve city coords: %w", err)
	}
	if len(payload) == 0 {
		return GeoCoords{}, fmt.Errorf("no results found for location %q", location)
	}
	return GeoCoords{Lat: payload[0].Lat, Lon: payload[0].Lon}, nil
}

func (c *WeatherClient) FetchWeather(ctx context.Context, coords GeoCoords) (WeatherData, error) {
	if c == nil {
		return WeatherData{}, errors.New("weather client is nil")
	}
	if c.apiKey == "" {
		return WeatherData{}, errors.New("OpenWeather API key is required")
	}

	current, err := c.fetchCurrentWeather(ctx, coords)
	if err != nil {
		return WeatherData{}, err
	}

	high := current.Main.TempMax
	low := current.Main.TempMin

	observedAt := time.Now()
	if current.DT > 0 {
		observedAt = time.Unix(current.DT, 0)
	}

	forecast, err := c.fetchForecast(ctx, coords)
	if err == nil && len(forecast.List) > 0 {
		forecastWindowEnd := observedAt.Add(24 * time.Hour)

		foundForecast := false
		for _, item := range forecast.List {
			itemTime := time.Unix(item.DT, 0)
			if itemTime.Before(observedAt) || itemTime.After(forecastWindowEnd) {
				continue
			}

			if !foundForecast {
				high = item.Main.TempMax
				low = item.Main.TempMin
				foundForecast = true
				continue
			}

			if item.Main.TempMax > high {
				high = item.Main.TempMax
			}
			if item.Main.TempMin < low {
				low = item.Main.TempMin
			}
		}
	}

	displayLocation := current.Name
	if current.Sys.Country != "" {
		displayLocation = fmt.Sprintf("%s, %s", displayLocation, current.Sys.Country)
	}

	return WeatherData{
		Location:    displayLocation,
		Temperature: current.Main.Temperature,
		Humidity:    current.Main.Humidity,
		DailyHigh:   high,
		DailyLow:    low,
		Units:       "imperial",
		ObservedAt:  observedAt,
	}, nil
}

func (c *WeatherClient) fetchCurrentWeather(ctx context.Context, coords GeoCoords) (openWeatherCurrentResponse, error) {
	var payload openWeatherCurrentResponse

	endpoint, err := c.weatherUrlForCoordinates("/data/2.5/weather", coords)
	if err != nil {
		return payload, err
	}

	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return payload, fmt.Errorf("fetch current weather: %w", err)
	}

	return payload, nil
}

func (c *WeatherClient) fetchForecast(ctx context.Context, coords GeoCoords) (openWeatherForecastResponse, error) {
	var payload openWeatherForecastResponse

	endpoint, err := c.weatherUrlForCoordinates("/data/2.5/forecast", coords)
	if err != nil {
		return payload, err
	}

	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return payload, fmt.Errorf("fetch weather forecast: %w", err)
	}

	return payload, nil
}

func (c *WeatherClient) weatherUrlForCoordinates(path string, coords GeoCoords) (string, error) {
	endpoint, err := JoinURL(c.baseURL, path)
	if err != nil {
		return "", err
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse OpenWeather URL: %w", err)
	}

	query := parsed.Query()
	query.Set("appid", c.apiKey)
	query.Set("units", "imperial")
	query.Set("lat", strconv.FormatFloat(coords.Lat, 'f', -1, 64))
	query.Set("lon", strconv.FormatFloat(coords.Lon, 'f', -1, 64))

	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (c *WeatherClient) getJSON(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func normalizeUSZIP(location string) string {
	location = strings.TrimSpace(location)
	if strings.Contains(location, ",") {
		return location
	}
	return location + ",us"
}
