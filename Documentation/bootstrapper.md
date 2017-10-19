# tectonic-torcx-bootstrap

This document gives a brief overview on how the bootstrapper works. Logic described here is specific to bootstrap context, actual code is generic to also fit as a CLUO daemonset hook.

## Execution and context

The bootstrapper runs on every new Tectonic node, at most once, before the node can join the cluster.
It must write the `/etc/kubernetes/kubelet.env` in order for the kubelet to start.
It is deployed via Ignition as a systemd service called `k8s-node-bootstrap`.
This service is installed and owned by tectonic-installer via the relevant [service unit][bootstrap-service].

## Configuration knobs

A few configuration options are available on the command-line. These are the most interesting:
 * `--upgrade-os=<bool>`: whether to check for new OS updates. Defaults to `true`
 * `--torcx-skip-setup=<bool>`: whether to skip all torcx-related steps. Defaults to `false`
 * `--no-verify-signatures=<bool>`: skip GPG verification on addons manifest. Default to `false`
 * `--force-kube-version=<string>`: force a specific kubernetes version, skipping default autodetection logic
 * `--torcx-manifest-url=<string>`: URL template for torcx addons manifest. More details below

Currently, torcx addons manifests are available at the following URL template:
```
https://tectonic-torcx.release.core-os.net/manifests/{{.Board}}/{{.OSVersion}}/torcx_manifest.json
```
Template variables are replaced with node-specific values. A detached signature is provided at the same URL suffixed with a `.asc` extension.

## Sources of information

Bootstrapper tries to gather state from the cluster and from a [remote bucket][remote], in order to prepare an up-to-date Kubernetes node.

This is the logic flow and the information sources it queries:
 1. Get cluster version from api-server `/version` endpoint
 1. Iff previous step permanently failed, use `/etc/kubernetes/installer/kubelet.env` to determine original cluster version (from installation time)
 1. Get runtime mappings from `/versions.yaml` hardcoded inside tectonic-torcx container
    * FUTURE - after getting rid of in-container runtime mappings instead:
    * Get runtime mappings from `tectonic-torcx-runtime-mappings` ConfigMap
    * Iff previous step permanently failed, use `/etc/kubernetes/installer/runtime-mappings.yaml` to determine runtime mappings
 1. Retrieve current OS version from `/usr/lib/os-release`
 1. Gather next OS version (if any) from `update_engine` via DBus
 1. Retrieve torcx remote manifests for both OS versions from a remote URL (see configuration notes above)
 1. Retrieve torcx addon images from URLs referenced by manifests

Please note that gathering cluster version and runtime-mappings can fail in some scenarios, most notably when bootstrapping a cluster from scratch.
For this reason, all remote resources needed by the installer must have a node-local fallback for seeding. Those assets are stored under `/etc/kubernetes/installer/`.

## Network requirements

The bootstrapper performs two kind of network requests:
 * to the api-server - these requests can permanently fail and gracefully fallback to local data. All configuration options are sourced from the `kubeconfig` on the node
 * to the remote torcx bucket - a permanent failure here results in a failed node bootstrap (i.e. no kubelet starting). The main entrypoint for this is the manifest template URL, which is configurable

[bootstrap-service]: https://github.com/coreos/tectonic-installer/blob/1.7.5-tectonic.1-rc.5/modules/ignition/resources/services/k8s-node-bootstrap.service
[remote]: https://tectonic-torcx.release.core-os.net/index.html
