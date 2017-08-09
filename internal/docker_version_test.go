package internal

import (
	"testing"
)

func TestDockerVersionFor(t *testing.T) {
	for _, tc := range []struct {
		in     string
		out    string
		haserr bool
	}{
		{
			"v1.6.7+coreos.0",
			"1.12.6",
			false,
		},
		{
			"v1.5.6+coreos.0",
			"1.11.2",
			false,
		},
		{
			"v1.7.0+coreos.0",
			"1.12.6",
			false,
		},
		{
			"awefawef",
			"",
			true,
		},
		{
			"v2.0.0",
			"",
			true,
		},
	} {
		act, err := DockerVersionFor(tc.in)
		if err != nil {
			if !tc.haserr {
				t.Fatalf("DockerVersionFor(%q) unexpected err %v", tc.in, err)
			} else {
				continue
			}
		} else if tc.haserr {
			t.Fatalf("DockerVersionFor(%q) expected err, got none", tc.in)
		}

		if act != tc.out {
			t.Fatalf("DockerVersionFor(%q) expect %q act %q", tc.in, tc.out, act)
		}
	}
}
