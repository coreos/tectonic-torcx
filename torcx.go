package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

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

func (a *App) InstallAddon(name string, reference string, osVersions []string) error {
	l := len(osVersions)
	logrus.Infof("Installing %s:%s for %d os versions", name, reference, l)

	for _, osVersion := range osVersions {
		if !a.AddonInStore(name, reference, osVersion) {
			path, err := a.FetchAddon(name, reference, osVersion)
			if err != nil {
				return errors.Wrapf(err, "failed to fetch addon")
			}

			err = a.moveToStore(path, name, reference, osVersion)
			if err != nil {
				return errors.Wrapf(err, "copy to store failed")
			}
		} else {
			logrus.Debugf("Skipping osVersion %s, already installed", osVersion)
		}
	}
	logrus.Debugf("fetch phase complete, adding to profile")

	err := a.UseAddon(name, reference)
	if err != nil {
		return errors.Wrapf(err, "failed to enable addon")
	}

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

// moveToStore moves an already downloaded addon to the store
func (a *App) moveToStore(path, name, reference, osVersion string) error {
	srcfd, err := os.Open(path)
	if err != nil {
		return err
	}
	defer srcfd.Close()

	var destPath string
	if err := os.MkdirAll(filepath.Join("/var/lib/torcx/store", osVersion), 0755); err != nil {
		return err
	}
	if osVersion != "" {
		destPath = fmt.Sprintf("/var/lib/torcx/store/%s/%s:%s.torcx.tgz",
			osVersion, name, reference)
	} else {
		destPath = fmt.Sprintf("/var/lib/torcx/store/%s:%s.torcx.tgz", name, reference)
	}
	destfd, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil
	}
	defer destfd.Close()
	logrus.Debugf("rename %s %s", path, destPath)

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
			logrus.Debugf("torcx edited with non-zero status code, stderr: %s", string(e.Stderr))
		}
		return err
	}

	if result != nil {
		return json.Unmarshal(out, result)
	}
	return nil
}
