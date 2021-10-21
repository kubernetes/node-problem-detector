package podrestartmonitor

import (
	"reflect"
	"testing"
	"time"
)

func Test_parseConfig(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantConfig MonitorConfig
		wantErr    bool
	}{
		{name: "defaults", wantConfig: MonitorConfig{
			Namespace:        "kube-system",
			PodSelector:      "foo=bar",
			RestartThreshold: 5,
			CheckInterval:    5 * time.Minute,
			ConditionName:    ConditionTooManyRestarts,
		}, raw: `{"PodSelector": "foo=bar"}`},
		{name: "bad interval", wantErr: true, raw: `{"PodSelector": "foo=bar", "CheckInterval": "1 month"}`},
		{name: "bad selector", wantErr: true, raw: `{"PodSelector": "foo is bar"}`},
		{name: "missing selector", wantErr: true, raw: `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotConfig, err := parseConfig([]byte(tt.raw))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(gotConfig, tt.wantConfig) {
				t.Errorf("parseConfig() gotConfig = %v, want %v", gotConfig, tt.wantConfig)
			}
		})
	}
}
