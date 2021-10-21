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

// Package reasoncode ...
package reasoncode

// ReasonCode ...
type ReasonCode string

const (

	// ErrorUnclassified indicates a generic unclassified error
	ErrorUnclassified = ReasonCode("ErrorUnclassified")

	// ErrorPanic indicates recovery from a panic
	ErrorPanic = ReasonCode("ErrorPanic")

	// ErrorTemporaryConnectionProblem indicates an *AMBIGUOUS RESPONSE* due to IaaS API timeout or reset
	// (Caller can continue to retry indefinitely)
	ErrorTemporaryConnectionProblem = ReasonCode("ErrorTemporaryConnectionProblem")

	// ErrorRateLimitExceeded indicates IaaS API rate limit has been exceeded
	// (Caller can continue to retry indefinitely)
	ErrorRateLimitExceeded = ReasonCode("ErrorRateLimitExceeded")
)

// -- General provider API (RPC) errors ---

const (

	// ErrorBadRequest indicates a generic bad request to the Provider API
	// (Caller can treat this as a fatal failure)
	ErrorBadRequest = ReasonCode("ErrorBadRequest")

	// ErrorRequiredFieldMissing indicates the required field is missing from the request
	// (Caller can treat this as a fatal failure)
	ErrorRequiredFieldMissing = ReasonCode("ErrorRequiredFieldMissing")

	// ErrorUnsupportedAuthType indicates the requested Auth-Type is not supported
	// (Caller can treat this as a fatal failure)
	ErrorUnsupportedAuthType = ReasonCode("ErrorUnsupportedAuthType")

	// ErrorUnsupportedMethod indicates the requested Provider API method is not supported
	// (Caller can treat this as a fatal failure)
	ErrorUnsupportedMethod = ReasonCode("ErrorUnsupportedMethod")
)

// -- Authentication and authorization problems --

const (

	//Timeout indicates that there was timeout reaching token exchange endpoint
	Timeout = ReasonCode("Timeout")

	//EndpointNotReachable indicates that token exchange endpoint is incorrect
	EndpointNotReachable = ReasonCode("EndpointNotReachable")

	// ErrorUnknownProvider indicates the named provider is not known
	ErrorUnknownProvider = ReasonCode("ErrorUnknownProvider")

	// ErrorUnauthorised indicates an IaaS authorisation error
	ErrorUnauthorised = ReasonCode("ErrorUnauthorised")

	// ErrorFailedTokenExchange indicates an IAM token exchange problem
	ErrorFailedTokenExchange = ReasonCode("ErrorFailedTokenExchange")

	// ErrorProviderAccountTemporarilyLocked indicates the IaaS account as it has been temporarily locked
	ErrorProviderAccountTemporarilyLocked = ReasonCode("ErrorProviderAccountTemporarilyLocked")

	// ErrorInsufficientPermissions indicates an operation failed due to a confirmed problem with IaaS user permissions
	// (Caller can retry later, but not indefinitely)
	ErrorInsufficientPermissions = ReasonCode("ErrorInsufficientPermissions")
)

// Attach / Detach problems
const (
	//ErrorVolumeAttachFailed indicates if volume attach to instance is failed
	ErrorVolumeAttachFailed = ReasonCode("ErrorVolumeAttachFailed")
	//ErrorVolumeDetachFailed indicates if volume detach from instance is failed
	ErrorVolumeDetachFailed = ReasonCode("ErrorVolumeDetachFailed")
)
