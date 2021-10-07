/**
 * Copyright 2020 IBM Corp.
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

// Package provider ...
package provider

// Context represents the volume provider management API for individual account, user ID, etc.
//go:generate counterfeiter -o fakes/context.go --fake-name Context . Context
type Context interface {
	VolumeManager
	VolumeAttachManager
	SnapshotManager
	VolumeFileAccessPointManager
}

// Session is an Context that is notified when it is no longer required
//go:generate counterfeiter -o fake/fake_session.go --fake-name FakeSession . Session
type Session interface {
	Context

	// GetProviderDisplayName returns the name of the provider that is being used
	GetProviderDisplayName() VolumeProvider

	// Close is called when the Session is nolonger required
	Close()
}
