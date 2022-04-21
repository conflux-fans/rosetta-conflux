// reqesut balance of account
// request block
// compare account balance

let request = require('request');
const { promisify, inspect } = require("util");
const BN = require('bn.js');
let prequest = promisify(request);

async function reconcil(user, startBlock, endBlock) {
    let initBalance = await getBalance(user, startBlock - 1);
    let expectBalance = initBalance
    for (let i = startBlock; i <= endBlock; i++) {
        let gotBalance = await getBalance(user, startBlock++);
        let block = await getBlock(i);
        block.transactions.forEach(tx => {
            if (!tx.operations) return
            tx.operations.forEach(op => {
                if (op.account.address != user) return;
                expectBalance = expectBalance.add(new BN(op.amount.value));
            })
        })
        console.log(`expect balance of ${user} at block ${startBlock - 1} is ${expectBalance}, got ${gotBalance}`,);
    }
}

async function getBlock(blockNumber) {
    var options = {
        'method': 'POST',
        'url': 'http://127.0.0.1:8080/block',
        'headers': {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            "network_identifier": {
                "blockchain": "Conflux",
                "network": "Mainnet"
            },
            "block_identifier": {
                "index": blockNumber
            }
        })
    };
    let { body } = await prequest(options);
    let { block } = JSON.parse(body)
    // console.log(`block of ${blockNumber}:`, inspect(block, { depth: 5, colors: true }));
    return block
}

async function getBalance(address, blockNumber) {
    var options = {
        'method': 'POST',
        'url': 'http://127.0.0.1:8080/account/balance',
        'headers': {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            "network_identifier": {
                "blockchain": "Conflux",
                "network": "Mainnet"
            },
            "account_identifier": {
                "address": address
            },
            "block_identifier": {
                "index": blockNumber
            },
            "currencies": [
                {
                    "symbol": "CFX",
                    "decimals": 18
                }
            ]
        })
    };
    const res = await prequest(options)
    const body = JSON.parse(res.body);
    // console.log("body:", body);
    // console.log("body.balances:", body.balances);
    // return

    let { value } = body.balances[0];
    // console.log(`balance of ${blockNumber}:`, value);
    return new BN(value);
}

reconcil("net8888:aak2rra2njvd77ezwjvx04kkds9fzagfe6xm1vavv4", 24462, 24538)