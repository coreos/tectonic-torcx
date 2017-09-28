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
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// CluoRuntimeMappings is the default path for hook runtime-mappings (tectonic-cluo ConfigMap)
	CluoRuntimeMappings = "/etc/runtime-mappings.yaml"
	// InstallerRuntimeMappings is the default path for bootstrapper runtime-mappings (installer file)
	InstallerRuntimeMappings = "/etc/kubernetes/installer/runtime-mappings.yaml"
	// versionManifestKind is the current type for the runtime-mappings YAML object
	versionManifestKind = "VersionManifestV1"
	// configMapNamespace is the default namespace for runtime-mappings ConfigMap
	configMapNamespace = "tectonic-system"
	// configMapName is the default name for the runtime-mappings ConfigMap
	configMapName = "tectonic-torcx-runtime-mappings"
	// configMapKey is the default object/key in the runtime-mappings ConfigMap
	configMapKey = "runtime-mappings.yaml"
)

type VersionManifest struct {
	Kind     string         `yaml:"kind"`
	Versions map[string]Dep `yaml:"versions"`
}

type Dep map[string]map[string][]string

// VersionFor parses the version manifest file and returns the list of preferred
// package versions for a given k8s version. The returned value will never be
// empty if error is nil.
func (a *App) VersionFor(localOnly bool, name, k8sVersion string) ([]string, error) {
	// TODO(lucab): consider caching this manifest if we grow to
	// more components other than docker.
	m, err := a.GetVersionManifest(localOnly)
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
func parseVersionManifest(data []byte) (*VersionManifest, error) {
	m := VersionManifest{}
	err := yaml.Unmarshal(data, &m)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse version manifest")
	}

	if m.Kind != versionManifestKind {
		return nil, fmt.Errorf("did not understand version kind %s", m.Kind)
	}

	return &m, nil
}

// GetVersionManifest parses the version manifest file supplied by the user.
func (a *App) GetVersionManifest(localOnly bool) (*VersionManifest, error) {
	path := a.Conf.VersionManifestPath
	if path == "" {
		return nil, errors.New("missing version manifest path")
	}

	// Conditionally try ConfigMap from api-server first (bootstrapper only)
	if !localOnly {
		logrus.Debug("Querying api-server for runtime mappings ConfigMap")
		manifest, err := a.versionManifestFromAPIServer()
		if err == nil {
			return parseVersionManifest([]byte(manifest))
		}
		logrus.Warnf("Failed to query api-server for ConfigMap: %s", err)
	}

	// Source mappings from local file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read runtime mappings from %q", path)
	}

	return parseVersionManifest(data)
}

// versionManifestFromAPIServer connects to the APIServer and determines
// runtime mappings from the relevant ConfigMap.
func (a *App) versionManifestFromAPIServer() (string, error) {
	config, err := clientcmd.BuildConfigFromFlags("", a.Conf.Kubeconfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to build kubeconfig")
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", errors.Wrap(err, "failed to build kube client")
	}

	var versionManifest string
	err = retry(3, 10, func() error {
		cmi := client.ConfigMaps(configMapNamespace)
		if cmi == nil {
			return errors.Errorf("nil ConfigMapInterface for namespace %s", configMapNamespace)
		}
		cm, e := cmi.Get(configMapName, meta_v1.GetOptions{})
		if e != nil {
			return e
		}
		if cm == nil || cm.Data[configMapKey] == "" {
			return errors.Errorf("missing entry %s/%s", configMapName, configMapKey)
		}
		versionManifest = cm.Data[configMapKey]
		return nil
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to get ConfigMap %s/%s", configMapNamespace, configMapName)
	}
	logrus.Debugf("Got %s from ConfigMap %s/%s", configMapKey, configMapNamespace, configMapName)

	return versionManifest, nil
}
