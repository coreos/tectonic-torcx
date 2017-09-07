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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOSRelease(t *testing.T) {
	assert := assert.New(t)
	inp := `NAME="Container Linux by CoreOS"
ID=coreos
VERSION=1465.0.0
VERSION_ID=1465.0.0
BUILD_ID=2017-07-06-0206
PRETTY_NAME="Container Linux by CoreOS 1465.0.0 (Ladybug)"
ANSI_COLOR="38;5;75"
HOME_URL="https://coreos.com/"
BUG_REPORT_URL="https://issues.coreos.com"
COREOS_BOARD="amd64-usr"`

	assert.Equal("1465.0.0", parseOSRelease(inp, "VERSION"))
	assert.Equal("amd64-usr", parseOSRelease(inp, "COREOS_BOARD"))
}
