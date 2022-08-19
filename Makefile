build-go:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/api.proto

build-js:
	protoc --js_out=import_style=commonjs:./api --grpc-web_out=import_style=commonjs,mode=grpcwebtext:./api/ api/api.proto