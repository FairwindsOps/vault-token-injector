package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
					TokenVariable: "VAULT_TOKEN",
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
					TokenVariable: "FOO",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewApp(tt.args.circleToken, tt.args.vaultTokenFile, tt.args.tfCloudToken, tt.args.config)
			assert.EqualValues(t, tt.want, got)
		})
	}
}
