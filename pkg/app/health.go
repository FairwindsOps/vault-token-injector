package app

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// healthHandler serves a /health endpoint and returns >400 if the vault
// errors are greater than 0. This indicates a general failure.
// TODO: Consider adding additional metrics counts, such as a failure
// at a certain threshold for tfcloud or circle errors
func (a *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	vaultErrorCount := getMetricValue(a.Metrics.vaultErrorCount)

	if vaultErrorCount > 0 {
		w.WriteHeader(http.StatusTeapot)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getMetricValue(col prometheus.Collector) float64 {
	c := make(chan prometheus.Metric, 1) // 1 for metric with no vector
	col.Collect(c)                       // collect current metric value into the channel
	m := dto.Metric{}
	_ = (<-c).Write(&m) // read metric value from the channel
	return *m.Counter.Value
}
