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

package cli

import (
	"os/exec"

	"github.com/Sirupsen/logrus"
	bootstrap "github.com/coreos-inc/torcx-tectonic-bootstrap/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	// TectonicTorcxProfile is the default torcx profile used by the bootstrapper
	TectonicTorcxProfile = "tectonic"
)

var (
	// BootstrapCmd is the top-level cobra command for `torcx-tectonic-bootstrap`
	BootstrapCmd = &cobra.Command{
		Use:          "torcx-tectonic-bootstrap",
		RunE:         runBootstrap,
		SilenceUsage: true,
	}
	cfg     = bootstrap.Config{}
	verbose string
)

func init() {
	tb, _ := exec.LookPath("torcx")

	BootstrapCmd.Flags().StringVar(&cfg.Kubeconfig, "kubeconfig", "/etc/kubernetes/kubeconfig", "path to kubeconfig")
	BootstrapCmd.Flags().StringVar(&cfg.KubeVersionPath, "version-file", "/etc/kubernetes/kube.version", "path to kube.version file")
	BootstrapCmd.Flags().StringVar(&cfg.TorcxBin, "torcx-bin", tb, "path to torcx")
	BootstrapCmd.Flags().BoolVar(&cfg.OSUpgrade, "do-os-upgrade", true, "force an OS upgrade")
	BootstrapCmd.Flags().StringVar(&cfg.ProfileName, "torcx-profile", TectonicTorcxProfile, "torcx profile to create, if needed")
	BootstrapCmd.Flags().StringVar(&cfg.ForceKubeVersion, "force-kube-version", "", "force a kubernetes version, rather than determining from the apiserver")
	BootstrapCmd.Flags().BoolVar(&cfg.NoVerifySig, "no-verify-signatures", false, "gpg-verify all downloaded addons")
	BootstrapCmd.Flags().StringVar(&cfg.GpgKeyringPath, "keyring", "/pubring.gpg", "path to the gpg keyring")
	BootstrapCmd.Flags().StringVar(&verbose, "verbose", "info", "verbosity level")
}

func runBootstrap(cmd *cobra.Command, args []string) error {
	conf, err := parseFlags()
	if err != nil {
		return err
	}

	app, err := bootstrap.NewApp(conf)
	if err != nil {
		return err
	}

	return app.Run()
}

// parseFlags parses CLI options, returning a populated configuration for the bootstrap agent
func parseFlags() (bootstrap.Config, error) {
	zero := bootstrap.Config{}

	lvl, err := logrus.ParseLevel(verbose)
	if err != nil {
		return zero, errors.Wrap(err, "invalid verbosity level")
	}
	logrus.SetLevel(lvl)

	if cfg.Kubeconfig == "" && cfg.ForceKubeVersion == "" {
		return zero, errors.New("kubeconfig required")
	}

	if cfg.ProfileName == "" {
		return zero, errors.New("profile name required")
	}

	if !cfg.NoVerifySig && cfg.GpgKeyringPath == "" {
		return zero, errors.New("keyring path required")
	}

	return cfg, nil
}
