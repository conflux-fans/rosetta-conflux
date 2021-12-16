// # 测试用例
// 1. 普通转账
// 2. 合约内转账
// 3. 调用 gas storgae 均被代付的合约
// 4. 调用合约且导致 storage release
//    1. release 目标地址是代付的
//    2. release 目标地址是未代付的

const { Conflux, Contract, format } = require('js-conflux-sdk');
const { SponsorWhitelistControl } = require('js-conflux-sdk/src/contract/internal')
const testTraceMaterial = require("/Users/wangdayong/myspace/mytemp/demo-truffle/build/contracts/TestTrace")

const cfx = new Conflux({
    url: 'http://127.0.0.1:12537',
    networkId: 1037,
    // logger: console, // for debug
});
const accounts = [];
const contracts = {
    normalContract: undefined,
    sponsoredContract: undefined,
    sponsorWhitelistControl: undefined,
}
const contractAddrs = {
    normalContract: undefined,
    sponsoredContract: undefined,
}

async function main() {
    try {
        await init();
        // await transCfxToUser(10);
        // await transCfxToContract(10);
        // await invokeContractLeadStorageRelease();
        // await invokeContractSponsored();
        // await invokeSpnsoneredContractLeadStorageRelease();
    } catch (e) {
        console.error("error:", e)
    }
}

async function init() {
    accounts.push(cfx.wallet.addPrivateKey("0x2139FB4C55CB9AF7F0086CD800962C2E9013E2292BAE77978A9209E3BEE71D49"));
    accounts.push(cfx.wallet.addPrivateKey("0xd32f1f94134be66e784230ff4813b8a1e79e5d521ca2f1be4f69d2f4a3686380"));
    accounts.push(cfx.wallet.addPrivateKey("0xe32f1f94134be66e784230ff4813b8a1e79e5d521ca2f1be4f69d2f4a3686381"))
    console.log("accounts", accounts)
    // return

    contracts.sponsorWhitelistControl = cfx.Contract({ abi: SponsorWhitelistControl.abi, address: format.address(SponsorWhitelistControl.address, cfx.networkId) })

    if (!contractAddrs.normalContract && !contractAddrs.sponsoredContract) {
        await deploy();
    }
    contracts.normalContract = cfx.Contract({ abi: testTraceMaterial.abi, address: contractAddrs.normalContract });
    contracts.sponsoredContract = cfx.Contract({ abi: testTraceMaterial.abi, address: contractAddrs.sponsoredContract });

    console.log("init done")
}

async function deploy() {
    let nc = cfx.Contract({ abi: testTraceMaterial.abi, bytecode: testTraceMaterial.bytecode })
    const ncHash = await nc.constructor().sendTransaction({ from: accounts[0].address })
    let { contractCreated, epochNumber } = await waitReceipt(ncHash)
    contractAddrs.normalContract = contractCreated
    console.log("deploy normalContract done on epoch", epochNumber)

    let sc = cfx.Contract({ abi: testTraceMaterial.abi, bytecode: testTraceMaterial.bytecode })
    const scHash = await sc.constructor().sendTransaction({ from: accounts[0].address })
    let receipt = await waitReceipt(scHash)
    contractAddrs.sponsoredContract = receipt.contractCreated;
    console.log("deploy sponsoredContract done on epoch", receipt.epochNumber)
    console.log("deploy done", contractAddrs)

    // accounts1 sponsor for it
    await contracts.sponsorWhitelistControl.setSponsorForGas(contractAddrs.sponsoredContract, 1e15).sendTransaction({ from: accounts[1].address, value: "100000000000000000000" })
    await contracts.sponsorWhitelistControl.setSponsorForCollateral(contractAddrs.sponsoredContract).sendTransaction({ from: accounts[1].address, value: "100000000000000000000" })
    console.log("sponsor done")
}

async function transCfxToUser(count) {
    for (let i = 0; i < count; i++) {
        console.log("send normal tx:", await cfx.cfx.sendTransaction({ from: accounts[0], to: accounts[2].address, value: 10 }))
    }
    console.log(`transCfxToUser ${count} times done`)
}

async function transCfxToContract(count) {
    for (let i = 0; i < count; i++) {
        await cfx.cfx.sendTransaction({ from: accounts[0], to: contracts.normalContract.address, value: 60000 })
    }
    console.log(`transCfxToContract ${count} times done`)
}

async function invokeContractLeadStorageRelease() {
    console.log("use storage of normal contract", await contracts.normalContract.setSlots([1, 2, 3, 4]).sendTransaction({ from: accounts[0] }))
    console.log("release storage of normal contract", await contracts.normalContract.setSlots([]).sendTransaction({ from: accounts[0] }))
}

async function invokeContractSponsored() {
    console.log("invoke sponsored contract", await contracts.sponsoredContract.newAndDestoryContract().sendTransaction({ from: accounts[0] }))
}

async function invokeSpnsoneredContractLeadStorageRelease() {
    console.log("use storage of sponored contract", await contracts.sponsoredContract.setSlots([1, 2, 3, 4]).sendTransaction({ from: accounts[0] }))
    console.log("release storage of sponsored contract", await contracts.sponsoredContract.setSlots([]).sendTransaction({ from: accounts[0] }))
}

async function waitReceipt(txhash) {
    while (true) {
        let receipt = await cfx.getTransactionReceipt(txhash)
        if (receipt) {
            return receipt
        }
        await new Promise(resolve => setTimeout(resolve, 1000))
    }
}

process.on('unhandledRejection', (reason, p) => {
    console.log('Unhandled Rejection at: Promise', p, 'reason:', reason)
    // application specific logging, throwing an error, or other logic here
}).on('uncaughtException', err => {
    console.log('Uncaught Exception thrown:', err)
    // application specific logging, throwing an error, or other logic here
})

main()
