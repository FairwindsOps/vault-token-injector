package app

import (
	"reflect"
	"testing"
)

func TestNewApp(t *testing.T) {
	type args struct {
		circleToken    string
		vaultTokenFile string
		tfCloudToken   string
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
			},
			want: &App{
				CircleToken:    "foo",
				VaultTokenFile: "",
				TFCloudToken:   "farglebargle",
				Config:         &Config{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewApp(tt.args.circleToken, tt.args.vaultTokenFile, tt.args.tfCloudToken); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewApp() = %v, want %v", got, tt.want)
			}
		})
	}
}
