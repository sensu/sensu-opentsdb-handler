package main

import (
	"testing"

	corev2 "github.com/sensu/sensu-go/api/core/v2"
)

func TestMetricPointToOpenTSDBString(t *testing.T) {
	tests := []struct {
		name       string
		config     HandlerConfig
		point      *corev2.MetricPoint
		entityName string
		expected   string
	}{
		{
			name:     "nil point gives empty string",
			point:    nil,
			expected: "",
		},
		{
			name: "spaces are replaced by SpaceReplacement option",
			config: HandlerConfig{
				SpaceReplacement: "-",
			},
			point: &corev2.MetricPoint{
				Name:      "cpu.idle",
				Value:     42,
				Timestamp: 1337,
				Tags:      []*corev2.MetricTag{&corev2.MetricTag{Name: "data center", Value: "us east 1"}},
			},
			expected: "put cpu.idle 1337 42 data-center=us-east-1\n",
		},
		{
			name: "prefix is properly applied",
			config: HandlerConfig{
				Prefix: "prefix",
			},
			point: &corev2.MetricPoint{
				Name:      "cpu.idle",
				Value:     42,
				Timestamp: 1337,
				Tags:      []*corev2.MetricTag{&corev2.MetricTag{Name: "dc", Value: "us-east-1"}},
			},
			expected: "put prefix.cpu.idle 1337 42 dc=us-east-1\n",
		},
		{
			name: "entity name prefix is properly applied",
			config: HandlerConfig{
				PrefixEntityName: true,
			},
			point: &corev2.MetricPoint{
				Name:      "cpu.idle",
				Value:     42,
				Timestamp: 1337,
				Tags:      []*corev2.MetricTag{&corev2.MetricTag{Name: "dc", Value: "us-east-1"}},
			},
			entityName: "webserver0",
			expected:   "put webserver0.cpu.idle 1337 42 dc=us-east-1\n",
		},
		{
			name: "both prefix and entity name prefix are properly applied",
			config: HandlerConfig{
				Prefix:           "prefix",
				PrefixEntityName: true,
			},
			point: &corev2.MetricPoint{
				Name:      "cpu.idle",
				Value:     42,
				Timestamp: 1337,
				Tags:      []*corev2.MetricTag{&corev2.MetricTag{Name: "dc", Value: "us-east-1"}},
			},
			entityName: "webserver0",
			expected:   "put prefix.webserver0.cpu.idle 1337 42 dc=us-east-1\n",
		},
		{
			name: "host tag is properly applied",
			config: HandlerConfig{
				TagHost: true,
			},
			point: &corev2.MetricPoint{
				Name:      "cpu.idle",
				Value:     42,
				Timestamp: 1337,
			},
			entityName: "webserver0",
			expected:   "put cpu.idle 1337 42 host=webserver0\n",
		},
		{
			name:   "host tag added if no other tags",
			config: HandlerConfig{},
			point: &corev2.MetricPoint{
				Name:      "cpu.idle",
				Value:     42,
				Timestamp: 1337,
			},
			entityName: "webserver0",
			expected:   "put cpu.idle 1337 42 host=webserver0\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handlerConfig = test.config
			result := MetricPointToOpenTSDBString(test.point, test.entityName)

			if result != test.expected {
				t.Errorf("got %s, expected %s", result, test.expected)
			}
		})
	}
}
