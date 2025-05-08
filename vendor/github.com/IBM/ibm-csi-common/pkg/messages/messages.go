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

// Package messages ...
package messages

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Message Wrapper Message/Error Class
type Message struct {
	Code         string
	Type         codes.Code
	RequestID    string
	Description  string
	BackendError string
	CSIError     string
	Action       string
}

// Error Implement the Error() interface method
func (msg Message) Error() string {
	return msg.Info()
}

// Info ...
func (msg Message) Info() string {
	/*If the BackendError is from library e.g BackendError: {Trace Code:920df6e8-6be9-4b4a-89e4-837ecb3f513d,
	Code:InvalidArgument ,Description:Please check parameters,RC:400 Bad Request}
	*/
	if msg.BackendError != "" {
		return fmt.Sprintf("{RequestID: %s, BackendError: %s, Action: %s}", msg.RequestID, msg.BackendError, msg.Action)
	}
	/*If there is CSIError from Driver side e.g
	Error: XYZ is mandatory, Action: Please provide valid parameters
	*/
	if msg.CSIError != "" {
		return fmt.Sprintf("{RequestID: %s, Code: %s, Description: %s, Error: %s, Action: %s}", msg.RequestID, msg.Code, msg.Description, msg.CSIError, msg.Action)
	}
	/*If there is no error object then use the internal message e.g
	{RequestID: 9829616e-c58b-47ce-9b49-85c0060db753 , Code: NoCapabilities, Description: Capabilities must be provided,
	Action: Please provide capabilities}
	*/
	return fmt.Sprintf("{RequestID: %s, Code: %s, Description: %s, Action: %s}", msg.RequestID, msg.Code, msg.Description, msg.Action)
}

// MessagesEn ...
var MessagesEn map[string]Message

// GetCSIError ...
func GetCSIError(logger *zap.Logger, code string, requestID string, err error, args ...interface{}) error {
	userMsg := GetCSIMessage(code, args...)
	if err != nil {
		userMsg.CSIError = err.Error()
	}
	userMsg.RequestID = requestID

	logger.Error("FAILED CSI ERROR", zap.Error(userMsg))
	return status.Error(userMsg.Type, userMsg.Info())
}

// Populate backendError from library and based on RC:xxx code set the CSI return code.
// GetCSIBackendError ...
func GetCSIBackendError(logger *zap.Logger, requestID string, err error, args ...interface{}) error {
	var backendError string
	var userMsg Message

	if err != nil {
		backendError = err.Error()
	}
	// Based on RC:xxx code from library set the appropriate CSI return code.
	// We will consider two generic codes 5xx server side issue and 4xx client side issue.
	if strings.Contains(strings.Replace(backendError, " ", "", -1), RC5XX) {
		userMsg = GetCSIMessage(InternalError, args...)
	} else {
		userMsg = GetCSIMessage(InvalidParameters, args...)
	}

	userMsg.RequestID = requestID
	userMsg.BackendError = backendError

	logger.Error("FAILED BACKEND ERROR", zap.Error(userMsg))
	return status.Error(userMsg.Type, userMsg.Info())
}

// GetCSIMessage ...
func GetCSIMessage(code string, args ...interface{}) Message {
	userMsg := MessagesEn[code]
	if len(args) > 0 {
		userMsg.Description = fmt.Sprintf(userMsg.Description, args...)
	}
	return userMsg
}
