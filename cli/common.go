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
	_ "crypto/sha512" // for go-digest
	"os/exec"
	"text/template"

	"github.com/coreos/tectonic-torcx/internal"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/tectonic-torcx/pkg/multicall"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

var (
	cfg                  = internal.Config{}
	verbose              string
	flagTorcxManifestURL string
)

// Init initializes the CLI environment for tectonic-torcx multicall
func Init() error {
	logrus.SetLevel(logrus.WarnLevel)

	multicall.AddCobra(BootstrapCmd.Use, BootstrapCmd)
	multicall.AddCobra(HookPreCmd.Use, HookPreCmd)

	return nil
}

// MultiExecute dispatches multicall execution
func MultiExecute() error {
	return multicall.MultiExecute(false)
}

func init() {
	bootstrapInit()
	hookPreInit()
}

func commonFlags(f *pflag.FlagSet) {
	tb, _ := exec.LookPath("torcx")

	f.StringVar(&cfg.Kubeconfig, "kubeconfig", "/etc/kubernetes/kubeconfig", "path to kubeconfig")
	f.StringVar(&cfg.TorcxBin, "torcx-bin", tb, "path to torcx")
	f.StringVar(&flagTorcxManifestURL, "torcx-manifest-url", internal.ManifestURLTemplate, "URL (template) for torcx package manifest")
	f.StringVar(&cfg.ProfileName, "torcx-profile", TectonicTorcxProfile, "torcx profile to create, if needed")
	f.StringVar(&cfg.ForceKubeVersion, "force-kube-version", "", "force a kubernetes version, rather than determining from the apiserver")
	f.BoolVar(&cfg.NoVerifySig, "no-verify-signatures", false, "don't gpg-verify all downloaded addons")
	f.StringVar(&cfg.GpgKeyringPath, "keyring", "/pubring.gpg", "path to the gpg keyring")
	f.StringVar(&cfg.VersionManifestPath, "version-manifest", "/versions.yaml", "path to the version manifest file")
	f.StringVar(&verbose, "verbose", "info", "verbosity level")
}

// parseFlags parses CLI options, returning a populated configuration for the bootstrap agent
func parseFlags() (internal.Config, error) {
	zero := internal.Config{}

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

	if flagTorcxManifestURL == "" {
		flagTorcxManifestURL = internal.ManifestURLTemplate
	}

	tmpl, err := template.New("TorcxManifestURL").Parse(flagTorcxManifestURL)
	if err != nil {
		return zero, errors.Wrap(err, "error parsing URL template")
	}
	cfg.TorcxManifestURL = tmpl

	if cfg.VersionManifestPath == "" {
		return zero, errors.New("version-manifest required")
	}

	return cfg, nil
}
