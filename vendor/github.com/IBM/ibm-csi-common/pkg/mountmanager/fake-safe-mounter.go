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
	"errors"
	"fmt"
	"strings"

	mount "k8s.io/mount-utils"
	exec "k8s.io/utils/exec"
	testExec "k8s.io/utils/exec/testing"
	testingexec "k8s.io/utils/exec/testing"
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
	}},
	}

	var fakeExec exec.Interface = &testExec.FakeExec{
		DisableScripts: true,
	}

	return &mount.SafeFormatAndMount{
		Interface: fakeMounter,
		Exec:      fakeExec,
	}
}

// MakeDir ...
func (f *FakeNodeMounter) MakeDir(pathname string) error {
	if pathname == "invalid-volPath-dir" {
		return errors.New("Path Creation failed")
	}
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

// GetSafeFormatAndMount returns the existing SafeFormatAndMount object of NodeMounter.
func (f *FakeNodeMounter) GetSafeFormatAndMount() *mount.SafeFormatAndMount {
	return f.SafeFormatAndMount
}

// FakeNodeMounterWithCustomActions ...
type FakeNodeMounterWithCustomActions struct {
	*mount.SafeFormatAndMount
	actionList []testingexec.FakeCommandAction
}

// NewFakeNodeMounterWithCustomActions ...
func NewFakeNodeMounterWithCustomActions(actionList []testingexec.FakeCommandAction) Mounter {
	fakeSafeMounter := NewFakeSafeMounterWithCustomActions(actionList)
	return &FakeNodeMounterWithCustomActions{fakeSafeMounter, actionList}
}

// MakeDir ...
func (f *FakeNodeMounterWithCustomActions) MakeDir(pathname string) error {
	return nil
}

// MakeFile ...
func (f *FakeNodeMounterWithCustomActions) MakeFile(pathname string) error {
	return nil
}

// Resize returns boolean and error if any
func (f *FakeNodeMounterWithCustomActions) Resize(devicePath string, deviceMountPath string) (bool, error) {
	r := mount.NewResizeFs(f.GetSafeFormatAndMount().Exec)
	needResize, err := r.NeedResize(devicePath, deviceMountPath)
	if err != nil {
		return false, err
	}
	if needResize {
		if _, err := r.Resize(devicePath, deviceMountPath); err != nil {
			return false, err
		}
	}
	return true, nil
}

// PathExists ...
func (f *FakeNodeMounterWithCustomActions) PathExists(pathname string) (bool, error) {
	if pathname == "fake" {
		return true, nil
	}
	return false, nil
}

// NewSafeFormatAndMount ...
func (f *FakeNodeMounterWithCustomActions) GetSafeFormatAndMount() *mount.SafeFormatAndMount {
	return f.SafeFormatAndMount
}

func NewFakeSafeMounterWithCustomActions(actionList []testingexec.FakeCommandAction) *mount.SafeFormatAndMount {
	var fakeExec exec.Interface = &testingexec.FakeExec{
		//DisableScripts: false,
		//ExactOrder:    true,
		CommandScript: actionList,
	}

	fakeMounter := &mount.FakeMounter{MountPoints: []mount.MountPoint{{
		Device: "devicePath",
		Path:   "vol-path",
		Type:   "ext4",
		Opts:   []string{"defaults"},
		Freq:   1,
		Pass:   2,
	}},
	}

	return &mount.SafeFormatAndMount{
		Interface: fakeMounter,
		Exec:      fakeExec,
	}
}

// Resize returns boolean and error if any
func (f *FakeNodeMounter) Resize(devicePath string, deviceMountPath string) (bool, error) {
	if devicePath == "fake" {
		return true, nil
	}
	return false, nil
}

// Mount
func (f *FakeNodeMounter) Mount(source, target, _ string, _ []string) error {
	if strings.Contains(source, "error_mount") {
		return fmt.Errorf("fake Mount: source error")
	} else if strings.Contains(target, "error_mount") {
		return fmt.Errorf("fake Mount: target error")
	}

	return nil
}
