package apiclients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	EnvTollRatesURL = "TOLL_RATES_URL"

	defaultTollRatesURL = "https://www.mobilityauthority.com/wp-admin/admin-ajax.php"

	DefaultTollFacilitiesQuery = "03TO,07TO,01TO"
)

type TollClient struct {
	url             string
	facilitiesQuery string
	httpClient      *http.Client
	now             func() time.Time
}

type TollData struct {
	RequestedAt time.Time
	Rates       []TollRate
}

type TollRate struct {
	Name          string
	TollTagRate   float64
	PayByMailRate float64
	Closed        bool
}

type tollRateResponse struct {
	TollingPointName string  `json:"tollingPointName"`
	TripRate         float64 `json:"tripRate"`
	PBMRate          float64 `json:"pbmRate"`
	Status           string  `json:"status"`
}

// curated183TollRoutes follows the Mobility Authority current-rates sign
// configuration for the US-183 North express lanes. Some displayed routes are
// aggregates of a 183N tolling point plus the 183A Lakeline main lane point.
var curated183TollRoutes = []struct {
	name              string
	tollingPointNames []string
}{
	{
		name:              "183 Northbound: Burnet → Duval/Anderson Mill",
		tollingPointNames: []string{"183N-STNLKBLVDNB-TO-STNLKBLVDNB"},
	},
	{
		name:              "183 Northbound: Burnet → 620/183/45",
		tollingPointNames: []string{"183N-STNLKBLVDNB-TO-MCNLMLNB"},
	},
	{
		name:              "183 Northbound: Burnet → 183A/Avery Ranch",
		tollingPointNames: []string{"183N-STNLKBLVDNB-TO-MCNLMLNB", "183A-LKLNMLNB-TO-LKLNMLNB"},
	},
	{
		name:              "183 Northbound: Duval → 620/183/45",
		tollingPointNames: []string{"183N-MCNLMLNB-TO-MCNLMLNB"},
	},
	{
		name:              "183 Northbound: Duval → 183A/Avery Ranch",
		tollingPointNames: []string{"183N-MCNLMLNB-TO-MCNLMLNB", "183A-LKLNMLNB-TO-LKLNMLNB"},
	},
	{
		name:              "183 Southbound: Avery Ranch → Braker Ln",
		tollingPointNames: []string{"183N-MCNLMLSB-TO-MCNLMLSB"},
	},
	{
		name:              "183 Southbound: Duval → MoPac",
		tollingPointNames: []string{"183N-MPACDCSB-TO-MPACDCSB"},
	},
	{
		name:              "183 Southbound: Duval → 183 South",
		tollingPointNames: []string{"183N-STNLKBLVDSB-TO-STNLKBLVDSB"},
	},
}

// non183TollPointNames maps raw non-183 tolling-point names returned by the
// MobilityAuthority API to human-readable display names. Names not present in
// this map are shown as-is.
var non183TollPointNames = map[string]string{
	// MOPAC Express (LP1X) — northbound
	"LP1X NB: CVZ to 183":     "MoPac Northbound: Cesar Chavez → US-183",
	"LP1X NB: CVZ to Parmer":  "MoPac Northbound: Cesar Chavez → Parmer Ln",
	"LP1X NB: 2222 to Parmer": "MoPac Northbound: FM 2222 → Parmer Ln",
	// MOPAC Express (LP1X) — southbound
	"LP1X SB: Parmer to 2222":    "MoPac Southbound: Parmer Ln → FM 2222",
	"LP1X SB: Parmer to 5th/CVZ": "MoPac Southbound: Parmer Ln → 5th/Cesar Chavez",
	"LP1X SB: 2222 to 5th/CVZ":   "MoPac Southbound: FM 2222 → 5th/Cesar Chavez",
}

func buildTollRates(rawRates []tollRateResponse) []TollRate {
	rates := buildCurated183TollRates(rawRates)
	rates = append(rates, buildNon183TollRates(rawRates)...)
	return rates
}

func buildCurated183TollRates(rawRates []tollRateResponse) []TollRate {
	rawRatesByName := make(map[string]tollRateResponse, len(rawRates))
	for _, rawRate := range rawRates {
		rawRatesByName[rawRate.TollingPointName] = rawRate
	}

	rates := make([]TollRate, 0, len(curated183TollRoutes))
	for _, route := range curated183TollRoutes {
		var tollTagRate float64
		var payByMailRate float64
		skip := false
		routeClosed := false

		for _, tollingPointName := range route.tollingPointNames {
			rawRate, ok := rawRatesByName[tollingPointName]
			if !ok {
				skip = true
				break
			}

			if rawRate.Status == "CLOSED" {
				routeClosed = true
				break
			}

			if rawRate.TripRate <= 0.01 {
				skip = true
				break
			}

			tollTagRate = addCurrency(tollTagRate, rawRate.TripRate)
			payByMailRate = addCurrency(payByMailRate, rawRate.PBMRate)
		}

		if routeClosed {
			rates = append(rates, TollRate{
				Name:   route.name,
				Closed: true,
			})
		} else if !skip {
			rates = append(rates, TollRate{
				Name:          route.name,
				TollTagRate:   tollTagRate,
				PayByMailRate: payByMailRate,
			})
		}
	}

	return rates
}

func buildNon183TollRates(rawRates []tollRateResponse) []TollRate {
	rates := make([]TollRate, 0, len(rawRates))
	for _, rawRate := range rawRates {
		if strings.HasPrefix(rawRate.TollingPointName, "183") {
			continue
		}

		if rawRate.Status == "CLOSED" {
			rates = append(rates, TollRate{
				Name:   prettyTollName(rawRate.TollingPointName),
				Closed: true,
			})
			continue
		}

		if rawRate.TripRate <= 0.01 {
			continue
		}

		rates = append(rates, TollRate{
			Name:          prettyTollName(rawRate.TollingPointName),
			TollTagRate:   rawRate.TripRate,
			PayByMailRate: rawRate.PBMRate,
		})
	}

	sort.SliceStable(rates, func(i, j int) bool {
		return rates[i].Name < rates[j].Name
	})

	return rates
}

func prettyTollName(raw string) string {
	if pretty, ok := non183TollPointNames[raw]; ok {
		return pretty
	}
	return raw
}

func addCurrency(left, right float64) float64 {
	return math.Round((left+right)*100) / 100
}

func NewTollClient(httpClient *http.Client) *TollClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &TollClient{
		url:             EnvOrDefault(EnvTollRatesURL, defaultTollRatesURL),
		facilitiesQuery: DefaultTollFacilitiesQuery,
		httpClient:      httpClient,
		now:             time.Now,
	}
}

func (c *TollClient) FetchTolls(ctx context.Context) (TollData, error) {
	if c == nil {
		return TollData{}, errors.New("toll client is nil")
	}

	requestedAt := c.now()
	chicagoTime := requestedAt

	chicagoLocation, err := time.LoadLocation("America/Chicago")
	if err == nil {
		chicagoTime = requestedAt.In(chicagoLocation)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fields := map[string]string{
		"action":     "express_rates",
		"starttime":  chicagoTime.Format("01/02/2006 15:04"),
		"facilities": c.facilitiesQuery,
	}

	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return TollData{}, fmt.Errorf("write multipart field %q: %w", name, err)
		}
	}

	if err := writer.Close(); err != nil {
		return TollData{}, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, body)
	if err != nil {
		return TollData{}, fmt.Errorf("create toll request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "https://www.mobilityauthority.com")
	req.Header.Set("Referer", "https://www.mobilityauthority.com/pay-your-toll/rates/current-rates/")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return TollData{}, fmt.Errorf("perform toll request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return TollData{}, fmt.Errorf("unexpected toll status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var rawRates []tollRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&rawRates); err != nil {
		return TollData{}, fmt.Errorf("decode toll response: %w", err)
	}

	rates := buildTollRates(rawRates)

	return TollData{
		RequestedAt: requestedAt,
		Rates:       rates,
	}, nil
}
