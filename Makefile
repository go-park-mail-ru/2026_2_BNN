.PHONY: generate test cover cover-html

generate:
	go generate ./...

test:
	go test ./...

cover:
	go test -coverprofile=cover.out.tmp ./...
	grep -v '/mocks/' cover.out.tmp > cover.out
	rm cover.out.tmp
	go tool cover -func=cover.out

cover-html: cover
	go tool cover -html=cover.out -o cover.html