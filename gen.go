package bnn

//go:generate protoc -I . --go_out=. --go_opt=module=bnn --go-grpc_out=. --go-grpc_opt=module=bnn --grpc-gateway_out=. --grpc-gateway_opt=module=bnn ./proto/auth.proto ./proto/tags.proto
//go:generate protoc -I . --openapiv2_out=./swagger ./proto/auth.proto ./proto/tags.proto
