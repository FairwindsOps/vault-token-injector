package app

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	totalErrorCount      prometheus.Counter
	vaultErrorCount      prometheus.Counter
	circleCIErrorCount   prometheus.Counter
	circleTokensUpdated  prometheus.Counter
	tfCloudErrorCount    prometheus.Counter
	tfcloudTokensUpdated prometheus.Counter
}

func (a *App) registerMetrics() {
	a.Metrics = &Metrics{
		totalErrorCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "vault_token_injector_errors_total",
			Help: "The number of errors encountered",
		}),
		vaultErrorCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "vault_token_injector_vault_errors_total",
			Help: "The number of errors encountered when calling the Vault API",
		}),
		circleCIErrorCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "vault_token_injector_circleci_errors_total",
			Help: "The number of errors encountered when calling the CircleCI API",
		}),
		circleTokensUpdated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "vault_token_injector_circle_tokens_updated",
			Help: "The number of CircleCI tokens updated",
		}),
		tfCloudErrorCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "vault_token_injector_tfcloud_errors_total",
			Help: "The number of errors encountered when calling the TFCloud API",
		}),
		tfcloudTokensUpdated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "vault_token_injector_tfcloud_tokens_updated",
			Help: "The number of TFCloud tokens updated",
		}),
	}
}

func (a App) incrementVaultError() {
	a.Metrics.vaultErrorCount.Inc()
	a.Metrics.totalErrorCount.Inc()
}

func (a App) incrementTfCloudError() {
	a.Metrics.tfCloudErrorCount.Inc()
	a.Metrics.totalErrorCount.Inc()
}

func (a App) incremenCircleCIError() {
	a.Metrics.circleCIErrorCount.Inc()
	a.Metrics.totalErrorCount.Inc()
}
