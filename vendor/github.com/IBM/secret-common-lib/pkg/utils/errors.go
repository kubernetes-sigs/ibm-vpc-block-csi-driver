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

package utils

const (
	// ErrDecryptionNotSupported ...
	ErrDecryptionNotSupported = "API key is encrypted as per the configuration, decryption of the same is not supported."

	// ErrorFetchingEndpoint ...
	ErrorFetchingEndpoint = "Unable to fetch %s endpoint"

	// ErrInitSecretProvider ...
	ErrInitSecretProvider = "Error initializing secret provider"

	// ErrEmptyEndpoint ...
	ErrEmptyEndpoint = "%s endpoint not found"

	// ErrMultipleKeysUnsupported ...
	ErrMultipleKeysUnsupported = "Mutiple secret keys are unsupported"

	// ErrInvalidProviderType ...
	ErrInvalidProviderType = "Invalid provider type given, expected values are vpc, bluemix, softlayer"

	// ErrInvalidArgument ...
	ErrInvalidArgument = "Invalid arguments provided in the map, Only ProviderType or SecretKey expected"

	// ErrEmptySecretKeyProvided ...
	ErrEmptySecretKeyProvided = "Provided secret key is empty"
)
