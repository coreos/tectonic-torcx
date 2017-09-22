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
	"text/template"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/go-systemd/dbus"
	"github.com/pkg/errors"
)

// App contains all the runtime state in a single, mutable place.
type App struct {
	Conf Config

	// The CL "board"
	Board string

	CurrentOSVersion string
	NextOSVersion    string

	K8sVersion string

	// Preferred docker versions
	DockerVersions []string

	// Whether a node reboot is required to finalize a docker upgrade.
	DockerRequiresReboot bool
	// Whether a node reboot is required to finalize an OS upgrade.
	OSRequiresReboot bool

	packageManifestCache map[string]*PackageManifest
}

type Config struct {
	// Path to the torcx binary
	TorcxBin string

	// Templated URL to torcx package manifest
	TorcxManifestURL *template.Template

	// The torcx profile name to create (if no others exist)
	ProfileName string

	// Path to the kubeconfig file
	Kubeconfig string

	// Path to the kubelet.env file that configures the kubelet service
	KubeletEnvPath string

	// Don't use the apiserver to determine k8s version, just use this
	ForceKubeVersion string

	// If true, do an OS upgrade before proceeding
	OSUpgrade bool

	// If false (default), gpg-verify all fetched images
	NoVerifySig bool

	// The path to the gpg keyring to validate
	GpgKeyringPath string

	// The node annotation to set to indicate completion
	// This also causes the process to never exit
	WriteNodeAnnotation string

	// Our kubernetes node name
	NodeName string

	// The torcx store path - this is only used for testing
	torcxStoreDir string

	// The path to the version manifest
	VersionManifestPath string

	// Whether to skip torcx setup entirely
	SkipTorcxSetup bool
}

func NewApp(c Config) (*App, error) {
	if c.torcxStoreDir == "" {
		c.torcxStoreDir = TORCX_STORE
	}

	a := App{
		Conf:                 c,
		packageManifestCache: map[string]*PackageManifest{},
	}

	// Test that torcx exists
	if !a.Conf.SkipTorcxSetup {
		err := a.torcxCmd(nil, []string{"help"})
		if err != nil {
			return nil, errors.Wrap(err, "could not execute torcx")
		}
	}

	return &a, nil
}

// GatherState collects the common system state - this has no side effects
func (a *App) GatherState(localOnly bool, envPath string) error {
	var err error

	a.CurrentOSVersion, a.Board, err = GetCurrentOSInfo()
	if err != nil {
		return err
	}
	logrus.Infof("Current OS version is %s, board is %s", a.CurrentOSVersion, a.Board)

	a.K8sVersion, err = a.GetKubeVersion(localOnly, envPath)
	if err != nil {
		return err
	}
	logrus.Infof("Detected Kubernetes version %q", a.K8sVersion)

	a.DockerVersions, err = a.VersionFor("docker", a.K8sVersion)
	if err != nil {
		return err
	}
	logrus.Infof("Kubernetes needs Docker version(s) %v", a.DockerVersions)

	return nil
}

// Bootstrap runs the steps necessary for bootstrapping a new node:
// - do an OS upgrade
// - install torcx packages
// - write kubelet.env
// - (if required) reboot the system
func (a *App) Bootstrap() error {
	dbusConn, err := dbus.New()
	if err != nil {
		return errors.Wrap(err, "failed to connect to login1 dbus")
	}
	defer dbusConn.Close()

	if err := a.GatherState(false, installerEnvPath); err != nil {
		return err
	}

	if a.Conf.OSUpgrade {
		if err := a.OSUpdate(); err != nil {
			return err
		}
	} else {
		if err := a.GetNextOSVersion(); err != nil {
			return err
		}
	}

	if a.Conf.SkipTorcxSetup {
		logrus.Warnf("Skipping torcx setup!")
	} else {
		dockerVersion, osVersions, err := a.PickVersion("docker", a.DockerVersions)
		if err != nil {
			return err
		}
		if len(osVersions) > 0 {
			if err := a.InstallAddon("docker", dockerVersion, osVersions); err != nil {
				return err
			}
		}
	}

	if a.Conf.KubeletEnvPath != "" {
		err = a.WriteKubeletEnv(a.Conf.KubeletEnvPath, a.K8sVersion)
		if err != nil {
			return err
		}
	}

	if a.DockerRequiresReboot || a.OSRequiresReboot {
		// Docker does not support version downgrades, so we need to
		// clean its datadir before reboot.
		if a.DockerRequiresReboot {
			logrus.Debug("docker change detected, cleaning datadir before reboot")
			if err := a.EnableDockerCleanupUnit(dbusConn); err != nil {
				logrus.Infof("unable to install docker cleanup unit: %s", err)
			}
		}

		// We trigger a reboot and block here, waiting for init to kill us.
		c := make(chan string)
		logrus.Info("node updated, triggering reboot to apply changes")
		_, err := dbusConn.StartUnit("reboot.target", "isolate", c)
		if err != nil {
			return errors.Wrapf(err, "failed to reboot")
		}
		return errors.Errorf("reboot result: %q", <-c)
	}

	return nil
}

// UpdateHook runs the steps expected for a pre-reboot hook
// - Install torcx package
// - gc if possible
// - write "hook successful" annotation
func (a *App) UpdateHook() error {
	if err := a.GatherState(true, kubeletEnvPath); err != nil {
		return err
	}

	if err := a.GetNextOSVersion(); err != nil {
		return err
	}

	dockerVersion, osVersions, err := a.PickVersion("docker", a.DockerVersions)
	if err != nil {
		return err
	}
	if len(osVersions) > 0 {
		if err := a.InstallAddon("docker", dockerVersion, osVersions); err != nil {
			return err
		}
	}

	if err := a.TorcxGC(a.CurrentOSVersion); err != nil {
		logrus.Warn("Failed to GC old torcx stores: ", err)
	}

	if a.Conf.WriteNodeAnnotation != "" {
		err := a.WriteNodeAnnotation()
		if err != nil {
			return err
		}
	}
	return nil
}
