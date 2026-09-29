/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ibmcsidriver

import (
	"os"
	"strconv"
	"strings"
)

const vgsFeatureFlag = "IS_VGS_ENABLED"

// isVGSEnabled fails closed so an absent or malformed flag never exposes the
// GroupController service accidentally.
func isVGSEnabled() bool {
	value, exists := os.LookupEnv(vgsFeatureFlag)
	if !exists {
		return false
	}

	enabled, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && enabled
}
