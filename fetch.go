package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

// FetchAddon fetches and verifies a torcx addon. It returns
// the path to the downloaded file if successful, or error
func (a *App) FetchAddon(name, reference, osVersion string) (string, error) {
	logrus.Infof("fetching addon %s:%s (%s)", name, reference, osVersion)

	tmpfile, err := ioutil.TempFile("", name+":"+reference)
	if err != nil {
		return "", errors.Wrapf(err, "could not create temporary addon")
	}
	defer tmpfile.Close()

	url := urlFor(name, reference, osVersion, runtime.GOARCH)
	logrus.Debugf("http get %s > %s", url, tmpfile.Name())

	err = fetchURL(url, tmpfile)
	if err != nil {
		os.Remove(tmpfile.Name())
		return "", errors.Wrapf(err, "failed to fetch addon")
	}

	// TODO: gpg verify file

	return tmpfile.Name(), nil
}

func urlFor(name, reference, osVersion, arch string) string {
	// XXX implement
	return fmt.Sprintf("http://192.168.121.1:8000/%s:%s.torcx.tgz",
		name, reference)
}

// fetchURL fetches a URL to a given destination
func fetchURL(url string, dst io.WriteCloser) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(dst, resp.Body)
	if err != nil {
		return err
	}

	return dst.Close()
}
