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
	"github.com/coreos-inc/torcx-tectonic-bootstrap/internal"
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
)

func bootstrapInit() {
	commonFlags(BootstrapCmd.Flags())

	// We configure the bootstrap systemd unit to only start if this file doesn't exist
	BootstrapCmd.Flags().StringVar(&cfg.KubeletEnvPath, "kubelet-env-path", "/etc/kubernetes/kubelet.env", "path to write kube.version file")
	BootstrapCmd.Flags().BoolVar(&cfg.OSUpgrade, "do-os-upgrade", true, "force an OS upgrade")
}

func runBootstrap(cmd *cobra.Command, args []string) error {
	conf, err := parseFlags()
	if err != nil {
		return err
	}

	app, err := internal.NewApp(conf)
	if err != nil {
		return err
	}

	return app.Bootstrap()
}
