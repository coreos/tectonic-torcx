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
	"io/ioutil"
	"runtime"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/coreos/container-linux-update-operator/pkg/updateengine"
)

// Avoid a race condition where this chan gets closed / gc'd while another
// G still has an outstanding message by declaring this global
var statusCh chan updateengine.Status

const (
	// OsReleaseFile contains the default path to the os-release file
	OsReleaseFile = "/usr/lib/os-release"
)

// GetCurrentOSInfo gets the current OS version and the board
func GetCurrentOSInfo() (string, string, error) {
	logrus.Debug("reading current OS version + board from " + OsReleaseFile)
	osr, err := ioutil.ReadFile(OsReleaseFile)
	if err != nil {
		return "", "", errors.Wrap(err, "could not read os-release file")
	}

	version := parseOSRelease(string(osr), "VERSION")
	if version == "" {
		return "", "", errors.New("invalid os-release file, unable to determine VERSION")
	}
	board := parseOSRelease(string(osr), "COREOS_BOARD")
	if board == "" {
		// Older releases did not expose `COREOS_BOARD` in `os-release`,
		logrus.Warn("missing COREOS_BOARD field, trying to fallback")
		board = coreosBoardFallback()
	}
	if board == "" {
		return "", "", errors.New("invalid os-release file, unable to determine COREOS_BOARD")
	}

	return version, board, nil
}

func parseOSRelease(releaseInfo, key string) string {
	lines := strings.Split(releaseInfo, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			val := strings.SplitN(line, "=", 2)[1]
			val = strings.Trim(val, "\"")
			return val
		}
	}
	return ""
}

// coreosBoardFallback provides a last chance fallback effort to determine
// board name, hardcoded on runtime arch label.
func coreosBoardFallback() string {
	board := ""
	switch arch := runtime.GOARCH; arch {
	case "amd64":
		board = "amd64-usr"
	case "arm64":
		board = "arm64-usr"
	}
	return board
}

// NextOSVersion gets the coming OS version from update_engine
// without changing anything.
func (a *App) GetNextOSVersion() error {
	logrus.Debug("Requesting next OS version")
	ue, err := updateengine.New()
	if err != nil {
		return errors.Wrapf(err, "failed to connect to update-engine")
	}
	defer ue.Close()

	status, err := ue.GetStatus()
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve update-engine status")
	}

	if status.CurrentOperation == updateengine.UpdateStatusUpdatedNeedReboot {
		logrus.Infof("Next OS version is %s", status.NewVersion)
		a.NextOSVersion = status.NewVersion
	} else {
		logrus.Debugf("update_engine status is %s, cannot determine next version", status.CurrentOperation)
	}
	return nil
}

// OSUpdate triggers the update engine to update and waits
// for it to finish
func (a *App) OSUpdate() error {
	logrus.Infof("Updating node OS")
	var err error

	// Connect to ue dbus api
	ue, err := updateengine.New()
	if err != nil {
		return errors.Wrap(err, "Failed to connect to update-engine")
	}
	defer ue.Close()

	logrus.Info("Triggering OS update")
	// Trigger check for update. This is non-blocking
	if err := ue.AttemptUpdate(); err != nil {
		return errors.Wrap(err, "failed to trigger update")
	}

	logrus.Debug("Waiting for update to finish")
	if err := a.waitForUpdate(ue); err != nil {
		return errors.Wrap(err, "failed to wait for update to complete")
	}

	return nil
}

// waitForUpdate watches the status channel and waits until
// it seems complete.
func (a *App) waitForUpdate(ue *updateengine.Client) error {
	statusCh = make(chan updateengine.Status, 10)
	stopCh := make(chan struct{})
	var wg sync.WaitGroup

	go func() {
		// use a waitgroup to fix a fun race condition where both
		// stopch and the client are closed, causing a panic
		wg.Add(1)
		ue.ReceiveStatuses(statusCh, stopCh)
		wg.Done()
	}()

	// Manually send the current status on the channel as a "start" marker
	firstStatus, err := ue.GetStatus()
	if err != nil {
		close(stopCh)
		return errors.Wrap(err, "failed to get status")
	}

	firstStatus.NewSize = -1 // hack
	statusCh <- firstStatus

	flushed := false
loop:
	for status := range statusCh {

		// The updateengine client starts queueing statuses as soon as
		// the connection is opened. Flush the channel until our manual
		// status is received. This is so we ignore any old errors.
		if !flushed && status.NewSize != -1 {
			flushed = true
			continue
		}

		logrus.Debug("current status: ", status.CurrentOperation, " ", status.NewVersion)

		switch status.CurrentOperation {
		case updateengine.UpdateStatusCheckingForUpdate, updateengine.UpdateStatusUpdateAvailable, updateengine.UpdateStatusDownloading, updateengine.UpdateStatusVerifying, updateengine.UpdateStatusFinalizing:
			// pass; update still in progress

		case updateengine.UpdateStatusUpdatedNeedReboot:
			// Update complete, reboot time
			logrus.Info("Update successful! Next version is ", status.NewVersion)
			a.NextOSVersion = status.NewVersion
			a.OSRequiresReboot = true
			break loop

		case updateengine.UpdateStatusIdle:
			// already up to date, no reboot needed
			logrus.Info("No update available")
			break loop

		case updateengine.UpdateStatusReportingErrorEvent:
			// TODO: determine if we care about errors
		}
	}

	close(stopCh)
	wg.Wait()
	return nil
}
