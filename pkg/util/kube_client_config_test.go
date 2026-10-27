/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

package util

import (
	"testing"

	"k8s.io/client-go/rest"
)

func TestNewKubeClientConfig(t *testing.T) {
	cfg, err := NewKubeClientConfig("http://127.0.0.1:8080?inClusterConfig=false", 100, 200)
	if err != nil {
		t.Fatalf("NewKubeClientConfig returned an error: %v", err)
	}
	if cfg.Host != "http://127.0.0.1:8080" {
		t.Errorf("expected Host %q, got %q", "http://127.0.0.1:8080", cfg.Host)
	}
	if cfg.QPS != 100 {
		t.Errorf("expected QPS 100, got %v", cfg.QPS)
	}
	if cfg.Burst != 200 {
		t.Errorf("expected Burst 200, got %v", cfg.Burst)
	}
	if cfg.GroupVersion == nil {
		t.Error("expected GroupVersion to be set")
	}
	if cfg.NegotiatedSerializer == nil {
		t.Error("expected NegotiatedSerializer to be set")
	}
	if _, err := rest.RESTClientFor(cfg); err != nil {
		t.Errorf("expected rest.RESTClientFor(cfg) to succeed, got: %v", err)
	}
}
