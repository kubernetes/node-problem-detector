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

package options

const (
	supportedComponentsFlagMessage = "The component to check health for. Supports kubelet, docker, kube-proxy, and cri"
	supportedServicesFlagMessage   = "The underlying service responsible for the component. Set to the corresponding component for docker and kubelet, containerd for cri."
	validComponentMessage          = "the component specified is not supported. Supported components are: <kubelet/docker/cri/kube-proxy>"
)
