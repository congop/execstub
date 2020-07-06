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

isUint8() { 
    val=$1; val="${val// /}"; val=${val:-xxxnan---}; 
    if (( val >= 0 &&  val <= 255 )); then 
        true 
    else 
        false
    fi  
}

strToBase64() {
  if [[ -n "$1" ]]; then 
    printf "%s" "$1" | base64
  fi
  #printf ""
}

echoStubRequestAsBase64CSV() {
  req="$( strToBase64 ${__EXECSTUBBING_STUB_KEY} )"
  req="${req},$( strToBase64 ${__EXECSTUBBING_CMD_TO_STUB} )"
  
  dst=""
  for var in "$@"
  do
    ##varx="${var//[$'\t\r\n']/}"
    if [[ -n $dst ]]; then
      req="${req},$(strToBase64 ${var})"
    else 
      #first argument is the destination
      dst="${var}"
    fi
  done
  timeout 3s echo "$req" > ${dst}
  timeoutRet=$?
  #echo "StubbingReqp=$req ---> ${__EXECSTUBBING_PIPE_STUBBER}(timeoutRet=$timeoutRet)"
  if [[ "0" != "$timeoutRet" ]]; then
    echo 1>&2 "failed to send request: timeout_or_err:$timeoutRet dst=$dst req=$req params=$@"
    exit 255
  fi
}

nextStubRequestFilePath() {
  dataDir="$__EXECSTUBBING_DATA_DIR"
  printf "%s/ser_stubrequest_%s_%0.6d" "$dataDir" "$(date +'%Y%m%d-%H%M%S-%N')" $RANDOM
}

getThenDoDynamicExecOutcome() {
  #echo "catting ${__EXECSTUBBING_PIPE_TEST_HELPER_PROC}"
  oBase64CSV=$( timeout 2s cat ${__EXECSTUBBING_PIPE_TEST_HELPER_PROC} 2>&1 || echo timeout or err:${?} )
  OLD_IFS=${IFS}
  IFS=","
  #printf "${oBase64CSV},sss" | read -ra oData
  read -ra oData <<< "${oBase64CSV}"
  IFS="${OLD_IFS}"
  oLen="${#oData[@]}"
  # last item may be empty --> length 4
  if (( ${oLen}!=5 &&  ${oLen}!=4 )); then
    printf "%s" "expects 5 records in cvs but got ${oLen}, cvs=${oBase64CSV}" >&2
    exit 255
  fi
  decodeExits=""  
  exitCode=$( echo "${oData[0]}" | base64 -d)
  [[ "0" != "$?" ]] && decodeExits="${decodeExits} ExitCode NotBase64='${oData[0]}'"
  internalErr=$( echo "${oData[1]}" | base64 -d)
  [[ "0" != "$?" ]] && decodeExits="${decodeExits} InternalErrTxt NotBase64='${oData[1]}'"
  key=$( echo "${oData[2]}" | base64 -d)
  [[ "0" != "$?" ]] && decodeExits="${decodeExits} Key NotBase64='${oData[2]}'"
  stderr=$( echo "${oData[3]}" | base64 -d)
  [[ "0" != "$?" ]] && decodeExits="${decodeExits} Stderr NotBase64='${oData[3]}'"
  stdout=$( echo "${oData[4]}" | base64 -d)
  [[ "0" != "$?" ]] && decodeExits="${decodeExits} Stdout NotBase64='${oData[4]}'"

  if [[ -n "${decodeExits}" ]]; then
    printf "%s" "bad base64 encoding ${decodeExits} csv=${oBase64CSV}" >&2
    exit 255
  fi
  
  if [[ -n "${stderr}" ]]; then
    printf "%s" "${stderr}" 1>&2
  fi  

  if [[ -n "${stdout}" ]]; then
    printf "%s" "${stdout}"
  fi
 
  if [[ -n "${internalErr}" ]]; then 
    printf "%s" "${internalErr}" 1>&2
    exit 255
  fi
  
  if isUint8 "${exitCode}" &>/dev/null; then 
    exit ${exitCode}; 
  else 
    printf "%s" "Invalid base64 found: '${exitCode}' csv=${oBase64CSV}" 1>&2
    exit 255 
  fi;   
}

echoTimeoutAndExit() {
    echo "Timeout" 1>&2 
    kill -9 $1
}


__CMD_ABS_PATH=$(dirname "$0")
__CMD_ABS_PATH=$( cd "${__CMD_ABS_PATH}" && pwd )
__CMD_ABS_PATH="${__CMD_ABS_PATH}/$(basename $0)"

__CMD_CONFIG_PATH="${__CMD_ABS_PATH}.config"

source ${__CMD_CONFIG_PATH}

export __EXECSTUBBING_STUB_KEY
export __EXECSTUBBING_CMD_TO_STUB
export __EXECSTUBBING_DATA_DIR
# The following is more elaborate to ease debugging while developping
# - Glob and take the most recent even their should be exactly one named pipe
# - Keep errors in the variable (|&)
export __EXECSTUBBING_PIPE_STUBBER="$(ls  -t ${__CMD_ABS_PATH}_stubber_pipe_* |& head -n1)"
export __EXECSTUBBING_PIPE_TEST_HELPER_PROC="$(ls  -t ${__CMD_ABS_PATH}_testprocesshelper_pipe_* |& head -n1)"

testMethodName="${__EXECSTUBBING_TEST_HELPER_PROCESS_METHOD}"
testMethodName="${testMethodName// /}"



if [[ -n "${testMethodName}" ]]; then
    
    export __EXECSTUBBING_GO_WANT_HELPER_PROCESS=1
    export __EXECSTUBBING_STUB_CMD_CONFIG="${__CMD_CONFIG_PATH}"
    # e.g. /tmp/go-build720053430/b001/execstubbing.test -test.run=TestHelperProcess -- "$@"
    # -test.run takes a regex therefore matching the exact test helper process method
    ${__EXECSTUBBING_UNIT_TEST_EXEC} -test.run="^${__EXECSTUBBING_TEST_HELPER_PROCESS_METHOD}\$" -- "$@"

else 
    staticConfig="${__EXECSTUBBING_STD_OUT}${__EXECSTUBBING_STD_ERR}${__EXECSTUBBING_EXIT_CODE}"
    staticConfig="${staticConfig// /}"
    
    if [[ -n "${staticConfig}" ]]; then
        export nextStubRequestFilePath
        reqDstFilePath="$(nextStubRequestFilePath)"
        echoStubRequestAsBase64CSV "$reqDstFilePath" "$@"

        if [[ -n "${__EXECSTUBBING_STD_OUT}" ]]; then
          printf "%s" "${__EXECSTUBBING_STD_OUT}" | base64 -d
        fi
        if [[ -n "${__EXECSTUBBING_STD_ERR}" ]]; then
          printf "%s" "${__EXECSTUBBING_STD_ERR}" | base64 -d | cat  1>&2
        fi
        exitCode="${__EXECSTUBBING_EXIT_CODE}"
        exitCode="${exitCode// /}"

        if isUint8 "${exitCode}" &>/dev/null; then 
            exit ${exitCode}; 
        else 
            printf "%s" "Bad exit code ${__EXECSTUBBING_EXIT_CODE} (trimmed:${exitCode})" | cat 1>&2
            exit 255 
        fi;
    else 
        echoStubRequestAsBase64CSV "${__EXECSTUBBING_PIPE_STUBBER}" "$@"
        getThenDoDynamicExecOutcome
    fi

fi