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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTorcxGC(t *testing.T) {
	assert := assert.New(t)
	storeDir, err := ioutil.TempDir("", ".torcx-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(storeDir)

	dirs := []string{"101.0.0", "102.0.0", "102.0.1", "102.1.1", "xtra"}
	for _, d := range dirs {
		if err := os.Mkdir(filepath.Join(storeDir, d), 0755); err != nil {
			t.Fatal(err)
		}
		touch(t, filepath.Join(storeDir, d, "a"))
	}
	touch(t, filepath.Join(storeDir, "a"))

	a, err := NewApp(Config{
		torcxStoreDir: storeDir,
		TorcxBin:      "/bin/true",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = a.TorcxGC("102.0.1")
	assert.Nil(err)

	expected := []string{"102.0.1/", "102.1.1/", "a", "xtra/"}
	actual := listDir(t, storeDir)
	assert.Equal(expected, actual)
}

func touch(t *testing.T, path string) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func listDir(t *testing.T, path string) []string {
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() {
			n += "/"
		}
		out = append(out, n)
	}

	return out
}
