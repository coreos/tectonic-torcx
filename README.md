# torcx-tectonic-bootstrap
Small agent to bootstrap a new Tectonic node.

## Background

When a new tectonic node is started in the cluster, it needs to have the correct
version of both the kubelet and corresponding docker torcx addons selected.

This is the first-boot equivalent to what the CLUO / NodeAgent do on a running
node.

## Process

1. Force an OS update
2. Determine the Kubelet version to install. 
3. Compute the correct Docker version.
4. Fetch the correct docker images
5. Set the correct kubelet version
6. Enable the real kubelet service

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
    -v /usr/share/torcx:/usr/share/torcx \
    -v /var/lib/torcx:/var/lib/torcx \
    -v /etc/torcx:/etc/torcx \
    -v /run/torcx:/run/torcx \
    -v /run/metadata:/run/metadata \
    -v /etc/kubernetes:/etc/kubernetes \
    -v /var/run/dbus:/var/run/dbus \
    -v /usr/share/coreos/os-release:/usr/share/coreos/os-release \
    quay.io/casey_callendrello/torcx-tectonic-bootstrap-amd64 \
    --verbose=debug
```


## See also
[kube-version](https://github.com/coreos/kube-version)


# Future plans
This tool should support three methods of operation:

1: a new node is added to a kubernetes cluster
2: an existing node has been os-updated
3: an existing node should upgrade the k8s version

It currently only supports number 1
