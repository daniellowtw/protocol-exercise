const PROTO_PATH = '../server/service.proto';
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const packageDefinition = protoLoader.loadSync(
    PROTO_PATH,
    {
        keepCase: true,
        longs: String,
        enums: String,
        defaults: true,
        oneofs: true
    });
const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
const foo = protoDescriptor.service.Foo;

const argv = require('minimist')(process.argv.slice(2));
const port = argv.port
const type = argv.type

// https://grpc.io/docs/languages/node/basics/#client
const client = new foo(`localhost:${port}`, grpc.credentials.createInsecure());

switch (type) {
    case "stateful":
        const id = argv.id ? argv.id : Math.random().toString().slice(2)
        stateful(false, 0, id, argv["total-overwrite"]);
        break
    case "stateless":
        const n = argv["_"][0]
        if (!n) {
            console.error("Missing n")
            process.exit(1)
        }
        stateless(0, n, 0);
        break
    default:
        console.error(`Usage: node index.js --port=<port> --type=stateless <n>`)
        process.exit(1)

}

function stateless(last_msg, totalMessages, sum) {
    console.error("Calling stateless with:", arguments)
    const call = client.stateless({
        last_msg
    });
    call.on('data', function (resp) {
        totalMessages--
        sum += resp.msg
        last_msg = resp.msg
        console.error(resp, totalMessages);
        if (totalMessages === 0) {
            call.cancel();
            // Print the sum
            console.log(sum);
        }
    });
    call.on('error', function (e) {
        if (e.code === 14) { // Unavailable. Most likely networking issue.
            console.error("unavailable")
            setTimeout(() => stateless(last_msg, totalMessages, sum), 3000)
        }
        if (e.code === 1) { // cancelled
            return
        }
        console.error(e)
    });
}

function stateful(isRetry, sum, clientID, totalMsgOverride) {
    const totalMsg = totalMsgOverride ? totalMsgOverride : Math.floor(Math.random() * 65536) + 1
    console.error(`totalMsg: ${totalMsg}`)
    const call = client.stateful({
        "client_id": clientID,
        "total_msg": totalMsg,
        "is_reconnect": isRetry,
    });
    call.on('data', function (resp) {
        sum = (sum + resp.msg) & 0xffffffff
        if (resp.checksum !== 0) {
            if ((resp.checksum & sum) === 0xffffffff) {
                console.log("Checksum mismatch", resp.checksum, sum);
            } else {
                console.log("Checksum matches");
            }
        }
        console.error(resp);
    });
    call.on('error', function (e) {
        if (e.code === 14) { // Unavailable
            setTimeout(() => stateful(true, sum, clientID, totalMsg), 3000)
        }
        if (e.code === 9) { // failed precondition. The server's state is no longer valid
            console.log("Expired. Please retry without reconnecting")
        }
        console.error(e)
    });
}
