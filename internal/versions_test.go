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
	"reflect"
	"testing"
)

func TestVersionFor(t *testing.T) {
	manifestData := `
kind: VersionManifestV1
versions:
  k8s:
    1.6:
        docker: [ "1.12"]
    1.7:
        docker: [ "1.12" ]
    1.8:
        docker: [ "1.13", "1.12"]
`

	m, err := parseVersionManifest([]byte(manifestData))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		in  string
		out []string
		err bool
	}{
		{in: "1.6", out: []string{"1.12"}},
		{in: "1.8", out: []string{"1.13", "1.12"}},
		{in: "1.5", err: true},
	} {
		actual, err := m.VersionFor("k8s", tc.in, "docker")
		if err != nil && !tc.err {
			t.Fatalf("VersionFor(%s): unexpected error %s", tc.in, err)
		}
		if err == nil && tc.err {
			t.Fatalf("VersionFor(%s): expected error, got none", tc.in)
		}

		if !reflect.DeepEqual(tc.out, actual) {
			t.Fatalf("VersionFor(%s), got %v expected %v", tc.in, actual, tc.out)
		}
	}

	_, err = m.VersionFor("fleet", "1.0", "docker")
	if err == nil {
		t.Fatal("expected err, got nil")
	}

	_, err = m.VersionFor("k8s", "1.8", "mysql")
	if err == nil {
		t.Fatal("expected err, got nil")
	}
}
