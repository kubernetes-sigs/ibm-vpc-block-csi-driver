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

// Package util ...
package util

// SafeStringValue returns the referenced string value, treating nil as equivalent to "".
// It is intended as a type-safe and nil-safe test for empty values in data fields of
func SafeStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// StringHasValue returns true if the argument is neither nil nor a pointer to the
// zero/empty string.
func StringHasValue(s *string) bool {
	return s != nil && *s != ""
}
