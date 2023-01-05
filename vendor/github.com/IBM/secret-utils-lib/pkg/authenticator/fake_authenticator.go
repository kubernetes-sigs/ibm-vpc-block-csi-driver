/**
 * Copyright 2022 IBM Corp.
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

package authenticator

import (
	"errors"

	"go.uber.org/zap"
)

type FakeAuthenticator struct {
	logger *zap.Logger
	secret string
	url    string
}

func NewFakeAuthenticator(logger *zap.Logger) *FakeAuthenticator {
	return &FakeAuthenticator{logger: logger}
}

// GetToken ...
func (fa *FakeAuthenticator) GetToken(freshTokenRequired bool) (string, uint64, error) {
	if !freshTokenRequired {
		return "token", 1000, nil
	}
	return "", 0, errors.New("Not nil")
}

// GetSecret ...
func (fa *FakeAuthenticator) GetSecret() string {
	return fa.secret
}

// SetSecret ...
func (fa *FakeAuthenticator) SetSecret(secret string) {
	fa.secret = secret
}

// SetURL ...
func (fa *FakeAuthenticator) SetURL(url string) {
	fa.url = url
}

// IsSecretEncrypted ...
func (fa *FakeAuthenticator) IsSecretEncrypted() bool {
	return true
}

// SetEncryption ...
func (fa *FakeAuthenticator) SetEncryption(encrypted bool) {
	fa.logger.Info("Unimplemented")
}

func (fa *FakeAuthenticator) getURL() string {
	return fa.url
}
