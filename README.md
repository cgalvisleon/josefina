```
go mod init github.com/cgalvisleon/josefina
go get github.com/cgalvisleon/et@v0.0.2
git remote add origin https://github.com/cgalvisleon/josefina.git
```

# Server

```
gofmt -w . && go run ./cmd/server -port 3500 -rpc 4200
gofmt -w . && go run ./cmd/server -port 3501 -rpc 4201
gofmt -w . && go run ./cmd/server -port 3502 -rpc 4202
```
