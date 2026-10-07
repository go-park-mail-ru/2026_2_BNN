.PHONY: test cover cover-html

test:
	go test ./...

cover:
	go test -coverprofile=cover.out ./...
	go tool cover -func=cover.out

cover-html: cover
	go tool cover -html=cover.out -o cover.html