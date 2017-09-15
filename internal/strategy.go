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

	"github.com/Sirupsen/logrus"
	"github.com/coreos/go-semver/semver"
	"github.com/pkg/errors"
)

// MinimumRemoteDocker is the first CL bucket with published docker addons
const MinimumRemoteDocker = "1520.2.0"

// This error is returned when there is no suitable package available to install.
var NoVersionError = errors.New("No suitable version available")

// PickVersion implements our version selection & fallback logic
// Returns the desired package version and the OS versions for which to install
// it. Returns NoVersionError if no suitable versions are available.
// May also return a nil result if nothing should be installed (i.e. the OS version
// is too old for Torcx)
//
// Our update strategy is simple: we get a list of preferred packageVersions
// (in other words, the list of docker versions supported by Kubernetes). Then,
// pick the first one that is in the manifest for the "coming" OS version.
func (a *App) PickVersion(packageName string, packageVersions []string) (string, []string, error) {
	logrus.Infof("Determining correct %s version", packageName)
	if a.CurrentOSVersion == "" && a.NextOSVersion == "" {
		return "", nil, fmt.Errorf("Don't know OS versions") // should be unreachable
	}

	// First, we always prefer the "coming" OS version, at the potential risk
	// of installing a version not available on the current version
	primaryOSVersion := a.NextOSVersion
	secondaryOSVersion := a.CurrentOSVersion

	var packageVersion string

	// If there's no pending update, the "current" version is also the "next"
	if primaryOSVersion == "" {
		primaryOSVersion = secondaryOSVersion
		secondaryOSVersion = ""
	}

	// If the primary OS is before the Torcx epoch, then don't do anything
	if shouldSkip(MinimumRemoteDocker, primaryOSVersion) {
		logrus.Warnf("No OS versions are new enough! Nothing to do")
		return "", nil, nil
	}

	// Determine the first preferred version
	pm, err := a.GetPackageManifest(primaryOSVersion)
	if err != nil {
		return "", nil, errors.Wrapf(err, "Could not get package manifest for %s", primaryOSVersion)
	}
	for _, v := range packageVersions {
		loc, _ := pm.LocationFor(packageName, v)
		if loc != nil {
			packageVersion = v
			break
		}
	}
	if packageVersion == "" {
		return "", nil, NoVersionError
	}

	osVersions := []string{primaryOSVersion}

	// Now, check that the desired package version is available for the other OS version
	if secondaryOSVersion != "" && !shouldSkip(MinimumRemoteDocker, secondaryOSVersion) {
		pm, err := a.GetPackageManifest(secondaryOSVersion)
		if err != nil {
			return "", nil, errors.Wrapf(err, "Could not get package manifest for %s", secondaryOSVersion)
		}

		loc, _ := pm.LocationFor(packageName, packageVersion)
		if loc != nil {
			osVersions = append(osVersions, secondaryOSVersion)
		}
	}

	return packageVersion, osVersions, nil
}

// FilterOsVersions removes versions of Container Linux that don't use torcx.
func shouldSkip(minVersion string, version string) bool {
	minVer, _ := semver.NewVersion(minVersion)
	if minVer == nil { // Should not happen...
		logrus.Warnf("Could not parse minVersion %s", minVersion)
		return false
	}

	ver, err := semver.NewVersion(version)
	if err != nil {
		logrus.Warnf("Couldn't parse CL version %s!", version)
		return false
	}

	if ver.LessThan(*minVer) {
		logrus.Debugf("CL version %s too old; skipping", version)
		return true
	}
	return false
}
