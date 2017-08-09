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
	"github.com/Sirupsen/logrus"
	"github.com/coreos-inc/torcx-tectonic-bootstrap/pkg/multicall"
)

// Init initializes the CLI environment for torcx-tectonic multicall
func Init() error {
	logrus.SetLevel(logrus.WarnLevel)

	multicall.AddCobra(BootstrapCmd.Use, BootstrapCmd)
	multicall.AddCobra(HookCmd.Use, HookCmd)

	return nil
}

// MultiExecute dispatches multicall execution
func MultiExecute() error {
	return multicall.MultiExecute(false)
}
