#/*
# Copyright 2021 The Kubernetes Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#    http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# */

#!/bin/bash

set -e
set +x
#git config --global url."https://$GHE_TOKEN@github.ibm.com/".insteadOf "https://github.ibm.com/"
set -x
cd /go/src/github.com/IBM/ibm-vpc-block-csi-driver
CGO_ENABLED=0 go build -a -ldflags '-X main.vendorVersion='"vpcBlockDriver-${TAG}"' -extldflags "-static"' -o /go/bin/ibm-vpc-block-csi-driver ./cmd/
