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

import (
	"errors"
	"reflect"
	"time"

	"github.com/IBM/ibmcloud-volume-interface/lib/provider"
	"github.com/IBM/ibmcloud-volume-interface/lib/utils/reasoncode"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewError returns an error that is implemented by provider.Error.
// If optional wrapped errors are a provider.Error, this preserves all child wrapped
// errors in depth-first order.
func NewError(code reasoncode.ReasonCode, msg string, wrapped ...error) error {
	return NewErrorWithProperties(code, msg, nil, wrapped...)
}

// NewErrorWithProperties returns an error that is implemented provider.Error and
// which is decorated with diagnostic properties.
// If optional wrapped errors are a provider.Error, this preserves all child wrapped
// errors in depth-first order.
func NewErrorWithProperties(code reasoncode.ReasonCode, msg string, properties map[string]string, wrapped ...error) error {
	if code == "" {
		code = "" // TODO: ErrorUnclassified
	}
	var werrs []string
	for _, w := range wrapped {
		if w != nil {
			werrs = append(werrs, w.Error())
			if p, isPerr := w.(provider.Error); isPerr {
				werrs = append(werrs, p.Wrapped()...)
			}
		}
	}
	return provider.Error{
		Fault: provider.Fault{
			ReasonCode: code,
			Message:    msg,
			Properties: properties,
			Wrapped:    werrs,
		},
	}
}

// ErrorDeepUnwrapString returns the full list of unwrapped error strings
// Returns empty slice if not a provider.Error
func ErrorDeepUnwrapString(err error) []string {
	if perr, isPerr := err.(provider.Error); isPerr && perr.Wrapped() != nil {
		return perr.Wrapped()
	}
	return []string{}
}

// ErrorReasonCode returns reason code if a provider.Error, else ErrorUnclassified
func ErrorReasonCode(err error) reasoncode.ReasonCode {
	if pErr, isPerr := err.(provider.Error); isPerr {
		if code := pErr.Code(); code != "" {
			return code
		}
	}
	return reasoncode.ErrorUnclassified
}

// ErrorToFault returns or builds a Fault pointer for an error (e.g. for a response object)
// Returns nil if no error,
func ErrorToFault(err error) *provider.Fault {
	if err == nil {
		return nil
	}
	if pErr, isPerr := err.(provider.Error); isPerr {
		return &pErr.Fault
	}
	return &provider.Fault{
		ReasonCode: "", // TODO: ErrorUnclassified,
		Message:    err.Error(),
	}
}

// FaultToError builds a Error from a Fault pointer (e.g. from a response object)
// Returns nil error if no Fault.
func FaultToError(fault *provider.Fault) error {
	if fault == nil {
		return nil
	}
	return provider.Error{Fault: *fault}
}

// SetFaultResponse sets the Fault field of any response struct
func SetFaultResponse(fault error, response interface{}) error {
	value := reflect.ValueOf(response)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		return errors.New("value must be a pointer to a struct")
	}
	field := value.Elem().FieldByName("Fault")
	if field.Kind() != reflect.Ptr {
		return errors.New("value struct must have Fault provider.Fault field")
	}
	field.Set(reflect.ValueOf(ErrorToFault(fault)))
	return nil
}

// ZapError returns a zapcore.Field for an error that includes the metadata
// associated with a provider.Error. If the error is not a provider.Error then
// the standard zap.Error is used.
func ZapError(err error) zapcore.Field {
	if perr, isPerr := err.(provider.Error); isPerr {
		// Use zap.Relfect() to format all fields of struct
		// zap.Any() would select standard error formatting
		return zap.Reflect("error", perr)
	}

	return zap.Error(err)
}

//ErrorRetrier retry the function
type ErrorRetrier struct {
	MaxAttempts   int
	RetryInterval time.Duration
	Logger        *zap.Logger
}

//NewErrorRetrier return new ErrorRetrier
func NewErrorRetrier(maxAttempt int, retryInterval time.Duration, logger *zap.Logger) *ErrorRetrier {
	return &ErrorRetrier{
		MaxAttempts:   maxAttempt,
		RetryInterval: retryInterval,
		Logger:        logger,
	}
}

//ErrorRetry path for retry logic with logger passed in
func (er *ErrorRetrier) ErrorRetry(funcToRetry func() (error, bool)) error {
	var err error
	var shouldStop bool
	for i := 0; ; i++ {
		err, shouldStop = funcToRetry()
		er.Logger.Debug("Retry Function Result", zap.Error(err), zap.Bool("shouldStop", shouldStop))
		if shouldStop {
			break
		}
		if err == nil {
			return err
		}
		//Stop if out of retries
		if i >= (er.MaxAttempts - 1) {
			break
		}
		time.Sleep(er.RetryInterval)
		er.Logger.Warn("retrying after Error:", zap.Error(err))
	}
	//error set by name above so no need to explicitly return it
	return err
}
