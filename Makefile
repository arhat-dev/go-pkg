GOTEST := GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) CGO_ENABLED=1 go test -mod=readonly -v -race -failfast -covermode=atomic

test.unit:
	${GOTEST} -coverprofile=coverage.txt -coverpkg=./... -run '!(TestJQCompability)' ./...

view.coverage:
	go tool cover -html=coverage.txt

test.unit.backoff:
	${GOTEST} -coverprofile=coverage.backoff.txt -coverpkg=./backoff ./backoff

view.coverage.backoff:
	go tool cover -html=coverage.backoff.txt

test.unit.confhelper:
	go run ./test/kube_incluster_env.go \
		-ca-data-from ./test/testdata/ca.crt \
		-token-data-from ./test/testdata/token \
		-- ${GOTEST} -coverprofile=coverage.confhelper.txt -coverpkg=./confhelper ./confhelper

view.coverage.confhelper:
	go tool cover -html=coverage.confhelper.txt

tidy:
	GOPROXY=direct GOSUMDB=off go mod tidy
