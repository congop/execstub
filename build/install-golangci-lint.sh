#!/bin/bash

# Copyright 2020 The Execstub Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

###############################################################################
# Does install golangci-lint
# Usage: 
#    install-golangci-lint.sh <version>
#  fb74c2e
###############################################################################  

version=${1}
#remove starting v
#Removing white, even those in beetween, version is anyway not suppose to have one
version_without_v_prefix="${version// /}"
version_without_v_prefix=${version#v}

GOLANGCI_LINT_PATH=$(go env GOPATH)/bin/golangci-lint

if [[ (-n "${version_without_v_prefix}") ]]; then
    echo "Usage: $0 <version e.g. 1.27.0 or v1.27.0>"
    exit 1
fi

if command ${GOLANGCI_LINT_PATH} 1>/dev/null 2>&1; then
    echo "Golangci-lint installed checking verion: required version:[${version}] trimmed-v-prefix: [${version_without_v_prefix}]"
    current_version_str=$(${GOLANGCI_LINT_PATH} --version)
    if [[ ${current_version_str} == *"${version_without_v_prefix}"* ]]; then
        echo "required version ( ${version} ) already installed, kipping installation: full version text: ${current_version_str}"
        exit 0
    fi
fi

echo "Installing golangci-lint ${version}"

curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin ${version}

${GOLANGCI_LINT_PATH} --version
