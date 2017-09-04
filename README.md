# tectonic-torcx

A self-contained node-helper to automatically operate [torcx][torcx] on [Tectonic][tectonic] machines.

[torcx]: https://github.com/coreos/torcx 
[tectonic]: https://coreos.com/tectonic

## Background

Tectonic needs a specific version of Docker to be installed. Since Docker
on Container Linux is managed by torcx, this tool suite keeps the torcx
configuration in sync with the cluster environment.

The tool ensures that the correct verison of Docker is in the torcx store for 
any potential OS versions. In other words, it populates torcx stores for Current
and Next OS versions.

## Details

This software handles two main cases:

1. A new node is added to the cluster and needs to be configured (bootstrap)
1. An existing node is ready to reboot to a new OS version (pre-reboot hook)

### 1: Bootstrap

1. Trigger an OS update (optional, default true)
1. Determine the Kubelet version to install
1. Determine the correct Docker version
1. Fetch and configure Docker torcx addons and profile
1. Set the correct kubelet version
1. Trigger node reboot (if needed by updates)

### 2: OS upgrade on a node

1. Watch for pre-reboot annotation 
1. Determine new OS version
1. Determine docker version
1. Fetch correct docker torcx addon
1. GC unneeded images
1. Add success annotation

In both cases, it can also determine/update kubelet based on cluster status.

## Build

`make all` to build for all supported architectures.

## Execute

This helper is normally run within a container:

```
docker run \
    --tmpfs /tmp \
    -v /usr/share:/usr/share:ro \
    -v /usr/lib/os-release:/usr/lib/os-release:ro \
    -v /usr/share/ca-certificates/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt:ro \
    -v /var/lib/torcx:/var/lib/torcx \
    -v /run/torcx:/run/torcx:ro \
    -v /run/metadata:/run/metadata:ro \
    -v /var/run/dbus:/var/run/dbus \
    -v /etc/coreos:/etc/coreos:ro \
    -v /etc/torcx:/etc/torcx \
    -v /etc/kubernetes:/etc/kubernetes \
    quay.io/coreos/tectonic-torcx:latest-dev \
    --verbose=debug
```


## See also

 * [bootkube](https://github.com/kubernetes-incubator/bootkube)
 * [kube-version](https://github.com/coreos/kube-version)
 * [tectonic-installer](https://github.com/coreos/tectonic-installer)
