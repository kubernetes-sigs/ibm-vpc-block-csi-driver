/**
 * Copyright 2021 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package mountmanager ...
package mountmanager

import (
	mount "k8s.io/mount-utils"
	exec "k8s.io/utils/exec"
	testExec "k8s.io/utils/exec/testing"
)

// FakeNodeMounter ...
type FakeNodeMounter struct {
	*mount.SafeFormatAndMount
}

// NewFakeNodeMounter ...
func NewFakeNodeMounter() Mounter {
	//Have to make changes here to pass the Mock functions
	fakesafemounter := NewFakeSafeMounter()
	return &FakeNodeMounter{fakesafemounter}
}

// NewFakeSafeMounter ...
func NewFakeSafeMounter() *mount.SafeFormatAndMount {
	fakeMounter := &mount.FakeMounter{MountPoints: []mount.MountPoint{{
		Device: "valid-devicePath",
		Path:   "valid-vol-path",
		Type:   "ext4",
		Opts:   []string{"defaults"},
		Freq:   1,
		Pass:   2,
<<<<<<< HEAD
	}}, Log: []mount.FakeAction{}, Filesystem: map[string]mount.FileType{"fake": "Direectory"}}
	fakeExec := mount.NewFakeExec(execCallback)
=======
	}},
	}

	var fakeExec exec.Interface = &testExec.FakeExec{
		DisableScripts: true,
	}

>>>>>>> 25ac11d (Upgrade k8s package)
	return &mount.SafeFormatAndMount{
		Interface: fakeMounter,
		Exec:      fakeExec,
	}
}

// MakeDir ...
func (f *FakeNodeMounter) MakeDir(pathname string) error {
	return nil
}

// MakeFile ...
func (f *FakeNodeMounter) MakeFile(pathname string) error {
	return nil
}

// PathExists ...
func (f *FakeNodeMounter) PathExists(pathname string) (bool, error) {
	if pathname == "fake" {
		return true, nil
	}
	return false, nil
}

// NewSafeFormatAndMount ...
func (f *FakeNodeMounter) NewSafeFormatAndMount() *mount.SafeFormatAndMount {
	return NewFakeSafeMounter()
}
