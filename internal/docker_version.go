package internal

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/go-semver/semver"
)

// DockerVersionFor determines the exact docker version for a given k8s version
func DockerVersionFor(k8sVersion string) (string, error) {
	// The k8s version is something like "v1.6.7+coreos.0"
	k8sVersion = strings.TrimLeft(k8sVersion, "v")

	ver, err := semver.NewVersion(k8sVersion)
	if err != nil {
		return "", err
	}

	dockerVersion := ""

	switch ver.Major {
	case 1:
		switch ver.Minor {
		case 5:
			dockerVersion = "1.11"
		case 6, 7:
			dockerVersion = "1.12"
		}
	}

	// TODO: determine correct fallback logic
	// we almost certainly just want to use the system docker
	if dockerVersion == "" {
		return "", fmt.Errorf("unsupported kubernetes version %v", k8sVersion)
	}

	logrus.Debug("using docker version: ", dockerVersion)

	return dockerVersion, nil
}
