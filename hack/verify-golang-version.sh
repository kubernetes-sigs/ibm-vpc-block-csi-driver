#!/bin/bash

# Copyright 2026 The Kubernetes Authors.
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

# Locate the repository root directory regardless of current working directory.
PKG_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

# List of files in the repository that declare or pin a Go version.
files=(
  go.mod
  Dockerfile.builder
  Makefile
  .github/workflows/coverage.yml
)

# Extract the declared Go version from each respective file:
# 1. go.mod: reads the "go <version>" directive.
# 2. Dockerfile.builder: extracts the tag from the "FROM golang:<version>" base image.
# 3. Makefile: extracts the value assigned to "GOLANG_VERSION".
# 4. coverage.yml: extracts the "go-version: <version>" input for setup-go.
versions=(
  "$(awk '$1 == "go" {print $2}' "${PKG_ROOT}/go.mod")"
  "$(awk 'toupper($1) == "FROM" && $2 ~ /^golang:/ {
      sub(/^golang:/, "", $2); print $2
    }' "${PKG_ROOT}/Dockerfile.builder")"
  "$(awk '/^[[:space:]]*GOLANG_VERSION[[:space:]]*=/ {
      sub(/^[^=]*=[[:space:]]*/, ""); sub(/[[:space:]]*#.*/, "");
      gsub(/["\047[:space:]]/, ""); print
    }' "${PKG_ROOT}/Makefile")"
  "$(awk '$1 == "go-version:" {
      gsub(/["\047]/, "", $2); print $2
    }' "${PKG_ROOT}/.github/workflows/coverage.yml")"
)

# Validate that each file specifies exactly one valid semantic version (major.minor.patch).
# Rejects missing declarations, multiple declarations, ranges, and floating tags.
for i in "${!files[@]}"; do
  version=${versions[$i]}
  if [[ ! ${version} =~ ^[1-9][0-9]*\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
    echo "ERROR: ${files[$i]} must declare exactly one Go version in major.minor.patch format (found '${version}')." >&2
    exit 1
  fi
  echo "${files[$i]}: ${version}"
done

# go.mod is treated as the primary source of truth for the Go version.
# Verify that all other files in the repository match the version declared in go.mod.
go_version=${versions[0]}
for i in "${!files[@]}"; do
  if [[ ${versions[$i]} != "${go_version}" ]]; then
    echo "ERROR: ${files[$i]} uses Go ${versions[$i]}, but go.mod declares ${go_version}. Update it to match go.mod." >&2
    exit 1
  fi
done

echo "All repository Go versions match go.mod (${go_version})."
