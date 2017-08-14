// Copyright 2017 CoreOS Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"testing"
	"text/template"
)

func TestUrlFor(t *testing.T) {
	tests := []struct {
		desc           string
		urlTemplate    string
		OSChannel      string
		OSVersion      string
		OSArch         string
		AddonName      string
		AddonReference string
		out            string
		haserr         bool
	}{
		{
			"CL URL",
			StoreTemplate,
			"stable",
			"1490.1.0",
			"amd64",
			"docker",
			"1.12",
			"https://stable.release.core-os.net/amd64-usr/1490.1.0/torcx/docker:1.12.torcx.tgz",
			false,
		},
		{
			"invalid template",
			"https://example.com//{{.Variable}}",
			"stable",
			"1490.1.0",
			"amd64",
			"docker",
			"1.12",
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Logf("Testing %q", tt.desc)

		templ, err := template.New(tt.desc).Parse(tt.urlTemplate)
		if err != nil {
			t.Fatalf("failed to parse template: %s", err)
		}
		params := urlParams{
			OSChannel:      tt.OSChannel,
			OSVersion:      tt.OSVersion,
			OSArch:         tt.OSArch,
			AddonName:      tt.AddonName,
			AddonReference: tt.AddonReference,
		}

		result, err := urlFor(templ, params)

		if !tt.haserr && err != nil {
			t.Fatalf("got unexpected error %q", err)
		}
		if tt.haserr && err == nil {
			t.Fatal("expected error, got nil")
		}
		if result != tt.out {
			t.Fatalf("expected %q, got %q", tt.out, result)
		}
	}
}
