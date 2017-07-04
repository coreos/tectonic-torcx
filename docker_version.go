package main

import "fmt"

// DockerVersionFor determines the exact docker version for a given k8s version
func (a *App) DockerVersionFor(k8sVersion string) (string, error) {
	return "", fmt.Errorf("not implemented")
}
