test:
	go test -tags='unit integration' -failfast -v -timeout 350m -coverprofile=coverage.txt `go list ./... | egrep -v 'examples|sms'`

unittest:
	go test -tags=unit -failfast -v -coverprofile=coverage.txt `go list ./... | egrep -v 'examples|sms'`

staticcheck:
	staticcheck -go 1.10 `go list ./... | egrep -v 'examples|sms'`
