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

// Package models ...
package models

import (
	"fmt"
)

// ErrorType ...
type ErrorType string

func (et ErrorType) String() string { return string(et) }

// Error types
const (
	ErrorTypeField     ErrorType = "field"
	ErrorTypeParameter ErrorType = "parameter"
	ErrorTypeHeader    ErrorType = "header"
)

// ErrorCode ...
type ErrorCode string

func (ec ErrorCode) String() string { return string(ec) }

// Error codes
const (
	ErrorCodeInvalidState ErrorCode = "invalid_state"
	ErrorCodeNotFound     ErrorCode = "not_found"
	ErrorCodeTokenInvalid ErrorCode = "token_invalid"
)

// Error ...
type Error struct {
	Errors []ErrorItem `json:"errors"`
	Trace  string      `json:"trace,omitempty"`
}

// ErrorItem ...
type ErrorItem struct {
	Code     ErrorCode    `json:"code,omitempty"`
	Message  string       `json:"message,omitempty"`
	MoreInfo string       `json:"more_info,omitempty"`
	Target   *ErrorTarget `json:"reqID,omitempty"`
}

// Error ...
func (ei ErrorItem) Error() string {
	return ei.Message + " Please check " + ei.MoreInfo
}

// Error ...
func (e Error) Error() string {
	if len(e.Errors) > 0 {
		return "Trace Code:" + e.Trace + ", " + e.Errors[0].Error()
	}

	return "Unknown error"
}

// ErrorTarget ...
type ErrorTarget struct {
	Name string    `json:"name,omitempty"`
	Type ErrorType `json:"type,omitempty"`
}

// IksError ...
type IksError struct {
	ReqID       string    `json:"incidentID" binding:"required"`
	Code        string    `json:"code" binding:"required"`
	Err         string    `json:"description" binding:"required"`
	Type        ErrorType `json:"type" binding:"required"`
	RecoveryCLI string    `json:"recoveryCLI,omitempty"`
	RecoveryUI  string    `json:"recoveryUI,omitempty"`
	RC          int       `json:"rc,omitempty"`
}

// Error ...
func (ikserr IksError) Error() string {
	return fmt.Sprintf("%s: %s", ikserr.Code, ikserr.Err)
}
