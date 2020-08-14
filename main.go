package main

import (
    "log"
	"net/http"
	"net/http/httptrace"
	"fmt"
	"time"
	"strconv"
	"encoding/json"
	"math"
    "github.com/gorilla/mux"
)

func MainHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var durstring []string
	vars := mux.Vars(r)

	target_url := vars["protocol"] + "://" + vars["host"]
	if !(vars["protocol"] == "http" || vars["protocol"] == "https") {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"message": "%v protocol is not supported, please provide http or https!"}`, vars["protocol"])))
		return
	}
	samples, err := strconv.Atoi(vars["samples"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"message": "%v provided in samples query parameter, please provide a valid number!"}`, vars["samples"])))
		return
	}
	if samples > 50 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"message": "%v samples provided, max 50 samples allowed"}`, vars["samples"])))
		return
	}

	durations, average, err := timeGet(target_url, samples)
	if err != nil {
		w.WriteHeader(http.StatusRequestTimeout)
		w.Write([]byte(fmt.Sprintf(`{"message": "%v unreachable"}`, target_url)))
		return
	}
	for _, v := range durations {
		s := fmt.Sprintf("%g", v) + "ms"
		durstring = append(durstring, s)
	}

	durJson, _ := json.Marshal(durstring)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"host": "%s", "protocol": "%s", "results": {"measurements": %v, "averageLatency": "%vms"}}`, vars["host"], vars["protocol"], string(durJson), average)))
}

func timeGet(url string, count int) ([]float64, float64, error) {
	var (
		start time.Time
		durations []float64
		total float64
		average float64
	)

	for i := 0; i < count; i++ {
        req, _ := http.NewRequest("GET", url, nil)

        trace := &httptrace.ClientTrace{
            GotFirstResponseByte: func() {
				duration := float64(time.Since(start).Microseconds())/1000
				durations = append(durations, duration)
				total += duration
			},
		}

		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
        start = time.Now()
        if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
			return durations, average, err
        }
	}
	average = math.Round((total / (float64(count))) * 1000) / 1000
	return durations, average, nil
}

func main() {
    r := mux.NewRouter()
	r.HandleFunc("/measure", MainHandler).
	Queries("host", "{host}", "protocol", "{protocol}", "samples", "{samples}").
	Methods(http.MethodGet)
	
	log.Fatal(http.ListenAndServe(":8080", r))
}