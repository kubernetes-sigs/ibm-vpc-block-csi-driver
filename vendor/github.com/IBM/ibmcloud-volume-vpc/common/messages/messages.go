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

// Package messages ...
package messages

import (
	"fmt"

	util "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/IBM/ibmcloud-volume-interface/lib/utils/reasoncode"
)

// MessagesEn ...
var MessagesEn map[string]util.Message

// GetUserErr ...
func GetUserErr(code string, err error, args ...interface{}) error {
	//Incase of no error message, dont construct the Error Object
	if err == nil {
		return nil
	}
	userMsg := GetUserMsg(code, args...)
	userMsg.BackendError = err.Error()
	return userMsg
}

// GetUserMsg ...
func GetUserMsg(code string, args ...interface{}) util.Message {
	userMsg := MessagesEn[code]
	if len(args) > 0 {
		userMsg.Description = fmt.Sprintf(userMsg.Description, args...)
	}
	return userMsg
}

// GetUserError ...
func GetUserError(code string, err error, args ...interface{}) error {
	userMsg := GetUserMsg(code, args...)

	if err != nil {
		userMsg.BackendError = err.Error()
	}
	return userMsg
}

// GetUserErrorCode returns reason code string if a util.Message, else ErrorUnclassified string
func GetUserErrorCode(err error) string {
	if uErr, isPerr := err.(util.Message); isPerr {
		if code := uErr.Code; code != "" {
			return code
		}
	}
	return string(reasoncode.ErrorUnclassified)
}
