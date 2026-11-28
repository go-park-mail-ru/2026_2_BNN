package grpc

//go:generate protoc -I ../../../../.. --go_out=../../../../.. --go_opt=module=bnn --go-grpc_out=../../../../.. --go-grpc_opt=module=bnn ../../../../../proto/onlineconf.proto
