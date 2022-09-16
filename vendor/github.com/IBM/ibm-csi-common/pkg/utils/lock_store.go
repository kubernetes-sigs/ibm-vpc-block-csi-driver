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

// Package utils ...
package utils

import (
	"flag"
	"sync"
)

// LockEnabled ...
var LockEnabled = flag.Bool("lock_enabled", true, "Enable or disable lock")

// LockStore ...
type LockStore struct {
	store map[string]*sync.Mutex
}

// lockstore mutex
var lockstoremux = sync.Mutex{}

func (s *LockStore) checkAndInitLockStore() {
	if s.store == nil {
		s.store = make(map[string]*sync.Mutex)
	}
}

func (s *LockStore) getLock(name string) *sync.Mutex {
	lockstoremux.Lock()
	defer lockstoremux.Unlock()

	//check and init lock storage
	s.checkAndInitLockStore()

	//Get the lock Object
	if s.store[name] == nil {
		s.store[name] = &sync.Mutex{}
	}
	return s.store[name]
}

// Lock ...
func (s *LockStore) Lock(name string) {
	if *LockEnabled {
		s.getLock(name).Lock()
	}
}

// Unlock ...
func (s *LockStore) Unlock(name string) {
	if *LockEnabled {
		s.getLock(name).Unlock()
	}
}
