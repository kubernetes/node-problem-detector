/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	kube_rest "k8s.io/client-go/rest"
	kubeClientCmd "k8s.io/client-go/tools/clientcmd"
	kubeClientCmdApi "k8s.io/client-go/tools/clientcmd/api"

	"k8s.io/node-problem-detector/pkg/version"
)

const (
	APIVersion = "v1"

	defaultUseServiceAccount  = false
	defaultServiceAccountFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	defaultInClusterConfig    = true
)

// GlobalKubeClientConfig is the global Kubernetes REST client configuration initialized at startup.
var GlobalKubeClientConfig *kube_rest.Config

func getConfigOverrides(uri *url.URL) (*kubeClientCmd.ConfigOverrides, error) {
	kubeConfigOverride := kubeClientCmd.ConfigOverrides{
		ClusterInfo: kubeClientCmdApi.Cluster{},
	}
	if len(uri.Scheme) != 0 && len(uri.Host) != 0 {
		kubeConfigOverride.ClusterInfo.Server = fmt.Sprintf("%s://%s", uri.Scheme, uri.Host)
	}

	opts := uri.Query()

	if len(opts["insecure"]) > 0 {
		insecure, err := strconv.ParseBool(opts["insecure"][0])
		if err != nil {
			return nil, err
		}
		kubeConfigOverride.ClusterInfo.InsecureSkipTLSVerify = insecure
	}

	return &kubeConfigOverride, nil
}

func getKubeClientConfig(uri *url.URL) (*kube_rest.Config, error) {
	var (
		kubeConfig *kube_rest.Config
		err        error
	)

	opts := uri.Query()
	configOverrides, err := getConfigOverrides(uri)
	if err != nil {
		return nil, err
	}

	inClusterConfig := defaultInClusterConfig
	if len(opts["inClusterConfig"]) > 0 {
		inClusterConfig, err = strconv.ParseBool(opts["inClusterConfig"][0])
		if err != nil {
			return nil, err
		}
	}

	if inClusterConfig {
		kubeConfig, err = kube_rest.InClusterConfig()
		if err != nil {
			return nil, err
		}

		if configOverrides.ClusterInfo.Server != "" {
			kubeConfig.Host = configOverrides.ClusterInfo.Server
		}
		kubeConfig.GroupVersion = &schema.GroupVersion{Version: APIVersion}
		kubeConfig.Insecure = configOverrides.ClusterInfo.InsecureSkipTLSVerify
		if configOverrides.ClusterInfo.InsecureSkipTLSVerify {
			kubeConfig.CAFile = ""
		}
	} else {
		authFile := ""
		if len(opts["auth"]) > 0 {
			authFile = opts["auth"][0]
		}

		if authFile != "" {
			// Load structured kubeconfig data from the given path.
			loader := &kubeClientCmd.ClientConfigLoadingRules{ExplicitPath: authFile}
			loadedConfig, err := loader.Load()
			if err != nil {
				return nil, err
			}

			// Flatten the loaded data to a particular restclient.Config based on the current context.
			if kubeConfig, err = kubeClientCmd.NewNonInteractiveClientConfig(
				*loadedConfig,
				loadedConfig.CurrentContext,
				&kubeClientCmd.ConfigOverrides{},
				loader).ClientConfig(); err != nil {
				return nil, err
			}
		} else {
			kubeConfig = &kube_rest.Config{
				Host: configOverrides.ClusterInfo.Server,
				TLSClientConfig: kube_rest.TLSClientConfig{
					Insecure: configOverrides.ClusterInfo.InsecureSkipTLSVerify,
				},
			}
			kubeConfig.GroupVersion = &schema.GroupVersion{Version: APIVersion}
		}
	}
	if len(kubeConfig.Host) == 0 {
		return nil, fmt.Errorf("invalid kubernetes master url specified")
	}

	useServiceAccount := defaultUseServiceAccount
	if len(opts["useServiceAccount"]) >= 1 {
		useServiceAccount, err = strconv.ParseBool(opts["useServiceAccount"][0])
		if err != nil {
			return nil, err
		}
	}

	if useServiceAccount {
		// If a readable service account token exists, then use it
		if contents, err := os.ReadFile(defaultServiceAccountFile); err == nil {
			kubeConfig.BearerToken = string(contents)
		}
	}

	kubeConfig.ContentType = "application/vnd.kubernetes.protobuf"

	return kubeConfig, nil
}

// NewKubeClientConfig creates a new kube_rest.Config from the given apiserver override URI, QPS, and burst.
func NewKubeClientConfig(apiServerOverride string, qps float32, burst int) (*kube_rest.Config, error) {
	uri, err := url.Parse(apiServerOverride)
	if err != nil {
		return nil, fmt.Errorf("apiserver-override %q is not a valid HTTP URI: %w", apiServerOverride, err)
	}

	cfg, err := getKubeClientConfig(uri)
	if err != nil {
		return nil, err
	}

	if cfg.GroupVersion == nil {
		cfg.GroupVersion = &schema.GroupVersion{Version: APIVersion}
	}
	if cfg.NegotiatedSerializer == nil {
		cfg.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	}

	cfg.UserAgent = fmt.Sprintf("%s/%s", filepath.Base(os.Args[0]), version.Version())
	cfg.QPS = qps
	cfg.Burst = burst
	return cfg, nil
}
