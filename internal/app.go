package internal

import (
	"text/template"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/go-systemd/dbus"
	"github.com/pkg/errors"
)

type App struct {
	Conf Config

	OSChannel string

	CurrentOSVersion string
	NextOSVersion    string

	K8sVersion string

	DockerVersion string

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

// GatherState collects the common system state - this has no side effects
func (a *App) GatherState() error {
	var err error

	a.CurrentOSVersion, err = GetCurrentOSVersion()
	if err != nil {
		return err
	}

	if a.Conf.ForceOSChannel != "" {
		a.OSChannel = a.Conf.ForceOSChannel
	} else {
		a.OSChannel, err = GetCurrentOSChannel()
		if err != nil {
			return err
		}
	}

	a.K8sVersion, err = a.GetKubeVersion()
	if err != nil {
		return err
	}
	logrus.Infof("running on Kubernetes version %q", a.K8sVersion)

	a.DockerVersion, err = DockerVersionFor(a.K8sVersion)
	if err != nil {
		return err
	}
	logrus.Infof("want docker version %s", a.DockerVersion)

	return nil
}

// Bootstrap runs the steps necessary for bootstrapping a new node:
// - do an OS upgrade
// - install torcx packages
// - write kubelet.env
// - (if required) reboot the system
func (a *App) Bootstrap() error {
	dbusConn, err := dbus.New()
	if err != nil {
		return errors.Wrap(err, "failed to connect to login1 dbus")
	}
	defer dbusConn.Close()

	if err := a.GatherState(); err != nil {
		return err
	}

	if err := a.OSUpdate(); err != nil {
		return err
	}

	osVersions := []string{}
	if a.CurrentOSVersion != "" {
		osVersions = append(osVersions, a.CurrentOSVersion)
	}
	if a.NextOSVersion != "" {
		osVersions = append(osVersions, a.NextOSVersion)
	}

	if err := a.InstallAddon("docker", a.DockerVersion, a.OSChannel, osVersions, MinimumRemoteDocker); err != nil {
		return err
	}

	if a.Conf.KubeletEnvPath != "" {
		err = a.WriteKubeletEnv(a.Conf.KubeletEnvPath, a.K8sVersion)
		if err != nil {
			return err
		}
	}

	if a.NeedReboot {
		// We trigger a reboot and block here, waiting for init to kill us.
		c := make(chan string)
		logrus.Info("node updated, triggering reboot to apply changes")
		_, err := dbusConn.StartUnit("reboot.target", "isolate", c)
		if err != nil {
			return errors.Wrapf(err, "failed to reboot")
		}
		return errors.Errorf("reboot result: %q", <-c)
	}

	return nil
}

// UpdateHook runs the steps expected for a pre-reboot hook
// - Install torcx package
// - gc if possible
// - write "hook successful" annotation
func (a *App) UpdateHook() error {
	if err := a.GatherState(); err != nil {
		return err
	}

	if err := a.GetNextOSVersion(); err != nil {
		return err
	}

	osVersions := []string{}
	if a.CurrentOSVersion != "" {
		osVersions = append(osVersions, a.CurrentOSVersion)
	}
	if a.NextOSVersion != "" {
		osVersions = append(osVersions, a.NextOSVersion)
	}

	err := a.InstallAddon("docker", a.DockerVersion, a.OSChannel, osVersions, MinimumRemoteDocker)
	if err != nil {
		return err
	}

	if a.Conf.WriteNodeAnnotation != "" {
		err = a.WriteNodeAnnotation()
		if err != nil {
			return err
		}
	}
	return nil
}
