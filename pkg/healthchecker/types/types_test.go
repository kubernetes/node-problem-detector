/*
Copyright 2021 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogPatternFlag(t *testing.T) {
	testCases := []struct {
		name                       string
		value                      string
		expectedStringVal          string
		expectedLogPatternCountMap map[string]int
		expectSetError             bool
	}{
		{
			name:                       "valid single flag value",
			value:                      "10:pattern1",
			expectedStringVal:          "pattern1:10",
			expectedLogPatternCountMap: map[string]int{"pattern1": 10},
			expectSetError:             false,
		},
		{
			name:                       "valid multiple flag values",
			value:                      "10:pattern1,20:pattern2",
			expectedStringVal:          "pattern1:10 pattern2:20",
			expectedLogPatternCountMap: map[string]int{"pattern1": 10, "pattern2": 20},
			expectSetError:             false,
		},
		{
			name:           "empty log pattern",
			value:          "10:",
			expectSetError: true,
		},
		{
			name:           "0 failure threshold count",
			value:          "0:pattern1",
			expectSetError: true,
		},
		{
			name:           "empty failure threshold count",
			value:          ":pattern1",
			expectSetError: true,
		},
		{
			name:           "empty failure threshold count and pattern",
			value:          ":",
			expectSetError: true,
		},
		{
			name:           "non integer value in failure threshold",
			value:          "notAnInteger:pattern1",
			expectSetError: true,
		},
		{
			name:                       "valid log pattern with ':'",
			value:                      "10:pattern1a:pattern1b,20:pattern2",
			expectedStringVal:          "pattern1a:pattern1b:10 pattern2:20",
			expectedLogPatternCountMap: map[string]int{"pattern1a:pattern1b": 10, "pattern2": 20},
			expectSetError:             false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			flag := LogPatternFlag{}
			err := flag.Set(test.value)
			if test.expectSetError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				actualStringVal := flag.String()
				actualLogPatternCountMap := flag.GetLogPatternCountMap()
				assert.Equal(t, test.expectedStringVal, actualStringVal)
				if !reflect.DeepEqual(test.expectedLogPatternCountMap, actualLogPatternCountMap) {
					t.Fatalf("logPatternCountMap mismatch, expected: %v, actual: %v", test.expectedLogPatternCountMap, actualLogPatternCountMap)
				}
				assert.Equal(t, test.expectedLogPatternCountMap, actualLogPatternCountMap)
			}
		})
	}
}

func TestKubeEndpointConfiguration(t *testing.T) {
	testCases := []struct {
		name                      string
		envConfig                 map[string]string
		expectedKubeletEndpoint   string
		expectedKubeProxyEndpoint string
	}{
		{
			name:                      "no overrides supplied",
			envConfig:                 map[string]string{},
			expectedKubeletEndpoint:   "http://127.0.0.1:10248/healthz",
			expectedKubeProxyEndpoint: "http://127.0.0.1:10256/healthz",
		}, {
			name: "HOST_ADDRESS override supplied",
			envConfig: map[string]string{
				"HOST_ADDRESS": "samplehost.testdomain.com",
			},
			expectedKubeletEndpoint:   "http://samplehost.testdomain.com:10248/healthz",
			expectedKubeProxyEndpoint: "http://samplehost.testdomain.com:10256/healthz",
		},
		{
			name: "KUBELET_PORT override supplied",
			envConfig: map[string]string{
				"KUBELET_PORT": "12345",
			},
			expectedKubeletEndpoint:   "http://127.0.0.1:12345/healthz",
			expectedKubeProxyEndpoint: "http://127.0.0.1:10256/healthz",
		},
		{
			name: "KUBEPROXY_PORT override supplied",
			envConfig: map[string]string{
				"KUBEPROXY_PORT": "12345",
			},
			expectedKubeletEndpoint:   "http://127.0.0.1:10248/healthz",
			expectedKubeProxyEndpoint: "http://127.0.0.1:12345/healthz",
		},
		{
			name: "HOST_ADDRESS and KUBELET_PORT override supplied",
			envConfig: map[string]string{
				"HOST_ADDRESS": "samplehost.testdomain.com",
				"KUBELET_PORT": "12345",
			},
			expectedKubeletEndpoint:   "http://samplehost.testdomain.com:12345/healthz",
			expectedKubeProxyEndpoint: "http://samplehost.testdomain.com:10256/healthz",
		},
		{
			name: "HOST_ADDRESS and KUBEPROXY_PORT override supplied",
			envConfig: map[string]string{
				"HOST_ADDRESS":   "samplehost.testdomain.com",
				"KUBEPROXY_PORT": "12345",
			},
			expectedKubeletEndpoint:   "http://samplehost.testdomain.com:10248/healthz",
			expectedKubeProxyEndpoint: "http://samplehost.testdomain.com:12345/healthz",
		},
		{
			name: "HOST_ADDRESS, KUBELET_PORT and KUBEPROXY_PORT override supplied",
			envConfig: map[string]string{
				"HOST_ADDRESS":   "10.0.10.1",
				"KUBELET_PROXY":  "12345",
				"KUBEPROXY_PORT": "12346",
			},
			expectedKubeletEndpoint:   "http://10.0.10.1:12345/healthz",
			expectedKubeProxyEndpoint: "http://10.0.10.1:12346/healthz",
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			for key, val := range test.envConfig {
				t.Setenv(key, val)
			}
			kubeProxyHCEndpoint := KubeProxyHealthCheckEndpoint()
			kubeletHCEndpoint := KubeletHealthCheckEndpoint()
			setKubeEndpoints()
			assert.Equal(t, kubeProxyHCEndpoint, test.expectedKubeProxyEndpoint)
			assert.Equal(t, kubeletHCEndpoint, test.expectedKubeletEndpoint)
		})
	}
}
