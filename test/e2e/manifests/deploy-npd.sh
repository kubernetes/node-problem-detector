#!/usr/bin/env bash

# Copyright 2026 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Deploys node-problem-detector as a DaemonSet onto a running cluster for the
# GCE cluster e2e jobs, which exercise k8s.io/kubernetes
# test/e2e/node/node_problem_detector.go. The kube-up NPD addon was removed from
# kubernetes/kubernetes, so that test falls back to looking for a
# "node-problem-detector" pod; this script provides it.
#
# The image comes from the per-branch community staging registry, matching the
# node e2e job (see test/build.sh). PULL_BASE_REF is set by Prow to the base
# branch under test and defaults to master for local runs.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly IMAGE="gcr.io/k8s-staging-npd/node-problem-detector"
readonly TAG="${PULL_BASE_REF:-master}"

echo "Deploying node-problem-detector DaemonSet (image tag: ${TAG})..."
kubectl kustomize "${SCRIPT_DIR}" |
  sed -E "s|image: ${IMAGE}:[^[:space:]]+|image: ${IMAGE}:${TAG}|" |
  kubectl apply -f -

echo "Waiting for node-problem-detector DaemonSet to become ready..."
kubectl --namespace=kube-system rollout status daemonset/node-problem-detector --timeout=5m
