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

func TestTollClientFetchTollsWithMockedMobilityAuthorityResponse(t *testing.T) {

	expectedChicagoStartTime := "04/25/2026 12:45"
	requestedAt := time.Date(2026, time.April, 25, 17, 45, 30, 0, time.UTC)

	var requestSeen bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestSeen = true

		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "https://www.mobilityauthority.com", r.Header.Get("Origin"))
		assert.Equal(t, "https://www.mobilityauthority.com/pay-your-toll/rates/current-rates/", r.Header.Get("Referer"))

		err := r.ParseMultipartForm(1 << 20)
		assert.NoError(t, err, "ParseMultipartForm returned error")

		assert.Equal(t, "express_rates", r.FormValue("action"))
		assert.Equal(t, DefaultTollFacilitiesQuery, r.FormValue("facilities"))
		assert.Equal(t, expectedChicagoStartTime, r.FormValue("starttime"))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{
				"tollingPointName": "183N-STNLKBLVDNB-TO-STNLKBLVDNB",
				"tripRate": 0.73,
				"pbmRate": 1.10
			},
			{
				"tollingPointName": "183N-STNLKBLVDNB-TO-MCNLMLNB",
				"tripRate": 1.46,
				"pbmRate": 2.20
			},
			{
				"tollingPointName": "183A-LKLNMLNB-TO-LKLNMLNB",
				"tripRate": 0.87,
				"pbmRate": 1.31
			},
			{
				"tollingPointName": "183N-MCNLMLNB-TO-MCNLMLNB",
				"tripRate": 0.73,
				"pbmRate": 1.10
			},
			{
				"tollingPointName": "183N-MCNLMLSB-TO-MCNLMLSB",
				"tripRate": 0.73,
				"pbmRate": 1.10
			},
			{
				"tollingPointName": "183N-MPACDCSB-TO-MPACDCSB",
				"tripRate": 0.73,
				"pbmRate": 1.10
			},
			{
				"tollingPointName": "183N-STNLKBLVDSB-TO-STNLKBLVDSB",
				"tripRate": 0.73,
				"pbmRate": 1.10
			}
		]`)
	}))
	defer server.Close()

	t.Setenv(EnvTollRatesURL, server.URL)

	client := NewTollClient(server.Client())
	client.now = func() time.Time {
		return requestedAt
	}

	got, err := client.FetchTolls(t.Context())
	require.NoError(t, err)

	require.True(t, requestSeen, "mock toll endpoint was not called")

	assert.True(t, got.RequestedAt.Equal(requestedAt), "RequestedAt = %s, want %s", got.RequestedAt, requestedAt)

	expectedRates := []TollRate{
		{
			Name:          "183 Northbound: Burnet → Duval/Anderson Mill",
			TollTagRate:   0.73,
			PayByMailRate: 1.10,
		},
		{
			Name:          "183 Northbound: Burnet → 620/183/45",
			TollTagRate:   1.46,
			PayByMailRate: 2.20,
		},
		{
			Name:          "183 Northbound: Burnet → 183A/Avery Ranch",
			TollTagRate:   2.33,
			PayByMailRate: 3.51,
		},
		{
			Name:          "183 Northbound: Duval → 620/183/45",
			TollTagRate:   0.73,
			PayByMailRate: 1.10,
		},
		{
			Name:          "183 Northbound: Duval → 183A/Avery Ranch",
			TollTagRate:   1.60,
			PayByMailRate: 2.41,
		},
		{
			Name:          "183 Southbound: Avery Ranch → Braker Ln",
			TollTagRate:   0.73,
			PayByMailRate: 1.10,
		},
		{
			Name:          "183 Southbound: Duval → MoPac",
			TollTagRate:   0.73,
			PayByMailRate: 1.10,
		},
		{
			Name:          "183 Southbound: Duval → 183 South",
			TollTagRate:   0.73,
			PayByMailRate: 1.10,
		},
	}

	require.Len(t, got.Rates, len(expectedRates))

	for i, want := range expectedRates {
		assert.Equal(t, want.Name, got.Rates[i].Name, "Rates[%d].Name", i)
		assert.Equal(t, want.TollTagRate, got.Rates[i].TollTagRate, "Rates[%d].TollTagRate", i)
		assert.Equal(t, want.PayByMailRate, got.Rates[i].PayByMailRate, "Rates[%d].PayByMailRate", i)
		assert.Equal(t, want.Closed, got.Rates[i].Closed, "Rates[%d].Closed", i)
	}
}

func TestTollClientFetchTollsClosedRoutes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{
				"tollingPointName": "183N-STNLKBLVDNB-TO-STNLKBLVDNB",
				"tripRate": 0.73,
				"pbmRate": 1.10,
				"status": "OPEN"
			},
			{
				"tollingPointName": "183N-STNLKBLVDNB-TO-MCNLMLNB",
				"tripRate": 0,
				"pbmRate": 0,
				"status": "CLOSED"
			},
			{
				"tollingPointName": "183N-MCNLMLNB-TO-MCNLMLNB",
				"tripRate": 0.73,
				"pbmRate": 1.10,
				"status": "OPEN"
			},
			{
				"tollingPointName": "183A-LKLNMLNB-TO-LKLNMLNB",
				"tripRate": 0.87,
				"pbmRate": 1.31,
				"status": "OPEN"
			},
			{
				"tollingPointName": "183N-MCNLMLSB-TO-MCNLMLSB",
				"tripRate": 0.73,
				"pbmRate": 1.10,
				"status": "OPEN"
			},
			{
				"tollingPointName": "183N-MPACDCSB-TO-MPACDCSB",
				"tripRate": 0.73,
				"pbmRate": 1.10,
				"status": "OPEN"
			},
			{
				"tollingPointName": "183N-STNLKBLVDSB-TO-STNLKBLVDSB",
				"tripRate": 0.73,
				"pbmRate": 1.10,
				"status": "OPEN"
			},
			{
				"tollingPointName": "LP1X NB: CVZ to 183",
				"tripRate": 1.25,
				"pbmRate": 1.88,
				"status": "OPEN"
			},
			{
				"tollingPointName": "LP1X SB: Parmer to 2222",
				"tripRate": 0,
				"pbmRate": 0,
				"status": "CLOSED"
			}
		]`)
	}))
	defer server.Close()

	t.Setenv(EnvTollRatesURL, server.URL)

	client := NewTollClient(server.Client())
	client.now = func() time.Time {
		return time.Date(2026, time.April, 25, 17, 45, 30, 0, time.UTC)
	}

	got, err := client.FetchTolls(t.Context())
	require.NoError(t, err)

	// Build a map of results by name for easier lookup.
	byName := make(map[string]TollRate, len(got.Rates))
	for _, rate := range got.Rates {
		byName[rate.Name] = rate
	}

	// 183 route whose segment is present and OPEN — should appear normally.
	r, ok := byName["183 Northbound: Burnet → Duval/Anderson Mill"]
	assert.True(t, ok, "expected open 183 route to be present: 183 Northbound: Burnet → Duval/Anderson Mill")
	if ok {
		assert.False(t, r.Closed, "183 Northbound: Burnet → Duval/Anderson Mill: Closed = true, want false")
		assert.Equal(t, 0.73, r.TollTagRate, "183 Northbound: Burnet → Duval/Anderson Mill: TollTagRate")
		assert.Equal(t, 1.10, r.PayByMailRate, "183 Northbound: Burnet → Duval/Anderson Mill: PayByMailRate")
	}

	// 183 route whose first segment is CLOSED — should appear with Closed: true and zero rates.
	r, ok = byName["183 Northbound: Burnet → 620/183/45"]
	assert.True(t, ok, "expected closed 183 route to be present: 183 Northbound: Burnet → 620/183/45")
	if ok {
		assert.True(t, r.Closed, "183 Northbound: Burnet → 620/183/45: Closed = false, want true")
		assert.Equal(t, 0.0, r.TollTagRate, "183 Northbound: Burnet → 620/183/45: TollTagRate")
		assert.Equal(t, 0.0, r.PayByMailRate, "183 Northbound: Burnet → 620/183/45: PayByMailRate")
	}

	// 183 multi-segment route whose first segment is CLOSED — should also appear with Closed: true.
	r, ok = byName["183 Northbound: Burnet → 183A/Avery Ranch"]
	assert.True(t, ok, "expected closed 183 multi-segment route to be present: 183 Northbound: Burnet → 183A/Avery Ranch")
	if ok {
		assert.True(t, r.Closed, "183 Northbound: Burnet → 183A/Avery Ranch: Closed = false, want true")
	}

	// 183 route whose segment is entirely missing from the response — should be omitted.
	// "183N-STNLKBLVDNB-TO-MCNLMLNB" was the only route referencing a missing point;
	// here we rely on a route that would require a tolling point not in the payload at all.
	// The route "183 Northbound: Duval → 183A/Avery Ranch" depends on 183N-MCNLMLNB-TO-MCNLMLNB
	// (present) and 183A-LKLNMLNB-TO-LKLNMLNB (present), so it should appear normally.
	r, ok = byName["183 Northbound: Duval → 183A/Avery Ranch"]
	assert.True(t, ok, "expected open multi-segment 183 route to be present: 183 Northbound: Duval → 183A/Avery Ranch")
	assert.False(t, r.Closed, "183 Northbound: Duval → 183A/Avery Ranch: Closed = true, want false")

	// Non-183 OPEN route — should appear normally.
	r, ok = byName["MoPac Northbound: Cesar Chavez → US-183"]
	assert.True(t, ok, "expected open non-183 route to be present: MoPac Northbound: Cesar Chavez → US-183")
	assert.False(t, r.Closed, "MoPac Northbound: Cesar Chavez → US-183: Closed = true, want false")
	assert.Equal(t, 1.25, r.TollTagRate, "MoPac Northbound: Cesar Chavez → US-183: TollTagRate")

	// Non-183 CLOSED route — should appear with Closed: true and zero rates.
	r, ok = byName["MoPac Southbound: Parmer Ln → FM 2222"]
	assert.True(t, ok, "expected closed non-183 route to be present: MoPac Southbound: Parmer Ln → FM 2222")
	assert.True(t, r.Closed, "MoPac Southbound: Parmer Ln → FM 2222: Closed = false, want true")
	assert.Equal(t, 0.0, r.TollTagRate, "MoPac Southbound: Parmer Ln → FM 2222: TollTagRate")
	assert.Equal(t, 0.0, r.PayByMailRate, "MoPac Southbound: Parmer Ln → FM 2222: PayByMailRate")
}

func TestTollClientFetchTollsReturnsHTTPError(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporary upstream failure", http.StatusBadGateway)
	}))
	defer server.Close()

	t.Setenv(EnvTollRatesURL, server.URL)

	client := NewTollClient(server.Client())

	_, err := client.FetchTolls(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "unexpected toll status 502")
}

func TestTollClientFetchTollsReturnsDecodeErrorForMalformedPayload(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{not-json`)
	}))
	defer server.Close()

	t.Setenv(EnvTollRatesURL, server.URL)

	client := NewTollClient(server.Client())

	_, err := client.FetchTolls(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "decode toll response")
}

func TestTollSamplePayloadShape(t *testing.T) {

	const samplePayload = `[
		{
			"tollingPointName": "183N-STNLKBLVDNB-TO-STNLKBLVDNB",
			"tripRate": 0.73,
			"pbmRate": 1.10
		}
	]`

	var decoded []tollRateResponse
	err := json.Unmarshal([]byte(samplePayload), &decoded)
	require.NoError(t, err, "sample toll payload failed to decode")

	require.Len(t, decoded, 1)
	assert.Equal(t, "183N-STNLKBLVDNB-TO-STNLKBLVDNB", decoded[0].TollingPointName)
	assert.Equal(t, 0.73, decoded[0].TripRate)
	assert.Equal(t, 1.10, decoded[0].PBMRate)
}
