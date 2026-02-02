```
go mod init github.com/cgalvisleon/josefina
go get github.com/cgalvisleon/et@v0.0.2
go get github.com/gorilla/websocket
git remote add origin https://github.com/cgalvisleon/josefina.git
```

# Server

```
gofmt -w . && go run ./cmd/server -port 3500 -rpc 4300
gofmt -w . && go run ./cmd/server -port 3501 -rpc 4301
gofmt -w . && go run ./cmd/server -port 3502 -rpc 4302
```

SELECT A.\_DATA||jsonb_build_object('date_make', A.DATE_MAKE,
'date_update', A.DATE_UPDATE,
'\_state', A.\_STATE,
'\_id', A.\_ID,
'device', A.DEVICE,
'app', A.APP,
'client_id', A.CLIENT_ID,
'type_service', A.TYPE_SERVICE,
'index', A.INDEX,
'\_idt', A.\_IDT) AS \_DATA
FROM services.SERVICES AS A
WHERE A.\_ID = ''
LIMIT 1;
