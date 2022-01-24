package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// var TestErrorsConfig = &Errors{
// 	totalErrorCount: promauto.NewCounter(prometheus.CounterOpts{
// 		Name: "vault_token_injector_errors_total",
// 		Help: "The number of errors encountered",
// 	}),
// 	vaultErrorCount: promauto.NewCounter(prometheus.CounterOpts{
// 		Name: "vault_token_injector_vault_errors_total",
// 		Help: "The number of errors encountered when calling the Vault API",
// 	}),
// 	circleCIErrorCount: promauto.NewCounter(prometheus.CounterOpts{
// 		Name: "vault_token_injector_circleci_errors_total",
// 		Help: "The number of errors encountered when calling the CircleCI API",
// 	}),
// 	tfCloudErrorCount: promauto.NewCounter(prometheus.CounterOpts{
// 		Name: "vault_token_injector_tfcloud_errors_total",
// 		Help: "The number of errors encountered when calling the TFCloud API",
// 	}),
// }

func TestNewApp(t *testing.T) {
	type args struct {
		circleToken    string
		vaultTokenFile string
		tfCloudToken   string
		config         *Config
	}
	tests := []struct {
		name string
		args args
		want *App
	}{
		{
			name: "basic",
			args: args{
				circleToken:    "foo",
				vaultTokenFile: "",
				tfCloudToken:   "farglebargle",
				config:         &Config{},
			},
			want: &App{
				CircleToken:    "foo",
				VaultTokenFile: "",
				TFCloudToken:   "farglebargle",
				Config: &Config{
					TokenVariable:        "VAULT_TOKEN",
					TokenTTL:             time.Minute * 60,
					TokenRefreshInterval: time.Minute * 30,
				},
			},
		},
		{
			name: "override token variable",
			args: args{
				circleToken:    "foo",
				vaultTokenFile: "",
				tfCloudToken:   "farglebargle",
				config: &Config{
					TokenVariable: "FOO",
				},
			},
			want: &App{
				CircleToken:    "foo",
				VaultTokenFile: "",
				TFCloudToken:   "farglebargle",
				Config: &Config{
					TokenVariable:        "FOO",
					TokenTTL:             time.Minute * 60,
					TokenRefreshInterval: time.Minute * 30,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewApp(tt.args.circleToken, tt.args.vaultTokenFile, tt.args.tfCloudToken, tt.args.config, false)
			assert.EqualValues(t, tt.want, got)
		})
	}
}
