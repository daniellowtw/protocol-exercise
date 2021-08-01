# Protocol exercise

This contains a go server and nodejs client implementation of the [protocol](./protocol.md).
The client and server communicate using gRPC with the given [protobuf definition](./server/service.proto).

## Generate proto

The following will generate the server stub.

```pwsh
cd server
./genproto.ps1
```

## Server

To build the server, run the following.
```
cd server
go build main.go
```

To run the server, give it a port to listen to.
```
./server <port>
```

## Nodejs Client

To run the client, run the following:
```
cd client
node index.js --port <port> --type stateless <n>
node index.js --port <port> --type stateful
```

For testing purposes, the following can be configured on the CLI:
```
node index.js --port <port> --type stateful --id <client_id> --total_overwrite <totalMsg>
```
