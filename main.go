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

package main

import (
	"os"
	"os/exec"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type Config struct {
	// Path to the torcx binary
	TorcxBin string

	// The torcx profile name to create (if no others exist)
	ProfileName string

	// Path to the kubeconfig file
	Kubeconfig string

	// Path to the kube.version file
	KubeVersionPath string

	// Don't use the apiserver to determine k8s version, just use this
	ForceKubeVersion string

	// If true (by default), do an OS upgrade before proceeding
	OSUpgrade bool
}

type App struct {
	Conf Config

	// The list of OS versions for which we'll install torcx addons
	OSVersions []string

	NeedReboot bool
}

const (
	DEFAULT_TORCX_PROFILE = "kubernetes-docker"
)

func ParseFlags() Config {
	c := Config{}

	tb, _ := exec.LookPath("torcx")

	pflag.StringVar(&c.Kubeconfig, "kubeconfig", "/etc/kubernetes/kubeconfig", "path to kubeconfig")
	pflag.StringVar(&c.KubeVersionPath, "version-file", "/etc/kubernetes/kube.version", "path to kube.version file")
	pflag.StringVar(&c.TorcxBin, "torcx-bin", tb, "path to torcx")
	pflag.BoolVar(&c.OSUpgrade, "do-os-upgrade", true, "force an OS upgrade")
	pflag.StringVar(&c.ProfileName, "torcx-profile", DEFAULT_TORCX_PROFILE, "torcx profile to create, if needed")
	pflag.StringVar(&c.ForceKubeVersion, "force-kube-version", "", "force a kubernetes version, rather than determining from the apiserver")

	vb := pflag.String("verbose", "warn", "verbosity level")
	pflag.Lookup("verbose").NoOptDefVal = "info"

	pflag.Parse()

	lvl, err := logrus.ParseLevel(*vb)
	if err != nil {
		logrus.Fatal("invalid verbosity level", *vb)
		os.Exit(2)
	}

	logrus.SetLevel(lvl)

	if c.Kubeconfig == "" && c.ForceKubeVersion == "" {
		logrus.Fatal("kubeconfig required")
	}

	if c.ProfileName == "" {
		logrus.Fatal("profile name required")
	}

	return c
}

func Init(c Config) (*App, error) {
	a := App{
		Conf: c,
	}

	// Test that torcx exists
	err := a.torcxCmd(nil, []string{"help"})
	if err != nil {
		return nil, errors.Wrap(err, "could not execute torcx")
	}

	return &a, nil
}

func (a *App) Run() error {
	if a.Conf.OSUpgrade {
		if err := a.OSUpdate(); err != nil {
			return err
		}
	} else {
		if err := a.NextOSVersion(); err != nil {
			return err
		}
	}
	if err := a.GetCurrentOSVersion(); err != nil {
		return err
	}

	var k8sVersion string
	if a.Conf.ForceKubeVersion != "" {
		k8sVersion = a.Conf.ForceKubeVersion
	} else {
		var err error
		k8sVersion, err = a.GetKubeVersion()
		if err != nil {
			return err
		}
	}

	dockerVersion, err := DockerVersionFor(k8sVersion)
	if err != nil {
		return err
	}

	err = a.InstallAddon("docker", dockerVersion, a.OSVersions)
	if err != nil {
		return err
	}

	// Writing the kubeversion file will block our systemd unit from running
	// so it's how we mark completion
	err = a.WriteKubeVersion(k8sVersion)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	conf := ParseFlags()

	app, err := Init(conf)
	if err != nil {
		logrus.Errorln(err)
		os.Exit(2)
	}

	err = app.Run()
	if err != nil {
		logrus.Errorln(err)
		os.Exit(1)
	}
}
