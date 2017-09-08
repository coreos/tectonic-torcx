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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"golang.org/x/crypto/openpgp"
)

// FetchAddon fetches and verifies a torcx addon. It returns
// the path to the downloaded file if successful, or error
func (a *App) FetchAddon(loc *Location) (string, error) {
	if existing := a.tryFindExisting(loc.Version); existing != "" {
		logrus.Infof("Found identical package at %s, skipping download", existing)
		return existing, nil
	}

	logrus.Infof("fetching addon at %s", loc.URL)
	tmpfile, err := ioutil.TempFile("", loc.Version.filename())
	if err != nil {
		return "", errors.Wrapf(err, "could not create temporary addon")
	}
	defer tmpfile.Close()

	logrus.Debugf("GET %s > %s", loc.URL, tmpfile.Name())

	err = fetchURL(loc.URL, tmpfile)
	if err != nil {
		os.Remove(tmpfile.Name())
		return "", errors.Wrapf(err, "failed to fetch addon")
	}

	if err := tmpfile.Sync(); err != nil {
		os.Remove(tmpfile.Name())
		return "", errors.Wrapf(err, "failed to write addon")
	}

	// Seek the fp back to 0 and validate the downloaded file
	if _, err := tmpfile.Seek(0, 0); err != nil {
		os.Remove(tmpfile.Name())
		return "", errors.Wrapf(err, "failed to seek tmpfile")
	}
	ok, err := loc.Version.ValidateHash(tmpfile)
	if err != nil {
		os.Remove(tmpfile.Name())
		return "", errors.Wrapf(err, "failure during download hash validation")
	}
	if !ok {
		os.Remove(tmpfile.Name())
		return "", errors.New("Signature validation failed")
	}

	return tmpfile.Name(), nil
}

// fetchURL fetches a URL to a given destination
func fetchURL(url string, dst io.Writer) error {
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
	return nil
}

// verify will make sure a downloaded addon is signed by a key in the keyring.
// It assumes the signature is available at "$url.asc".
func (a *App) gpgVerify(data, sig io.Reader) error {
	if a.Conf.NoVerifySig {
		logrus.Warn("signature verification disabled, skipping")
		return nil
	}

	// Get the keyring
	keyring, err := a.openKeyring()
	if err != nil {
		return errors.Wrap(err, "failed to open keyring")
	}
	logrus.Debugf("Opened keyring with %d keys", len(keyring))

	// Validate
	signer, err := openpgp.CheckArmoredDetachedSignature(keyring, data, sig)
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

// findOnDisk is a simple shortcut that can find packages already downloaded.
// Since we know the hash of the package we want, we can check to see if we
// already have it. If the correct file is found, copy it to a temporary file.
// Returns empty string if not found or an error occurred
func (a *App) tryFindExisting(v *PackageVersion) string {
	scan_dirs := []string{
		"/usr/share/torcx/store",
		a.Conf.torcxStoreDir,
	}

	wantName := v.filename()
	foundPath := ""

	for _, dir := range scan_dirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			if info.Name() != wantName {
				return nil
			}

			fp, err := os.Open(path)
			if err != nil {
				return err
			}

			ok, _ := v.ValidateHash(fp)
			if ok {
				foundPath = path
				return filepath.SkipDir
			}

			return nil
		})
	}
	return foundPath
}
