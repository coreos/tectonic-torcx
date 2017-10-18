# Torcx configuration managed by Tectonic

This document gives a brief overview on all torcx details managed by Tectonic.

# Torcx profile

All Tectonic nodes will boot into a torcx profile named `tectonic`. This can be verified via:

```
$ cat /etc/torcx/next-profile

tectonic
```

This profile will pin down the container runtime version to be used by the kubelet. It can be verified via:

```
$ jq . /etc/torcx/profiles/tectonic.json

{
  "kind": "profile-manifest-v0",
  "value": {
    "images": [
      {
        "name": "docker",
        "reference": "1.12"
      }
    ]
  }
}
```

The result of the example above is that the `docker:1.12` addon will be applied on the next machine boot.

To check torcx runtime details for a running Tectonic node, please refer to the [runtime metadata doc][torcx-metadata].

# Torcx addon images

Images are consumed by torcx systemd generator each time a node is booting up. For this reason, addon images _must_:

 * be available at boot time in one of the local torcx stores
 * match the OS version being booted

`tectonic-torcx` takes care of both requirements by checking OS versions on both primary and secondary USR partitions, and by downloading any addon images not shipped directly in the OS.

Vendored addons are stored in `/usr/share/torcx/store/`. For example, this may look like:

```
$ ls -la /usr/share/torcx/store/

-rw-r--r--. 1 root root 22731839 Sep 27 00:16 docker:1.12.torcx.tgz
-rw-r--r--. 1 root root 26522617 Sep 27 00:16 docker:17.06.torcx.tgz
lrwxrwxrwx. 1 root root       22 Sep 27 00:16 docker:com.coreos.cl.torcx.tgz -> docker:17.06.torcx.tgz
```

By design `tectonic-torcx` does not make any assumption regarding the content of this directory.
However, it is able to gain information for current and next OS version via a remote manifest, described below.

# Torcx remote addons for Tectonic

Remote addons are available in a bucket for Tectonic to consume. Each CL version has an associated addons manifest listing all available images.
Such manifest is downloaded by `tectonic-torcx` to discover additional remote images.

Manifest URL is a templated configuration option for `tectonic-torcx`. For a concrete example, here is the manifest for CL release `1520.5.0`:
<https://tectonic-torcx.release.core-os.net/manifests/amd64-usr/1520.5.0/torcx_manifest.json>

[torcx-metadata]: https://github.com/coreos/docs/blob/master/torcx/metadata-and-systemd-target.md
