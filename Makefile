PROJECT_DIR=$(shell pwd)

###################################################
### OpenAPI         
###################################################

OAPI_CODEGEN_VERSION=v2.1.0
OAPI_CODEGEN_BIN=$(PROJECT_DIR)/bin/oapi-codegen
OAPI_GEN_DIR=$(PROJECT_DIR)/pkg/apigen
WORKER_APIGEN_DIR=$(PROJECT_DIR)/pkg/apigen/worker
COMPONENTS_APIGEN_DIR=$(PROJECT_DIR)/pkg/apigen/components

install-oapi-codegen:
	DIR=$(PROJECT_DIR)/bin VERSION=${OAPI_CODEGEN_VERSION} ./scripts/install-oapi-codegen.sh

prune-spec:
	@rm -f $(OAPI_GEN_DIR)/spec_gen.go

OAPI_GENERATE_ARG=types,fiber,client

gen-spec: install-oapi-codegen prune-spec
	$(OAPI_CODEGEN_BIN) -generate $(OAPI_GENERATE_ARG) -o $(OAPI_GEN_DIR)/spec_gen.go -package apigen $(PROJECT_DIR)/api/v1.yaml

###################################################
### Wire
###################################################

WIRE_VERSION=v0.6.0

install-wire:
	DIR=$(PROJECT_DIR)/bin VERSION=${WIRE_VERSION} ./scripts/install-wire.sh

WIRE_GEN=$(PROJECT_DIR)/bin/wire
gen-wire: install-wire
	$(WIRE_GEN) ./wire

###################################################
### SQL  
###################################################

SQLC_VERSION=1.25.0
QUERIER_DIR=$(PROJECT_DIR)/pkg/model/querier
SQLC_BIN=$(PROJECT_DIR)/bin/sqlc

install-sqlc:
	DIR=$(PROJECT_DIR)/bin VERSION=${SQLC_VERSION} ./scripts/install-sqlc.sh

clean-querier:
	@rm -f $(QUERIER_DIR)/*sql.gen.go
	@rm -f $(QUERIER_DIR)/copyfrom_gen.go   
	@rm -f $(QUERIER_DIR)/db_gen.go
	@rm -f $(QUERIER_DIR)/models_gen.go
	@rm -f $(QUERIER_DIR)/querier_gen.go

gen-querier: install-sqlc clean-querier
	$(SQLC_BIN) generate

###################################################
### mock 
###################################################

MOCKGEN_VERSION=1.6.0
MOCKGEN_BIN=$(PROJECT_DIR)/bin/mockgen

install-mockgen: 
	DIR=$(PROJECT_DIR)/bin VERSION=${MOCKGEN_VERSION} ./scripts/install-mockgen.sh

gen-mock: install-mockgen
	$(MOCKGEN_BIN) -source=pkg/model/model.go -destination=pkg/model/mock_gen.go -package=model
	$(MOCKGEN_BIN) -source=pkg/cloud/sms/sms.go -destination=pkg/cloud/sms/mock_gen.go -package=sms

###################################################
### Common
###################################################

gen: gen-spec gen-querier gen-wire gen-mock
	@go mod tidy

###################################################
### Dev enviornment
###################################################

dev:
	docker-compose up

reload:
	docker-compose restart dev

db:
	psql "postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable"

test:
	TEST_DIR=$(PROJECT_DIR)/e2e HOLD="$(HOLD)" ./scripts/run-local-test.sh "$(K)" 

ut:
	COLOR=ALWAYS go test -race -covermode=atomic -coverprofile=coverage.out -tags ut ./... 
	@go tool cover -html coverage.out -o coverage.html
	@go tool cover -func coverage.out | fgrep total | awk '{print "Coverage:", $$3}'
