// Copyright 2017 CoreOS Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"golang.org/x/crypto/openpgp"
	"text/template"
)

const (
	// StoreTemplate is the URL template for the default ContainerLinux torcx store
	StoreTemplate = "https://{{.OSChannel}}.release.core-os.net/{{.OSArch}}-usr/{{.OSVersion}}/torcx/{{.AddonName}}:{{.AddonReference}}.torcx.tgz"
)

// urlParams contains required parameters for store URL rendering
type urlParams struct {
	OSChannel      string
	OSArch         string
	OSVersion      string
	AddonName      string
	AddonReference string
}

// FetchAddon fetches and verifies a torcx addon. It returns
// the path to the downloaded file if successful, or error
func (a *App) FetchAddon(name, reference, osChannel, osVersion string) (string, error) {
	logrus.Infof("fetching addon %s:%s (%s)", name, reference, osVersion)

	tmpfile, err := ioutil.TempFile("", name+":"+reference)
	if err != nil {
		return "", errors.Wrapf(err, "could not create temporary addon")
	}
	defer tmpfile.Close()

	params := urlParams{
		OSChannel:      osChannel,
		OSVersion:      osVersion,
		OSArch:         runtime.GOARCH,
		AddonName:      name,
		AddonReference: reference,
	}
	url, err := urlFor(a.Conf.TorcxStoreURL, params)
	if err != nil {
		return "", errors.Wrap(err, "failed to get addon URL")
	}
	logrus.Debugf("GET %s > %s", url, tmpfile.Name())

	err = fetchURL(url, tmpfile)
	if err != nil {
		os.Remove(tmpfile.Name())
		return "", errors.Wrapf(err, "failed to fetch addon")
	}

	if !a.Conf.NoVerifySig {
		logrus.Debug("download complete, verifying...")
		if err := a.verify(url, tmpfile.Name()); err != nil {
			return "", errors.Wrapf(err, "gpg validation failed")
		}
	} else {
		logrus.Warn("Signature verification disabled! Skipping")
	}

	return tmpfile.Name(), nil
}

func urlFor(urlTemplate *template.Template, params urlParams) (string, error) {
	if urlTemplate == nil {
		return "", errors.New("missing URL template")
	}
	if params.OSChannel == "" || params.OSVersion == "" || params.OSArch == "" || params.AddonName == "" || params.AddonReference == "" {
		return "", errors.Errorf("missing URL parameter, got %#v", params)
	}

	var target bytes.Buffer
	if err := urlTemplate.Execute(&target, params); err != nil {
		return "", errors.Wrap(err, "failed to render URL template")
	}

	return target.String(), nil
}

// fetchURL fetches a URL to a given destination
func fetchURL(url string, dst io.WriteCloser) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.Errorf("failed to download %q: %s", url, resp.Status)
	}

	_, err = io.Copy(dst, resp.Body)
	if err != nil {
		return err
	}

	return dst.Close()
}

// verify will make sure a downloaded addon is signed by a key in the keyring.
// It assumes the signature is available at "$url.aci", and tries
// to fetch that
func (a *App) verify(url string, path string) error {
	if url == "" || path == "" {
		return fmt.Errorf("Invalid parameters")
	}

	// Retrieve the signature
	url = url + ".asc"
	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "failed to request signature")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to request signature: %s %s", url, resp.Status)
	}
	sig, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve signature")
	}
	sigb := bytes.NewBuffer(sig)

	// Get the keyring
	keyring, err := a.openKeyring()
	if err != nil {
		return errors.Wrap(err, "failed to open keyring")
	}
	logrus.Debugf("Opened keyring with %d keys", len(keyring))

	// Open the downloaded file
	target, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "failed to open addon")
	}
	defer target.Close()

	// Validate
	signer, err := openpgp.CheckArmoredDetachedSignature(keyring, target, sigb)
	if err != nil {
		return errors.Wrap(err, "failed to validate signature")
	}
	logrus.Debugf("good signature from %s", signer.PrimaryKey.KeyIdString())
	return nil
}

// openKeying returns the parsed keyring file.
func (a *App) openKeyring() (openpgp.EntityList, error) {
	if a.Conf.GpgKeyringPath == "" {
		return nil, fmt.Errorf("no gpg keyring specified")
	}
	fp, err := os.Open(a.Conf.GpgKeyringPath)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	return openpgp.ReadArmoredKeyRing(fp)
}
