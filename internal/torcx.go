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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/go-semver/semver"
	"github.com/pkg/errors"
)

const TORCX_STORE = "/var/lib/torcx/store"

type profileList struct {
	LowerProfileNames  []string `json:"lower_profile_names"`
	UserProfileName    *string  `json:"user_profile_name"`
	CurrentProfilePath *string  `json:"current_profile_path"`
	NextProfileName    *string  `json:"next_profile_name"`
	Profiles           []string `json:"profiles"`
}
type profileListBox struct {
	Kind  string      `json:"kind"`
	Value profileList `json:"value"`
}

type imageEntry struct {
	Name      string `json:"name"`
	Reference string `json:"reference"`
	Filepath  string `json:"filepath"`
}
type imageListBox struct {
	Kind  string       `json:"kind"`
	Value []imageEntry `json:"value"`
}

// InstallAddon fetches, verify and store an addon image
func (a *App) InstallAddon(name string, reference string, osVersions []string) error {
	logrus.Infof("Installing %s:%s for os versions %v", name, reference, osVersions)
	for _, osVersion := range osVersions {
		if a.AddonInStore(name, reference, osVersion) {
			logrus.Debugf("Skipping osVersion %s, already installed", osVersion)
			continue
		}

		manif, err := a.GetPackageManifest(osVersion)
		if err != nil {
			return err // should not happen; it is already in cache
		}

		loc, err := manif.LocationFor(name, reference)
		if err != nil {
			return err // should not happen, strategy caught this case
		}
		if loc.Path != "" {
			logrus.Debugf("Skipping osVersion %s, already in store", osVersion)
			continue
		}

		path, err := a.FetchAddon(loc)
		if err != nil {
			return errors.Wrapf(err, "failed to fetch addon")
		}

		err = a.copyToStore(path, name, reference, osVersion)
		if err != nil {
			return errors.Wrapf(err, "copy to store failed")
		}
	}
	logrus.Debugf("fetch phase complete, adding to profile")

	err := a.UseAddon(name, reference)
	if err != nil {
		return errors.Wrapf(err, "failed to enable addon")
	}
	a.DockerRequiresReboot = true

	return nil
}

// AddonInStore returns true if the referenced addon is already
// in the store
func (a *App) AddonInStore(name, reference, osVersion string) bool {
	il := imageListBox{}

	a.torcxCmd(&il, []string{
		"image", "list",
		"-n", osVersion,
		name,
	})

	for _, entry := range il.Value {
		if entry.Name == name && entry.Reference == reference {
			return true
		}
	}

	return false
}

// copyToStore moves an already downloaded addon to the store
func (a *App) copyToStore(path, name, reference, osVersion string) error {
	srcfd, err := os.Open(path)
	if err != nil {
		return err
	}
	defer srcfd.Close()

	if strings.HasPrefix(path, "/tmp/") {
		defer os.Remove(path)
	}

	var destPath string
	if err := os.MkdirAll(filepath.Join(a.Conf.torcxStoreDir, osVersion), 0755); err != nil {
		return err
	}
	if osVersion != "" {
		destPath = fmt.Sprintf("%s/%s/%s:%s.torcx.tgz",
			a.Conf.torcxStoreDir, osVersion, name, reference)
	} else {
		destPath = fmt.Sprintf("%s/%s:%s.torcx.tgz", a.Conf.torcxStoreDir, name, reference)
	}
	destfd, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil
	}
	defer destfd.Close()
	logrus.Debugf("copying to store: src %s dst %s", path, destPath)

	if _, err := io.Copy(destfd, srcfd); err != nil {
		return err
	}

	return destfd.Sync()
}

// UseAddon selects the addon for installation on next boot.
// When run on a fresh machine, this will create a profile
// of our choosing, otherwise will use the already-enabled version.
func (a *App) UseAddon(name string, reference string) error {
	profileName, err := a.profileName()
	if err != nil {
		return errors.Wrap(err, "could not determine / create torcx profile")
	}

	// Add this addon to the profile
	err = a.torcxCmd(nil, []string{
		"profile", "use-image",
		"--allow=missing",
		"--name", profileName,
		name + ":" + reference,
	})
	if err != nil {
		return errors.Wrap(err, "could not add image to profile")
	}

	err = a.torcxCmd(nil, []string{
		"profile", "set-next", profileName})
	if err != nil {
		return errors.Wrap(err, "could not set-next profile")
	}
	return nil
}

// TorcxGC removes versioned stores that we know we won't need.
// All versioned stores older than a given version are removed.
func (a *App) TorcxGC(minOSVersion string) error {
	minOSVers, err := semver.NewVersion(minOSVersion)
	if err != nil {
		return errors.Wrapf(err, "didn't understand minOSVersion")
	}

	// Only consider directories that look like CL versions
	versionRE := regexp.MustCompile(`^\d+\.\d+\.\d+$`)

	// List the user store
	entries, err := ioutil.ReadDir(a.Conf.torcxStoreDir)
	if err != nil {
		return errors.Wrap(err, "failed to list torcx store")
	}

	// Each torcx store directory holds addons and os version sub-stores
	// Remove all unwanted stores
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !versionRE.MatchString(entry.Name()) {
			continue
		}

		vers := semver.New(entry.Name())
		if vers == nil {
			continue
		}
		if !vers.LessThan(*minOSVers) {
			continue
		}

		p := filepath.Join(a.Conf.torcxStoreDir, entry.Name())
		logrus.Debugf("Removing unneeded torcx store directory %s", p)
		if err := os.RemoveAll(p); err != nil {
			return errors.Wrap(err, "failed to remove old torcx addons")
		}
	}

	return nil
}

// profileName determines which profile name to use.
// If this is an untouched machine, we want to create
// a new profile. If there is alread an existing profile,
// we should use that instead
func (a *App) profileName() (string, error) {
	plb := profileListBox{}
	err := a.torcxCmd(&plb, []string{"profile", "list"})
	if err != nil {
		return "", err
	}

	// If the next-profile name isn't default, just use it
	if plb.Value.NextProfileName != nil && *plb.Value.NextProfileName != "vendor" {
		logrus.Debugf("non-default torcx profile %s already active, using", *plb.Value.NextProfileName)
		return *plb.Value.NextProfileName, nil
	}

	// Otherwise, create our profile if it doesn't exist
	exists := false
	for _, profileName := range plb.Value.Profiles {
		if profileName == a.Conf.ProfileName {
			exists = true
			break
		}
	}
	if !exists {
		logrus.Debugf("creating torcx profile %s", a.Conf.ProfileName)
		err = a.torcxCmd(nil, []string{
			"profile", "new",
			"--name", a.Conf.ProfileName})
		if err != nil {
			return "", err
		}
	}
	return a.Conf.ProfileName, nil
}

// torcxCmd executes a torcx command. If result is not nil, attempt to
// json-unmarshal stdout in to the result
func (a *App) torcxCmd(result interface{}, args []string) error {
	logrus.Debug("executing: ", a.Conf.TorcxBin, " ", args)
	cmd := exec.Command(a.Conf.TorcxBin, args...)

	out, err := cmd.Output()
	if err != nil {
		switch e := err.(type) {
		case *exec.ExitError:
			logrus.Debugf("torcx exited with non-zero status code, stderr: %s", string(e.Stderr))
		}
		return err
	}

	if result != nil {
		return json.Unmarshal(out, result)
	}
	return nil
}
