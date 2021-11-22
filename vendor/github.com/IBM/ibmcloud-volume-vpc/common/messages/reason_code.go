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

const (
	//Timeout indicates IAM_TOKEN exchange request failed due to timeout
	Timeout = "Timeout"
	//EndpointNotReachable indicates IAM_TOKEN exchange request failed due to incorrect endpoint
	EndpointNotReachable = "EndpointNotReachable"
	//AuthenticationFailed indicate authentication to IAM endpoint failed. e,g IAM_TOKEN refresh
	AuthenticationFailed = "AuthenticationFailed"
	//VolumeAttachFailed indicates if volume attach to instance is failed
	VolumeAttachFailed = "VolumeAttachFailed"
	//VolumeDetachFailed indicates if volume detach from instance is failed
	VolumeDetachFailed = "VolumeDetachFailed"
	//VolumeAttachFindFailed indicates if the volume attachment is not found with given request
	VolumeAttachFindFailed = "VolumeAttachFindFailed"
	//VolumeAttachTimedOut indicates the volume attach is not completed within the specified time out
	VolumeAttachTimedOut = "VolumeAttachTimedOut"
	//VolumeDetachTimedOut indicates the volume detach is not completed within the specified time out
	VolumeDetachTimedOut = "VolumeDetachTimedOut"
	//InvalidServiceSession indicates that there is some issue with IAM token exchange request for container service
	InvalidServiceSession = "InvalidServiceSession"
)
