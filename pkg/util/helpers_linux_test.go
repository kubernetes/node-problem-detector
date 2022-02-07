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

package util

import (
	"testing"
)

func TestGetOSVersionLinux(t *testing.T) {
	testCases := []struct {
		name              string
		fakeOSReleasePath string
		expectedOSVersion string
		expectErr         bool
	}{
		{
			name:              "COS",
			fakeOSReleasePath: "testdata/os-release-cos",
			expectedOSVersion: "cos 77-12293.0.0",
			expectErr:         false,
		},
		{
			name:              "Debian",
			fakeOSReleasePath: "testdata/os-release-debian",
			expectedOSVersion: "debian 9 (stretch)",
			expectErr:         false,
		},
		{
			name:              "Ubuntu",
			fakeOSReleasePath: "testdata/os-release-ubuntu",
			expectedOSVersion: "ubuntu 16.04.6 LTS (Xenial Xerus)",
			expectErr:         false,
		},
		{
			name:              "centos",
			fakeOSReleasePath: "testdata/os-release-centos",
			expectedOSVersion: "centos 7 (Core)",
			expectErr:         false,
		},
		{
			name:              "rhel",
			fakeOSReleasePath: "testdata/os-release-rhel",
			expectedOSVersion: "rhel 7.7 (Maipo)",
			expectErr:         false,
		},
		{
			name:              "fedora",
			fakeOSReleasePath: "testdata/os-release-fedora",
			expectedOSVersion: "35 (Thirty Five)",
			expectErr:         false,
		},
		{
			name:              "Unknown",
			fakeOSReleasePath: "testdata/os-release-unknown",
			expectedOSVersion: "",
			expectErr:         true,
		},
		{
			name:              "Empty",
			fakeOSReleasePath: "testdata/os-release-empty",
			expectedOSVersion: "",
			expectErr:         true,
		},
	}

	for _, tt := range testCases {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			osVersion, err := getOSVersion(tc.fakeOSReleasePath)

			if tc.expectErr && err == nil {
				t.Errorf("Expect to get error, but got no returned error.")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expect to get no error, but got returned error: %v", err)
			}
			if !tc.expectErr && osVersion != tc.expectedOSVersion {
				t.Errorf("Wanted: %+v. \nGot: %+v", tc.expectedOSVersion, osVersion)
			}
		})
	}
}
