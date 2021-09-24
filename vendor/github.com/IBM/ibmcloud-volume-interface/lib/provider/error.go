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

import (
	"github.com/IBM/ibmcloud-volume-interface/lib/utils/reasoncode"
)

// Error implements the error interface for a Fault.
// Most easily constructed using util.NewError() or util.NewErrorWithProperties()
type Error struct {
	// Fault ...
	Fault Fault
}

// Fault encodes a fault condition.
// Does not implement the error interface so that cannot be accidentally
// misassigned to error variables when returned in a function response.
type Fault struct {
	// Message is the fault message (required)
	Message string `json:"msg"`

	// ReasonCode is fault reason code (required)  //TODO: will have better reasoncode mechanism
	ReasonCode reasoncode.ReasonCode `json:"code"`

	// WrappedErrors contains wrapped error messages (if applicable)
	Wrapped []string `json:"wrapped,omitempty"`

	// Properties contains diagnostic properties (if applicable)
	Properties map[string]string `json:"properties,omitempty"`
}

// FaultResponse is an optional Fault
type FaultResponse struct {
	Fault *Fault `json:"fault,omitempty"`
}

var _ error = Error{}

// Error satisfies the error contract
func (err Error) Error() string {
	return err.Fault.Message
}

// Code satisfies the legacy provider.Error interface
func (err Error) Code() reasoncode.ReasonCode {
	if err.Fault.ReasonCode == "" {
		return reasoncode.ErrorUnclassified
	}
	return err.Fault.ReasonCode
}

// Wrapped mirrors the legacy provider.Error interface
func (err Error) Wrapped() []string {
	return err.Fault.Wrapped
}

// Properties satisfies the legacy provider.Error interface
func (err Error) Properties() map[string]string {
	return err.Fault.Properties
}
