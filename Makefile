# set env var
.EXPORT_ALL_VARIABLES:
SHELL:=/bin/bash
# Variables
########################################################
# Lint                          			 		  #
########################################################
.PHONY: go-lint
go-lint:
	golangci-lint run

########################################################
# Test                         			 	          #
########################################################
.PHONY: go-test
go-test:
	go test -v -cover -count=1 ./... 

########################################################
# Run                                                  #
########################################################
.PHONY: go-run-local
go-run-local:
	export HELM_API_CREATE_API_KEY=create-key-123 && \
	export HELM_API_DELETE_API_KEY=delete-key-123 && \
	export HELM_API_UPDATE_API_KEY=update-key-123 && \
	export HELM_DRIVER=memory && \
	export HELM_API_LOG_LEVEL=debug && \
	go run main.go

########################################################
# Test tools                      			 							 #
########################################################

# kubectl port-forward -n helm-api-pg service/helm-api 8080:8080

.PHONY: test-create-post
test-create-post:
	curl -X POST http://localhost:8080/create-env \
  -H "Content-Type: application/json" \
	-H "X-API-Key: $FILL_ME_OUT" \
  -d '{ \
    "chartMetadata": { \
      "name": "chart1", \
      "version": "0.1.0", \
      "description": "A custom Helm chart", \
			"apiversion": "v2", \
		  "type": "application" \
    } \
  }'


.PHONY: test-update-post
test-update-post:
	curl -X POST http://localhost:8080/update-env/chart1 -H "Content-Type: application/json" -H "X-API-Key:  $FILL_ME_OUT" -d '{"action": "up"}'

.PHONY: test-del
 test-del:
	rm -rf charts/test-chart1

.PHONY: test-hc
test-hc:
	curl -s "http://localhost:8080/health-check"

.PHONY: test-del-post
test-del-post:
		curl -X POST http://localhost:8080/delete-env/chart1 \
		-H "X-API-Key:  $FILL_ME_OUT" \

.PHONY: test-list
test-list:
			curl -s http://localhost:8080/list

# Set the name of your application
APP_NAME=helm-api

# Set the version of your application
APP_VERSION=0.1

# Set the Go version you want to use
GO_VERSION=1.23.3

# Set the operating system to build for
GOOS=linux

# Set the Go flags for building and testing
GO_BUILD_FLAGS=-v
GO_TEST_FLAGS=-v


# Build the application
# go env -w GOOS=linux
#	go env -w GOARCH=amd64
.PHONY: go-build
go-build:
	go build $(GO_BUILD_FLAGS) -o $(APP_NAME) ./main.go

# Delete the application binary
#delete:
#	rm -f $(APP_NAME)