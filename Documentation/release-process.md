# Release process

This document shows how to perform a `tectonic-torcx` release and which tools are required for that.

Requirements:
 * git
 * make
 * gpg
 * docker

Environment:
 * `${NEWVER}`: new release version (e.g. `v2.1.0`)

Steps:
 1. Ensure you have a local clean checkout of current master branch:
    * `git checkout -f master`
    * `git reset --hard`
    * `git pull`
 1. Ensure master can be properly built and tested:
    * `make clean && make`
    * `make test`
 1. Apply a signed tag to top commit and push it:
    * `git tag -s ${NEWVER} -m "tectonic-torcx ${NEWVER}"`
 1. Build container image:
    * `make clean && make container-amd64`
    * This will print the container image name. Double check it is in the form `quay.io/coreos/tectonic-torcx:vx.y.z` and does NOT contain any `-dirty` or commit suffix.
 1. Push all release artifacts:
    * `git push --tags`
    * make push
    * make push-production
 1. Perform a release on github:
    * Go to <https://github.com/coreos/tectonic-torcx/releases> and add a new release.
    * Write a short summary of PRs and notable changes.
    * Publish the release.

# Distribution updates

After each release, `tectonic-torcx` needs to be updated in downstream projects distributing it.

These are all known relevant places:
 * `tectonic-cluo-operator`: version need to be bumped in manifest and a new release performed
 * `tectonic-installer`: this is automatically updated via the mutable `installer-latest` tag
