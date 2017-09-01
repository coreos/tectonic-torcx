package internal

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

const KIND_VERSION_MANIFEST = "VersionManifestV1"

type VersionManifest struct {
	Kind     string         `yaml:"kind"`
	Versions map[string]Dep `yaml:"versions"`
}

type Dep map[string]map[string][]string

// VersionFor parses the version manifest file and returns the list of preferred
// package versions for a given k8s version. The returned value will never be
// empty if error is nil.
func (a *App) VersionFor(name, k8sVersion string) ([]string, error) {
	m, err := a.GetVersionManifest()
	if err != nil {
		return nil, err
	}
	// The k8s version is something like "v1.6.7+coreos.0"
	// Trim it to "1.6"
	k8sVersion = strings.TrimLeft(k8sVersion, "v")
	ver, err := semver.NewVersion(k8sVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse k8s version")
	}
	k8sVersion = fmt.Sprintf("%d.%d", ver.Major, ver.Minor)

	return m.VersionFor("k8s", k8sVersion, name)
}

// VersionFor is the actual version lookup logic.
func (m *VersionManifest) VersionFor(haveName, haveVersion, wantName string) ([]string, error) {
	// Try and find the package + version
	h, ok := m.Versions[haveName]
	if !ok {
		return nil, fmt.Errorf("Version manifest has no versions for %s", haveName)
	}

	hv, ok := h[haveVersion]
	if !ok {
		return nil, fmt.Errorf("Version manifest has no entries for %s version %s", haveName, haveVersion)
	}

	wv, ok := hv[wantName]
	if !ok || len(wv) == 0 {
		return nil, fmt.Errorf("Version manifest for %s version %s doesn't specify %s", haveName, haveVersion, wantName)
	}
	return wv, nil
}

// Parse and quickly validate the yaml version manifest
func parseManifest(data []byte) (*VersionManifest, error) {
	m := VersionManifest{}
	err := yaml.Unmarshal(data, &m)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse version manifest")
	}

	if m.Kind != KIND_VERSION_MANIFEST {
		return nil, fmt.Errorf("did not understand version kind %s", m.Kind)
	}

	return &m, nil
}

// GetVersionManifest parses the version manifest file supplied by the user.
func (a *App) GetVersionManifest() (*VersionManifest, error) {
	data, err := ioutil.ReadFile(a.Conf.VersionManifestPath)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read version manifest")
	}

	return parseManifest(data)
}
