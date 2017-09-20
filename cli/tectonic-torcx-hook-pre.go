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
	"errors"
	"log/syslog"
	"os"
	"time"

	"github.com/coreos/tectonic-torcx/internal"
	"github.com/sirupsen/logrus"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
	"github.com/spf13/cobra"
)

var (
	// HookPreCmd is the top-level cobra command for `tectonic-torcx-hook-pre`
	HookPreCmd = &cobra.Command{
		Use:          "tectonic-torcx-hook-pre",
		RunE:         runHookPre,
		SilenceUsage: true,
	}
	sleep int
)

func hookPreInit() {
	commonFlags(HookPreCmd.Flags())

	HookPreCmd.Flags().StringVar(&cfg.WriteNodeAnnotation, "node-annotation", "", "Node annotation to write after successful operation")
	HookPreCmd.Flags().StringVar(&cfg.NodeName, "node-name", "", "Our node name")
	HookPreCmd.Flags().IntVar(&sleep, "sleep", 0, "sleep N seconds after success")
}

func runHookPre(cmd *cobra.Command, args []string) error {
	// Tee log output to syslog; docker logs are not persisted across reboots
	// by default, so this hook may be very difficult to debug
	hook, err := logrus_syslog.NewSyslogHook("", "", syslog.LOG_INFO, "")
	if err == nil {
		logrus.AddHook(hook)
	}
	conf, err := parseFlags()
	if err != nil {
		return err
	}

	// The kubernetes downward api passes values via environment vars
	if v := os.Getenv("NODE"); v != "" && conf.NodeName == "" {
		conf.NodeName = v
	}

	if conf.WriteNodeAnnotation != "" && conf.NodeName == "" {
		return errors.New("--node-annotation requires --node-name or env-var NODE")
	}

	app, err := internal.NewApp(conf)
	if err != nil {
		return err
	}

	err = app.UpdateHook()
	if err != nil {
		return err
	}

	if sleep > 0 {
		logrus.Info("Pre-reboot hook complete, sleeping forever")
		for {
			time.Sleep(time.Duration(sleep) * time.Second)
		}
	}
	return nil
}
