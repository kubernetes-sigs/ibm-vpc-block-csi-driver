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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVGSFeatureFlagDefaultsDisabled(t *testing.T) {
	previousValue, existed := os.LookupEnv(vgsFeatureFlag)
	assert.NoError(t, os.Unsetenv(vgsFeatureFlag))
	t.Cleanup(func() {
		if existed {
			assert.NoError(t, os.Setenv(vgsFeatureFlag, previousValue))
			return
		}
		assert.NoError(t, os.Unsetenv(vgsFeatureFlag))
	})

	assert.False(t, isVGSEnabled())
}

func TestVGSFeatureFlagRequiresValidTrueValue(t *testing.T) {
	testCases := []struct {
		value   string
		enabled bool
	}{
		{value: "true", enabled: true},
		{value: "TRUE", enabled: true},
		{value: "false", enabled: false},
		{value: "invalid", enabled: false},
		{value: "", enabled: false},
	}

	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv(vgsFeatureFlag, tc.value)
			assert.Equal(t, tc.enabled, isVGSEnabled())
		})
	}
}
