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

# This is strictly for build purpose only.
# Main objective is to provide a clean room build of execstubbing.
# We however try to avoid downloads by using docker layer caching
# and so provinding a sort of tooled environment

FROM golang:1.14-alpine3.12 as TooledBase

ARG version=v1.27.0

#apk add --no-cache --virtual .build-deps \
# bash --> for bash exec stub
# gcc + musl-dev(~glibc) --> cgo based binary builing(generate)
# coreutils --> date (alpine default is busybox date)
RUN set -eux; \
	apk add --no-cache --virtual .build-deps \
		curl \
		make \
        bash \
        gcc \
        musl-dev \
        coreutils \
        grep \
	; 

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin ${version}

WORKDIR /execstubbing
COPY go.* ./
RUN go mod download -x

FROM TooledBase

WORKDIR /execstubbing

COPY . ./

#RUN echo ###################################################################
#RUN echo $PATH
#RUN bash --version
#RUN base64 --help
#RUN date --version

RUN make