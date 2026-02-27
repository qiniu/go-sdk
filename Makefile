test:
	go test -tags='unit integration' -failfast -count=1 -v -timeout 350m -coverprofile=coverage.txt `go list ./... | egrep -v 'examples|sms'` | tee -a test.log

unittest:
	go test -tags=unit -failfast -count=1 -v -coverprofile=coverage.txt `go list ./... | egrep -v 'examples|sms'`

integrationtest:
	go test -tags=integration -failfast -count=1 -parallel 1 -v -coverprofile=coverage.txt `go list ./... | egrep -v 'examples|sms'`

staticcheck:
	staticcheck `go list ./... | egrep -v 'examples|sms'`

generate:
	go generate ./storagev2/
	go generate ./iam/
	go generate ./media/
	go generate ./audit/
	gofmt -w .
	gofumpt -w .

generate-sandbox:
	# 控制面 API
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 \
		--config sandbox/apis/oapi-codegen.yaml \
		../api-specs/sandbox/openapi.yml
	# envd HTTP API
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 \
		--config sandbox/envdapi/oapi-codegen.yaml \
		../api-specs/sandbox/envd/envd.yaml
	# envd ConnectRPC（需要安装 buf、protoc-gen-go、protoc-gen-connect-go）
	cd ../api-specs/sandbox/envd && buf generate
	# 验证编译
	cd sandbox && go build ./...
