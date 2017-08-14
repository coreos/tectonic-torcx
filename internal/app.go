package internal

import (
	"text/template"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

type App struct {
	Conf Config

	// The list of OS versions for which we'll install torcx addons
	OSVersions []string

	NeedReboot bool
}

type Config struct {
	// Path to the torcx binary
	TorcxBin string

	// Templated URL to torcx store
	TorcxStoreURL *template.Template

	// The torcx profile name to create (if no others exist)
	ProfileName string

	// Path to the kubeconfig file
	Kubeconfig string

	// Path to the kubelet.env file that configures the kubelet service
	KubeletEnvPath string

	// Don't use the apiserver to determine k8s version, just use this
	ForceKubeVersion string

	// Don't use node configuration to determine OS channel, just use this
	ForceOSChannel string

	// If true, do an OS upgrade before proceeding
	OSUpgrade bool

	// If false (default), gpg-verify all fetched images
	NoVerifySig bool

	// The path to the gpg keyring to validate
	GpgKeyringPath string

	// The node annotation to set to indicate completion
	// This also causes the process to never exit
	WriteNodeAnnotation string

	// Our kubernetes node name
	NodeName string

	// The torcx store path - this is only used for testing
	torcxStoreDir string
}

func NewApp(c Config) (*App, error) {
	if c.torcxStoreDir == "" {
		c.torcxStoreDir = TORCX_STORE
	}

	a := App{
		Conf: c,
	}

	// Test that torcx exists
	err := a.torcxCmd(nil, []string{"help"})
	if err != nil {
		return nil, errors.Wrap(err, "could not execute torcx")
	}

	return &a, nil
}

func (a *App) Run() error {
	if a.Conf.OSUpgrade {
		if err := a.OSUpdate(); err != nil {
			return err
		}
	} else {
		if err := a.NextOSVersion(); err != nil {
			return err
		}
	}
	if err := a.GetCurrentOSVersion(); err != nil {
		return err
	}

	osChannel, err := a.GetCurrentOSChannel(a.Conf.ForceOSChannel)
	if err != nil {
		return err
	}

	k8sVersion, err := a.GetKubeVersion()
	if err != nil {
		return err
	}
	logrus.Infof("running on Kubernetes version %q", k8sVersion)

	dockerVersion, err := DockerVersionFor(k8sVersion)
	if err != nil {
		return err
	}

	err = a.InstallAddon("docker", dockerVersion, osChannel, a.OSVersions)
	if err != nil {
		return err
	}

	if a.Conf.KubeletEnvPath != "" {
		err = a.WriteKubeletEnv(a.Conf.KubeletEnvPath, k8sVersion)
		if err != nil {
			return err
		}
	}

	if a.Conf.WriteNodeAnnotation != "" {
		err = a.WriteNodeAnnotation()
		if err != nil {
			return err
		}
	}
	return nil
}
