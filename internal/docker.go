package internal

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/coreos/go-systemd/dbus"
	"github.com/pkg/errors"
)

const cleanupUnit = `
[Unit]
Description=Clean docker datadir for torcx changes
DefaultDependencies=no
Before=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/bin/rm -rf /var/lib/docker

[Install]
RequiredBy=umount.target
`

// EnableDockerCleanupUnit install a systemd service which
// purges docker datadir before reboot.
func (a *App) EnableDockerCleanupUnit(conn *dbus.Conn) error {
	if conn == nil {
		return fmt.Errorf("got nil connection")
	}

	unitName := "torcx-docker-cleanup.service"
	unitPath := filepath.Join("/run/systemd/system/", unitName)
	if err := ioutil.WriteFile(unitPath, []byte(cleanupUnit), 0755); err != nil {
		errors.Wrapf(err, "failed to write %s", unitPath)
	}

	installed, _, err := conn.EnableUnitFiles([]string{unitPath}, true, true)
	if err != nil {
		return errors.Wrapf(err, "failed to enable runtime unit %q", unitName)
	}
	if !installed {
		return errors.Errorf("failed to install runtime unit %q", unitName)
	}
	if err := conn.Reload(); err != nil {
		errors.Wrap(err, "failed to daemon-reload systemd")
	}

	return nil
}
