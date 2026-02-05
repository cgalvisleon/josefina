```
goenv local 1.23.0
go mod init github.com/cgalvisleon/josefina
go get github.com/cgalvisleon/et@v1.0.14
go get github.com/gorilla/websocket
git remote add origin https://github.com/cgalvisleon/josefina.git
```

# Server

```
gofmt -w . && go run ./cmd/server -port 3500 -rpc 4300
gofmt -w . && go run ./cmd/server -port 3501 -rpc 4301
gofmt -w . && go run ./cmd/server -port 3502 -rpc 4302
```
