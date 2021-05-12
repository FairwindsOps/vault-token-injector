package app

import (
	"reflect"
	"testing"
)

func TestNewApp(t *testing.T) {
	type args struct {
		circleToken string
	}
	tests := []struct {
		name string
		args args
		want *App
	}{
		{
			name: "basic",
			args: args{
				circleToken: "foo",
			},
			want: &App{
				CircleToken: "foo",
				Config:      &Config{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewApp(tt.args.circleToken); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewApp() = %v, want %v", got, tt.want)
			}
		})
	}
}
