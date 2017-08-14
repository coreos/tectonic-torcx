# torcx-tectonic
Configure torcx correctly for Tectonic machines.

## Background

Tectonic needs a specific version of Docker to be installed. Since Docker
on Container Linux is managed by Torcx, this tool suite keeps the Torcx
configuration in sync with the cluster environment.

The tool ensures that the correct verison of Docker is in the torcx store for 
any potential OS versions. In other words, the Current and Next OS versions
must be available in the store.

## Details

The tools handle 4 cases:

1. A new node is added to the cluster and needs to be configured
2. an existing node is ready to reboot to a new OS version
3. an existing node is ready to use a new kubelet version

### 1: Bootstrap

1. Force an OS update
2. Determine the Kubelet version to install. 
3. Compute the correct Docker version.
4. Fetch and configure the correct docker torcx addons
5. Set the correct kubelet version
6. Enable the real kubelet service

### 2: New OS

1. Determine new OS version
2. Determine docker version
3. Fetch correct docker torcx addon
4. GC unneeded images
5. Add success annotation

### 3: New kubelet version

1. Determine new kubelet version
2. Fetch correct docker torcx addon

### Open design questions:
- What happens when an OS update fails?
- What happens if we don't know which docker version to use?

## Build
`make all` to build for all supported architectures.

## Execute
It can be run as a container:

```
docker run \
    --tmpfs /tmp \
    -v /usr/share/ca-certificates/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt \
    -v /usr/share/torcx:/usr/share/torcx \
    -v /var/lib/torcx:/var/lib/torcx \
    -v /etc/torcx:/etc/torcx \
    -v /etc/coreos:/etc/coreos \
    -v /run/torcx:/run/torcx \
    -v /run/metadata:/run/metadata \
    -v /etc/kubernetes:/etc/kubernetes \
    -v /var/run/dbus:/var/run/dbus \
    -v /usr/share/coreos/os-release:/usr/share/coreos/os-release \
    -v /usr/lib/os-release:/usr/lib/os-release \
    quay.io/casey_callendrello/torcx-tectonic-bootstrap-amd64 \
    --verbose=debug
```


## See also
[kube-version](https://github.com/coreos/kube-version)
