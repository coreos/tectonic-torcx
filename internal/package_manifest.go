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
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/opencontainers/go-digest"

	"github.com/pkg/errors"
)

const (
	// StoreTemplate is the URL template for the default ContainerLinux torcx store
	ManifestURLTemplate = "https://tectonic-torcx.release.core-os.net/manifests/{{.Board}}/{{.OSVersion}}/torcx_manifest.json"

	KIND_PACKAGE_MANIFEST = "torcx-package-list-v0"
)

type packageManifestBox struct {
	Kind  string          `json:"kind"`
	Value PackageManifest `json:"value"`
}

type PackageManifest struct {
	Packages []Package `json:"packages"`
}

type Package struct {
	Name           string           `json:"name"`
	DefaultVersion string           `json:"DefaultVersion"`
	Versions       []PackageVersion `json:"versions"`
}

type PackageVersion struct {
	Package   *Package
	Version   string     `json:"version"`
	Hash      string     `json:"hash"`
	Locations []Location `json:"locations"`
}

type Location struct {
	Version *PackageVersion
	Path    string `json:"path"`
	URL     string `json:"url"`
}

// LocationFor determines the optimal location for a desired torcx package,
// given a specific docker version. It downloads and verifies the package manifest
// for a given OS version, caching the parsed manifest for reuse.
func (a *App) GetPackageManifest(osVersion string) (*PackageManifest, error) {
	manifest, ok := a.packageManifestCache[osVersion]
	if ok {
		return manifest, nil
	}

	if a.Conf.TorcxManifestURL == nil {
		return nil, errors.New("missing URL template")
	}

	type params struct {
		Board, OSVersion string
	}

	var manifestURLB, manifestBuff, manifestSigB bytes.Buffer
	if err := a.Conf.TorcxManifestURL.Execute(&manifestURLB, params{a.Board, osVersion}); err != nil {
		return nil, errors.Wrap(err, "failed to render URL template")
	}
	manifestURL := manifestURLB.String()
	logrus.Debugf("GET %s", manifestURL)

	// Fetch the manifest and signature
	if err := fetchURL(manifestURL, &manifestBuff); err != nil {
		return nil, errors.Wrapf(err, "could not fetch package manifest at %s", manifestURL)
	}

	if err := fetchURL(manifestURL+".asc", &manifestSigB); err != nil {
		return nil, errors.Wrapf(err, "could not fetch manifest signature at %s.aci", manifestURL)
	}

	mb := manifestBuff.Bytes()

	if err := a.gpgVerify(bytes.NewReader(mb), &manifestSigB); err != nil {
		return nil, errors.Wrap(err, "gpg validation failed")
	}

	manifest, err := parseTorcxManifest(mb)
	if err != nil {
		return nil, err
	}
	a.packageManifestCache[osVersion] = manifest

	return manifest, nil
}

// LocationFor picks the best location for a given package + version
// from a manifest. If the package or version doesn't exist, returns
// nil. When multiple locations are present, it prefers ones with a
// Path on disk, so fetching can be skipped.
func (m *PackageManifest) LocationFor(name, version string) (*Location, error) {
	var pkg *Package
	for _, p := range m.Packages {
		if p.Name == name {
			pkg = &p
			break
		}
	}
	if pkg == nil {
		return nil, fmt.Errorf("OS does not include package %s", name)
	}

	for _, v := range pkg.Versions {
		if v.Version == version {
			var loc *Location

			// loop through locations, prefering on-disk
			for _, l := range v.Locations {
				if l.isTorcxStore() {
					return &l, nil
				} else {
					loc = &l
				}
			}
			if loc != nil {
				return loc, nil
			}
		}
	}
	return nil, fmt.Errorf("Could not find version %s for package %s", version, name)
}

// IsTorcxStore determines if a given Location is already installed
// in a torcx store and we can skip installing it
func (l *Location) isTorcxStore() bool {
	// TODO(cdc): determine if this is actually something to worry about
	return strings.HasPrefix(l.Path, "/usr/share/torcx/store/")
}

// ValidateHash checks if the supplied reader matches the package's
// expected hash.
func (v *PackageVersion) ValidateHash(inp io.Reader) (bool, error) {
	d, err := digest.Parse(strings.Replace(v.Hash, "-", ":", 1))
	if err != nil {
		return false, errors.Wrap(err, "could not understand package hash")
	}

	verifier := d.Verifier()
	if _, err := io.Copy(verifier, inp); err != nil {
		return false, errors.Wrap(err, "could not read file for hash validation")
	}

	return verifier.Verified(), nil
}

// filename returns the expected filename on disk in the torcx store
func (v *PackageVersion) filename() string {
	return fmt.Sprintf("%s:%s.torcx.tgz", v.Package.Name, v.Version)
}

func parseTorcxManifest(data []byte) (*PackageManifest, error) {
	box := packageManifestBox{}

	err := json.Unmarshal(data, &box)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse manifest")
	}

	if box.Kind != KIND_PACKAGE_MANIFEST {
		return nil, errors.New("Unexpected manifest kind " + box.Kind)
	}
	fillManifestBackrefs(&box.Value)

	return &box.Value, nil
}

func fillManifestBackrefs(m *PackageManifest) {
	// Fill in the parent references
	for i := range m.Packages {
		for j := range m.Packages[i].Versions {
			m.Packages[i].Versions[j].Package = &m.Packages[i]

			for k := range m.Packages[i].Versions[j].Locations {
				m.Packages[i].Versions[j].Locations[k].Version = &m.Packages[i].Versions[j]
			}
		}
	}

}
