#!/usr/bin/env bash

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

this_cmd="$0"
base_dir="$(dirname $this_cmd)"
source "$base_dir/get_os.sh"
os_name="$(get_os_name)"
echo "os_name=$os_name  --- $base_dir  --- $this_cmd"


# Check gofmt
echo "==> Checking that code complies with gofmt requirements..."
gofmt_files=$(find . -name '*.go' | xargs gofmt -l -s)
if [[ -n ${gofmt_files} ]]; then
    echo 'gofmt needs running on the following files:'
    echo "${gofmt_files}"
    if [[ "$os_name" == "linux" ]]; then
        exit 1
    else
        echo "Ignoring formating issue because of os ($os_name)"
    fi
fi

exit 0
