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

	"github.com/stretchr/testify/assert"
)

// Test the standard case: both versions OK,
func TestStrategyNormal(t *testing.T) {
	m := makeManifest([]string{"1.13"})

	a := App{
		Conf: Config{},
		packageManifestCache: map[string]*PackageManifest{
			"9999.0.0": m,
			"9998.0.0": m,
		},
		NextOSVersion:    "9999.0.0",
		CurrentOSVersion: "9998.0.0",
	}

	pv, osv, err := a.PickVersion("docker", []string{"1.13"})
	assert.Nil(t, err)
	assert.Equal(t, pv, "1.13")
	assert.Equal(t, []string{"9999.0.0", "9998.0.0"}, osv, "os versions")
}

// Test that we do a reasonable thing when the box is running the latest version
func TestStrategyNoUpgrade(t *testing.T) {
	m := makeManifest([]string{"1.13"})
	a := App{
		Conf: Config{},
		packageManifestCache: map[string]*PackageManifest{
			"9998.0.0": m,
		},
		CurrentOSVersion: "9998.0.0",
	}
	pv, osv, err := a.PickVersion("docker", []string{"1.13"})
	assert.Nil(t, err)
	assert.Equal(t, pv, "1.13")
	assert.Equal(t, []string{"9998.0.0"}, osv, "os versions")
}

// If both OS versions are too old, we don't do anything
func TestStrategyTooOld(t *testing.T) {
	a := App{
		Conf:             Config{},
		CurrentOSVersion: "1500.0.0",
		NextOSVersion:    "1501.0.0",
	}
	pv, osv, err := a.PickVersion("docker", []string{"1.13"})
	assert.Nil(t, err)
	assert.Equal(t, pv, "")
	assert.Nil(t, osv)
}

// Test that we pick the preferred docker Version
func TestStrategyBestVersion(t *testing.T) {
	m := makeManifest([]string{"1.12", "1.13"})
	a := App{
		Conf: Config{},
		packageManifestCache: map[string]*PackageManifest{
			"9998.0.0": m,
		},
		CurrentOSVersion: "9998.0.0",
	}
	pv, osv, err := a.PickVersion("docker", []string{"17.03", "1.13", "1.12"})
	assert.Nil(t, err)
	assert.Equal(t, pv, "1.13")
	assert.Equal(t, []string{"9998.0.0"}, osv, "os versions")
}

// Test that we do whatever the NextOSVersion wants
func TestStrategyPrimary(t *testing.T) {
	m1 := makeManifest([]string{"1.13"})
	m2 := makeManifest([]string{"1.12"})

	a := App{
		Conf: Config{},
		packageManifestCache: map[string]*PackageManifest{
			"9999.0.0": m1,
			"9998.0.0": m2,
		},
		NextOSVersion:    "9999.0.0",
		CurrentOSVersion: "9998.0.0",
	}

	pv, osv, err := a.PickVersion("docker", []string{"1.13"})
	assert.Nil(t, err)
	assert.Equal(t, pv, "1.13")
	assert.Equal(t, []string{"9999.0.0"}, osv, "os versions")
}

func makeManifest(dockerVersions []string) *PackageManifest {
	m := PackageManifest{
		Packages: []Package{
			{
				Name:           "docker",
				DefaultVersion: dockerVersions[0],
				Versions:       []PackageVersion{},
			},
		},
	}

	for _, v := range dockerVersions {
		m.Packages[0].Versions = append(m.Packages[0].Versions,
			PackageVersion{
				Hash:    "sha512-000",
				Version: v,
				Locations: []Location{
					{
						Path: "/usr/share/torcx/store/docker.torcx.tgz",
					},
					{
						URL: "https://example.net/docker.torcx.tgz",
					},
				},
			})
	}

	fillManifestBackrefs(&m)
	return &m
}
