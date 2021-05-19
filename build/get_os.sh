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

get_os_name() {
  local os_name
  os_name="$(uname -o | tr '[:upper:]' '[:lower:]')"
  case "$os_name" in
  *linux*)
    os_name="linux"
    ;;
  *darwin*)
    os_name="osx"
    ;;
  *msys*)
    os_name="windows"
    ;;
  *)
    os_name="sot_supported_$os_name"
    ;;
  esac
  printf "%s" "$os_name"
}