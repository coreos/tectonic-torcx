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
	"github.com/spf13/pflag"
)

type Config struct {
	// Path to the torcx binary
	TorcxBin string

	// Path to the kubeconfig file
	Kubeconfig string

	// If true (by default), do an OS upgrade before proceeding
	OSUpgrade bool
}

type App struct {
	Conf Config
}

func ParseFlags() Config {
	c := Config{}

	tb, _ := exec.LookPath("torcx")

	pflag.StringVar(&c.Kubeconfig, "kubeconfig", "/etc/kubernetes/kubeconfig", "path to kubeconfig")
	pflag.StringVar(&c.TorcxBin, "torcx-bin", tb, "path to torcx")
	pflag.BoolVar(&c.OSUpgrade, "os-upgrade", true, "force an OS upgrade")

	pflag.Parse()

	return c
}

func Init(c Config) (*App, error) {
	a := App{
		Conf: c,
	}

	return &a, nil
}

func (a *App) Run() error {
	if a.Conf.OSUpgrade {
		if err := a.OSUpdate(); err != nil {
			return err
		}
	}

	k8sVersion, err := a.KubeVersion()
	if err != nil {
		return err
	}

	dockerVersion, err := a.DockerVersionFor(k8sVersion)
	if err != nil {
		return err
	}

	err = a.InstallDocker(dockerVersion)
	if err != nil {
		return err
	}

	err = a.WriteKubeVersion()
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
