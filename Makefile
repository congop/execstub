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

.PHONY: all build-go vet-go test-go install-go

GO = GO111MODULE=on go
GO_FILES_ALL = ./...
GO_FILES ?= $(GO_FILES_ALL)
GOPATH ?= $(shell go env GOPATH)

all: go-generate go-build go-test go-fmtcheck go-vet golangci-lint-run check-license-header

all-and-cover: all go-cover-with-race-check
	#

go-generate: 
	$(GO) generate ./...

go-build:
	$(GO) build -v $(GO_FILES)

go-vet:
	$(GO) vet $(GO_FILES)

go-test:
	$(GO) clean -testcache
	$(GO) test $(GO_FILES)

go-install:
	$(GO) install $(GO_FILES)

go-fmtcheck:
	bash "$(CURDIR)/build/gofmtcheck.sh"

go-cover-with-race-check:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

go-cover-with-race-check-show-html: go-cover-with-race-check
	go tool cover -html=coverage.txt

godoc-at-port9999:
	$(GOPATH)/bin/godoc -http=localhost:9999 -links=true -notes=TODO

golangci-lint-run:
	bash "$(CURDIR)/build/install-golangci-lint.sh"
	$(GOPATH)/bin/golangci-lint -E misspell -E dupl \
		-E gosec -E stylecheck -E gocritic -E nakedret -E gocyclo \
		-E golint -E goconst -E gocognit -E prealloc -E unparam \
		--max-issues-per-linter 200 --max-same-issues 20 \
		-v run

check-license-header:
	@build/check-license.sh

tools:
	$(GO) install ./awsproviderlint
	$(GO) install github.com/client9/misspell/cmd/misspell
	$(GO) install golang.org/x/tools/cmd/godoc

all-clean-room:
	# docker run --rm -v "$(CURDIR)":/usr/src/myapp -w /usr/src/myapp golang:1.14 make
	docker build .
