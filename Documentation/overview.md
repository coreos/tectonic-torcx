# Project overview

This document gives a brief overview on this project, code organization and usages.

## Code navigation

This repository contains source for a multicall binary named `tectonic-torcx`.
It comprises two components which share most of their logic but are used in different places:
 * `tectonic-torcx-bootstrap`: this is invoked via docker as a plain systemd service by tectonic-installer.
 * `tectonic-torcx-hook-pre`: this is deployed as an inert daemonset by `tectonic-cluo-operator` and triggered by CLUO via a [pre-reboot hook][cluo-hook].
 
Project is structured as follow:
  * `main.go`: common main entrypoint, it dispatches the multicall logic
  * `cli/`: contains each multicall name as a separate file (i.e `/tectonic-torcx-bootstrap` runs `tectonic-torcx-bootstrap.go`)
  * `deploy/`: examples to manually deploy this container image on kubernetes
  * `internal/`: internal logic, further split in:
    * `torcx.go`: torcx store and profile manipulation
    * `update_engine.go`: trigger and watcher for `update_engine`
    * `package_manifest.go`: consumer of package manifests, as published in [ContainerLinux buckets][remote]

## Consumers

This repository is published as a container image on quay: <https://quay.io/repository/coreos/tectonic-torcx>

It has two main consumers:
 * tectonic-installer: image tag defined in [terraform config][config.tf] and used by `k8s-node-bootstrap` [service][bootstrap-service].
 * tectonic-cluo-operator: daemonset [manifest][tcluo-manifest] for the CLUO bundle.

Please note that at the moment the installer uses a dedicated mutable tag called `installer-latest` to bring up-to-date version mappings to old clusters.

## Version manifests

TODO(lucab): add here details about runtime mappings once sorted out

[cluo-hook]: https://github.com/coreos/container-linux-update-operator/blob/v0.3.1/doc/before-after-reboot-checks.md 
[remote]: https://tectonic-torcx.release.core-os.net/index.html
[config.tf]: https://github.com/coreos/tectonic-installer/blob/1.7.5-tectonic.1-rc.5/config.tf#L85
[bootstrap-service]: https://github.com/coreos/tectonic-installer/blob/1.7.5-tectonic.1-rc.5/modules/ignition/resources/services/k8s-node-bootstrap.service
[tcluo-manifest]: https://github.com/coreos-inc/tectonic-cluo-operator/blob/v0.2.1/manifests/0.2.1/torcx-pre-reboot-hook.yaml
