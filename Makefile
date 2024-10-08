test:
	go test -tags='unit integration' -failfast -count=1 -v -timeout 350m -coverprofile=coverage.txt `go list ./... | egrep -v 'examples|sms'`

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
