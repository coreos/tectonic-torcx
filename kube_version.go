package main

import "fmt"

// KubeVersion connects to the APIServer and determines the kubernetes version
func (a *App) KubeVersion() (string, error) {
	// XXX implement

	return "", fmt.Errorf("not implemented")
}

// WriteKubeVersion writes the kube version file.
// This should be done when everything has been successfully installed,
// so that the kubelet will start on boot.
func (a *App) WriteKubeVersion() error {
	return fmt.Errorf("not implemented")
}
