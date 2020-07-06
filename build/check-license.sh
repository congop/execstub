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

#  head -n 20 ./stubbing.go | grep -E ".*Copyright.*2020.*The.*Execstub.*Authors.*Licensed.*under.*the.*Apache.*License.*Version.*2\.0.*limitations.*under.*the.*License.*" - ; echo $?
# head -n 20 ./stubbing.go | grep -z -E ".*Copyright\s+2020\s+The\s+Execstub\s+Authors.*Li.*" - ; echo $?

isWithoutLicenseHeader() {
  fileToCheck="$1"
  lpattern=".*Copyright\s+2020\s+The\s+Execstub\s+Authors.*"
  lpattern="${lpattern}Licensed\s+under\s+the\s+Apache\s+License\,\s+Version\s+2\.0.*"
  lpattern="${lpattern}limitations\s+under\s+the\s+License"
  head -n 20 "$fileToCheck" | grep -z -E "${lpattern}" -  &>/dev/null
  grepRet="$?"
  if [[ "0" != "$grepRet" ]]; then
    printf "%s" "$fileToCheck"
  fi
}

filesToCheck=$(find . -name '*.go' -o -name '*.sh' -o -name Makefile -o -name Dockerfile -o -name '*.yml')

filesWithoutLicense=""

for fToCheck in $filesToCheck
  do
    fwl=$(isWithoutLicenseHeader "$fToCheck")
    if [[ -n "$fwl" ]]; then
      fwl=$(printf "\n\t%s" "$fwl")
      filesWithoutLicense="${filesWithoutLicense}${fwl}"
    fi
  done

if [[ -n "$filesWithoutLicense" ]]; then 
  printf "\nThe following files do not have a license header:%s\n"  "$filesWithoutLicense" >&2
  exit 1
fi

exit 0