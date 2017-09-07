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
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocationFor(t *testing.T) {
	assert := assert.New(t)

	manif, err := ioutil.ReadFile("../testdata/package-manifest.json")
	if err != nil {
		t.Fatal(err)
	}

	m, err := parseTorcxManifest(manif)
	if err != nil {
		t.Fatal(err)
	}

	_, err = m.LocationFor("pants", "")
	assert.NotNil(err)

	_, err = m.LocationFor("docker", "0")
	assert.NotNil(err)

	l, err := m.LocationFor("docker", "17.06")

	// Test all the backrefs
	assert.Nil(err)
	assert.NotNil(l.Version)
	assert.Equal("17.06", l.Version.Version)
	assert.NotNil(l.Version.Package)
	assert.Equal("docker", l.Version.Package.Name)

	expected := Location{
		URL:     "https://tectonic-torcx.release.core-os.net/pkgs/amd64-usr/docker/fb608f.../docker:17.06.torcx.tgz",
		Version: l.Version, // kind of cheating
	}
	assert.Equal(expected, *l)

	l, err = m.LocationFor("docker", "1.12")
	assert.Nil(err)
	assert.NotNil(l.Version)
	expected = Location{
		Path:    "/usr/share/torcx/store/docker:1.12.torcx.tgz",
		Version: l.Version,
	}
	assert.Equal(expected, *l)
}

func TestValidateHash(t *testing.T) {
	assert := assert.New(t)

	manif, err := ioutil.ReadFile("../testdata/package-manifest.json")
	if err != nil {
		t.Fatal(err)
	}

	m, err := parseTorcxManifest(manif)
	if err != nil {
		t.Fatal(err)
	}

	// The manifest has a chosen hash here
	v := m.Packages[0].Versions[0]

	buff := bytes.NewReader([]byte("test\n"))
	ok, err := v.ValidateHash(buff)
	assert.Nil(err)
	assert.True(ok)

	buff = bytes.NewReader([]byte{})
	ok, err = v.ValidateHash(buff)
	assert.Nil(err)
	assert.False(ok)
}
