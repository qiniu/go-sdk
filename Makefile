.PHONY: test unittest integrationtest staticcheck sync-api-specs generate generate-sandbox sandbox-examples

test:
	go test -tags='unit integration' -failfast -count=1 -v -timeout 350m -coverprofile=coverage.txt `go list ./... | egrep -v 'examples|sms'` | tee -a test.log

unittest:
	go test -tags=unit -failfast -count=1 -v -coverprofile=coverage.txt `go list ./... | egrep -v 'examples|sms'`

integrationtest:
	go test -tags=integration -failfast -count=1 -parallel 1 -v -coverprofile=coverage.txt `go list ./... | egrep -v 'examples|sms'`

staticcheck:
	staticcheck `go list ./... | egrep -v 'examples|sms'`

# 从远端更新 api-specs submodule 到最新提交。
# 运行后请先检查 api-specs 的变更，再运行 make generate 或 make generate-sandbox 并提交。
sync-api-specs:
	git submodule update --init --recursive --remote api-specs

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
		--config sandbox/internal/apis/oapi-codegen.yaml \
		api-specs/sandbox/openapi.yml
	# envd HTTP API
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 \
		--config sandbox/internal/envdapi/oapi-codegen.yaml \
		api-specs/sandbox/envd/envd.yaml
	# envd ConnectRPC（需要安装 buf、protoc-gen-go、protoc-gen-connect-go）
	cd api-specs/sandbox/envd && buf generate
	# 验证编译
	go build ./sandbox/...

sandbox-examples:
	@for dir in examples/sandbox_*/; do \
		name=$$(basename $$dir); \
		echo "=== $$name ==="; \
		go run ./$$dir || exit 1; \
		echo ""; \
	done
