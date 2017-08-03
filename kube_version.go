package main

import (
	"io/ioutil"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeVersion connects to the APIServer and determines the kubernetes version
func (a *App) GetKubeVersion() (string, error) {
	logrus.Info("Determining kubernetes version")
	config, err := clientcmd.BuildConfigFromFlags("", a.Conf.Kubeconfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to build kubeconfig")
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", errors.Wrap(err, "failed to build kube client")
	}

	version, err := client.ServerVersion()
	if err != nil {
		return "", errors.Wrap(err, "failed to get server version")
	}
	logrus.Debug("Got kubernetes version ", version.GitVersion)

	return version.GitVersion, nil

}

// WriteKubeVersion writes the kube version file.
// This should be done when everything has been successfully installed,
// so that the kubelet will start on boot.
func (a *App) WriteKubeVersion(version string) error {
	// XXX Do same formatting as the old unit file
	return ioutil.WriteFile(a.Conf.KubeVersionPath, []byte(version), 0644)
}
