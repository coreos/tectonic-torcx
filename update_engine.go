package main

import (
	"fmt"

	"github.com/Sirupsen/logrus"
)

// GetCurrentVersion determines the CL version currently running
func (a *App) GetCurrentVersion() (string, error) {

	return "", nil
}

// GetOtherVersion determines the CL version on the other partition
func (a *App) GetOtherVersion() (string, error) {
	return "", nil
}

// WaitForUpdate waits until the update agent has successfully
// updated the node.
func (a *App) OSUpdate() error {
	logrus.Infof("Updating node OS")

	// Trigger check for update

	// Wait until status is Idle or Needs Reboot

	return fmt.Errorf("not implemented")
}
