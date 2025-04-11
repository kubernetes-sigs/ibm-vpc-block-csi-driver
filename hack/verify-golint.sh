#!/bin/bash

# Copyright 2021 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

LINT_CMD=$(go env GOPATH)/bin/golangci-lint
if [[ -z "$(command -v ${LINT_CMD})" ]]; then
  echo "Cannot find ${LINT_CMD}"
  exit 1
fi

echo "Verifying golint"
readonly PKG_ROOT="$(git rev-parse --show-toplevel)"

${LINT_CMD} version
${LINT_CMD} run --timeout=10m

echo "Congratulations! Lint check completed for all Go source files."
