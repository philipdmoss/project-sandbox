package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

type upstreamResponse struct {
	Status  string `json:"status"`
	Results struct {
		Sunrise            string `json:"sunrise"`
		Sunset             string `json:"sunset"`
		SolarNoon          string `json:"solar_noon"`
		DayLength          int    `json:"day_length"`
		CivilTwilightBegin string `json:"civil_twilight_begin"`
		CivilTwilightEnd   string `json:"civil_twilight_end"`
	} `json:"results"`
}

type goldenHour struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Duration string `json:"duration"`
}

type SunTimesResponse struct {
	Status            string     `json:"status"`
	Sunrise           string     `json:"sunrise"`
	Sunset            string     `json:"sunset"`
	SolarNoon         string     `json:"solar_noon"`
	DayLength         string     `json:"day_length"`
	MorningGoldenHour goldenHour `json:"morning_golden_hour"`
	EveningGoldenHour goldenHour `json:"evening_golden_hour"`
}

const sunApiEndpoint = "https://api.sunrise-sunset.org/json"

func fetchSunTimes(lat, lng string) (upstreamResponse, error) {
	var parsed upstreamResponse

	query := url.Values{}
	query.Set("lat", lat)
	query.Set("lng", lng)
	query.Set("formatted", "0")
	requestURL := sunApiEndpoint + "?" + query.Encode()

	resp, err := http.Get(requestURL)
	if err != nil {
		return parsed, fmt.Errorf("contacting upstream API: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return parsed, fmt.Errorf("decoding upstream response: %w", err)
	}
	if parsed.Status != "OK" {
		return parsed, fmt.Errorf("upstream API returned status %q", parsed.Status)
	}
	return parsed, nil
}

func approximateGoldenHour(horizonCrossing, civilTwilightEdge time.Time, morning bool) goldenHour {
	twilightSpan := horizonCrossing.Sub(civilTwilightEdge)
	if twilightSpan < 0 {
		twilightSpan = -twilightSpan
	}

	var start, end time.Time
	if morning {
		start = horizonCrossing
		end = horizonCrossing.Add(twilightSpan)
	} else {
		start = horizonCrossing.Add(-twilightSpan)
		end = horizonCrossing
	}

	return goldenHour{
		Start:    start.Format(time.RFC3339),
		End:      end.Format(time.RFC3339),
		Duration: twilightSpan.String(),
	}
}

func sunTimesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	lat := r.URL.Query().Get("lat")
	lng := r.URL.Query().Get("lng")
	if lat == "" || lng == "" {
		http.Error(w, `{"error": "Missing 'lat' or 'lng' parameters"}`, http.StatusBadRequest)
		return
	}

	upstream, err := fetchSunTimes(lat, lng)
	if err != nil {
		log.Printf("sun times lookup failed for lat=%s lng=%s: %v", lat, lng, err)
		http.Error(w, `{"error": "Failed to retrieve sun times"}`, http.StatusBadGateway)
		return
	}

	sunrise, sunriseErr := time.Parse(time.RFC3339, upstream.Results.Sunrise)
	sunset, sunsetErr := time.Parse(time.RFC3339, upstream.Results.Sunset)
	solarNoon, solarNoonErr := time.Parse(time.RFC3339, upstream.Results.SolarNoon)
	civilTwilightBegin, civilBeginErr := time.Parse(time.RFC3339, upstream.Results.CivilTwilightBegin)
	civilTwilightEnd, civilEndErr := time.Parse(time.RFC3339, upstream.Results.CivilTwilightEnd)
	if sunriseErr != nil || sunsetErr != nil || solarNoonErr != nil || civilBeginErr != nil || civilEndErr != nil {
		log.Printf("failed to parse upstream timestamps for lat=%s lng=%s: sunrise=%v sunset=%v solarNoon=%v civilBegin=%v civilEnd=%v",
			lat, lng, sunriseErr, sunsetErr, solarNoonErr, civilBeginErr, civilEndErr)
		http.Error(w, `{"error": "Unexpected upstream response format"}`, http.StatusBadGateway)
		return
	}

	response := SunTimesResponse{
		Status:            "success",
		Sunrise:           sunrise.Format(time.RFC3339),
		Sunset:            sunset.Format(time.RFC3339),
		SolarNoon:         solarNoon.Format(time.RFC3339),
		DayLength:         (time.Duration(upstream.Results.DayLength) * time.Second).String(),
		MorningGoldenHour: approximateGoldenHour(sunrise, civilTwilightBegin, true),
		EveningGoldenHour: approximateGoldenHour(sunset, civilTwilightEnd, false),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to encode response for lat=%s lng=%s: %v", lat, lng, err)
	}
}

func main() {
	http.HandleFunc("/api/suntimes", sunTimesHandler)
	log.Println("sun-calculator-be listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
