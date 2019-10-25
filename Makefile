
.PHONY: sample-files take-snapshot deploy
sample-files: clean build-main
	- $(CLEAR)
	# - .$(PSEP)bin$(PSEP)main$(PSEP)cmd sample demo
	- $(MKDIR) .$(PSEP)bin$(PSEP)main$(PSEP)tmp
	# - dd if=/dev/urandom of=.$(PSEP)bin$(PSEP)main$(PSEP)/tmp/file-1 bs=1048576 count=1024
	- dd if=/dev/urandom of=.$(PSEP)bin$(PSEP)main$(PSEP)/tmp/file-1 bs=1048576 count=15360
take-snapshot: build-main
	- $(CLEAR)
	- $(RMDIR) .$(PSEP)bin$(PSEP)main$(PSEP)tmp$(PSEP).chunks
	- $(RMDIR) .$(PSEP)bin$(PSEP)main$(PSEP)tmp$(PSEP).metadata
	- .$(PSEP)bin$(PSEP)main$(PSEP)cmd splitter snapshot --tag some-tag
restore-snapshot: build-main
	- $(CLEAR)
	- .$(PSEP)bin$(PSEP)main$(PSEP)cmd splitter restore --tag some-tag
build: clean
	make build-main
	make build-darwin-amd64
	make build-linux-amd64
	make build-windows-amd64

build-main:
	for target in $(TARGET); do \
		GO111MODULE=${GO_MODULE} go build -ldflags "-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.Version=${VERSION} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.Revision=${REVISION} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.Branch=${BRANCH} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.BuildUser=${BUILDUSER} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.BuildDate=${BUILDTIME}" \
			-o .$(PSEP)bin$(PSEP)main$(PSEP)$$target .$(PSEP)$$target; \
	done

build-darwin-amd64:
	for target in $(TARGET); do \
		GO111MODULE=${GO_MODULE} CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -a -installsuffix cgo -ldflags "-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.Version=${VERSION} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.Revision=${REVISION} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.Branch=${BRANCH} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.BuildUser=${BUILDUSER} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.BuildDate=${BUILDTIME}" \
			-o .$(PSEP)bin$(PSEP)darwin$(PSEP)${VERSION}$(PSEP)$$target .$(PSEP)$$target; \
	done

build-linux-amd64:
	for target in $(TARGET); do \
		GO111MODULE=${GO_MODULE} CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -a -installsuffix cgo -ldflags "-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.Version=${VERSION} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.Revision=${REVISION} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.Branch=${BRANCH} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.BuildUser=${BUILDUSER} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.BuildDate=${BUILDTIME}" \
			-o .$(PSEP)bin$(PSEP)linux$(PSEP)${VERSION}$(PSEP)$$target .$(PSEP)$$target; \
	done

build-windows-amd64:
	for target in $(TARGET); do \
		GO111MODULE=${GO_MODULE} CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -a -installsuffix cgo -ldflags "-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.Version=${VERSION} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.Revision=${REVISION} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.Branch=${BRANCH} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.BuildUser=${BUILDUSER} \
			-X github.com/damoonazarpazhooh/File-Ingestion/pkg/version.BuildDate=${BUILDTIME}" \
			-o .$(PSEP)bin$(PSEP)windows$(PSEP)${VERSION}$(PSEP)$$target.exe .$(PSEP)$$target; \
	done
clean:
	rm -rf ./bin;
dep:

print:
	- $(CLEAR)
	- @echo $(space) ${PWD}
