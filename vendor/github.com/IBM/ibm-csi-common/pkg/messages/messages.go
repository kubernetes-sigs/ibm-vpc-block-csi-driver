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
	Action       string
}

// Error Implement the Error() interface method
func (msg Message) Error() string {
	return msg.Info()
}

// Info ...
func (msg Message) Info() string {
	if msg.BackendError != "" {
		return fmt.Sprintf("{RequestID: %s, Code: %s, Description: %s, BackendError: %s, Action: %s}", msg.RequestID, msg.Code, msg.Description, msg.BackendError, msg.Action)
	}
	return fmt.Sprintf("{RequestID: %s, Code: %s, Description: %s, Action: %s}", msg.RequestID, msg.Code, msg.Description, msg.Action)
}

// MessagesEn ...
var MessagesEn map[string]Message

// GetCSIError ...
func GetCSIError(logger *zap.Logger, code string, requestID string, err error, args ...interface{}) error {
	userMsg := GetCSIMessage(code, args...)
	if err != nil {
		userMsg.BackendError = err.Error()
	}
	userMsg.RequestID = requestID

	logger.Error("FAILED CSI ERROR", zap.Error(userMsg))
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
