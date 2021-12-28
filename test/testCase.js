// # 测试用例
// 1. 普通转账
// 2. 合约内转账
// 3. 调用 gas storgae 均被代付的合约
// 4. 调用合约且导致 storage release
//    1. release 目标地址是代付的
//    2. release 目标地址是未代付的
// 5. 内置合约
//    1. stake/unstake
//    2. kill 合约
//    3. 内置合约调用失败
// 6. 给内置合约转账
// 7. 给0地址转账
// 8. 合约内部转账失败
//    1. 转cfx成功
//    2. 转cfx失败
// 9. 替换合约的sponsor
// 10. pos reward
// 11. gas 返还的情况
// 12. 合约调用外层失败内层成功

// TODO: 
// 1. 在多节点环境下测试


// # 发现的问题
// 1. 调用被赞助的合约且导致 storage release的，estimate gas 有一倍误差 （未复现）
// 2. 调用被赞助的合约且失败时，receipt中 gas/storage 是否被赞助的信息是错误的 （full node bug CIP-78，辰星修改后在trace中体现)
// 3. 被赞助的合约销毁时，gas/storage 返还没有在 trace 中提现
// 4. 合约有先使用存储又释放存储的操作时，存储抵押预估值为整体使用的值；实际应该为使用的最大值
// 5. js-conflux-sdk 给合约转账不estimte

const { Conflux, Contract, format, Drip } = require('js-conflux-sdk');
const config = require('./config')
const path = require('path')
const fs = require('fs')
const TestRosettaMeta = require(path.join(__dirname, './build/contracts/TestRosetta.json'));
const cfx = new Conflux({
    url: 'http://127.0.0.1:12537',
    networkId: 1037,

    // url: 'https://test.confluxrpc.com',
    // networkId: 1,

    // logger: console, // for debug
});

const accounts = [];
let contracts = {
    normalContract: undefined,
    sponsoredContract: undefined,
    // for testing cip-78 "In whitelist but sponsor cannot afford" case
    sponsoredUnaffordContract: undefined,
    destroyContract: undefined,
}
let contractAddrs = {
    normalContract: undefined,
    sponsoredContract: undefined,
    sponsoredUnaffordContract: undefined,
    destroyContract: undefined,
}

const Staking = cfx.InternalContract('Staking');
const AdminControl = cfx.InternalContract('AdminControl');
const SponsorControl = cfx.InternalContract('SponsorWhitelistControl');

async function main() {
    try {
        await init();
        // await showSponsorState(contractAddrs.sponsoredUnaffordContract);
        // await transCfxToUser(2);
        // await transCfxToContract(2);
        // await transCfxToInternalContract(2);
        // await transCfxToNullAddress(2);
        // await invokeContractLeadStorageRelease();
        // await invokeContractSponsored();
        // await invokeSpnsoneredContractLeadStorageRelease();
        // await invokeSponsoredUnaffordContract();
        // await replaceSponsor();
        // await stakeUnstake();
        await internalTransferCfx();
        await gasRefund();
        return


        // TODO: wait full-node fix cip-78
        await failedToinvokeSpnsoneredContractLeadStorageRelease();
        // TODO: wait full-node fix issue 3
        await destroyContract();

    } catch (e) {
        console.error("error:", e)
    }
}

async function init() {
    accounts.push(cfx.wallet.addPrivateKey("0xa32f1f94134be66e784230ff4813b8a1e79e5d521ca2f1be4f69d2f4a368638a"));
    accounts.push(cfx.wallet.addPrivateKey("0xb32f1f94134be66e784230ff4813b8a1e79e5d521ca2f1be4f69d2f4a368638b"));
    accounts.push(cfx.wallet.addPrivateKey("0xc32f1f94134be66e784230ff4813b8a1e79e5d521ca2f1be4f69d2f4a368638c"));

    let getBalances = accounts.map(a => cfx.getBalance(a.address));
    let balances = await Promise.all(getBalances);

    console.log("accounts", accounts.map(a => a.address), balances)
    // return
    await deploy();

    contracts.normalContract = cfx.Contract({ abi: TestRosettaMeta.abi, address: contractAddrs.normalContract });
    contracts.sponsoredContract = cfx.Contract({ abi: TestRosettaMeta.abi, address: contractAddrs.sponsoredContract });
    contracts.sponsoredUnaffordContract = cfx.Contract({ abi: TestRosettaMeta.abi, address: contractAddrs.sponsoredUnaffordContract });
    contracts.destroyContract = cfx.Contract({ abi: TestRosettaMeta.abi, address: contractAddrs.destroyContract });

    console.log("init done\n")
}

async function deploy() {
    config[cfx.networkId] = config[cfx.networkId] || {}
    contractAddrs = config[cfx.networkId]

    if (!contractAddrs.normalContract) {
        let nc = cfx.Contract({ abi: TestRosettaMeta.abi, bytecode: TestRosettaMeta.bytecode })
        const ncHash = await nc.constructor().sendTransaction({ from: accounts[0].address })
        let { contractCreated, epochNumber } = await waitReceipt(ncHash)
        contractAddrs.normalContract = contractCreated
        console.log("deploy normalContract done on epoch", epochNumber)
    }


    if (!contractAddrs.sponsoredContract) {
        let sc = cfx.Contract({ abi: TestRosettaMeta.abi, bytecode: TestRosettaMeta.bytecode })
        const scHash = await sc.constructor().sendTransaction({ from: accounts[0].address })
        let receipt = await waitReceipt(scHash)
        contractAddrs.sponsoredContract = receipt.contractCreated;
        console.log("deploy sponsoredContract done on epoch", receipt.epochNumber)

        // accounts1 sponsor for it
        console.log("sponsor gas for sponsoredContract done", await SponsorControl.setSponsorForGas(contractAddrs.sponsoredContract, 1e10).sendTransaction({ from: accounts[1].address, value: 1e17 }))
        console.log("sponsor storage for sponsoredContract done", await SponsorControl.setSponsorForCollateral(contractAddrs.sponsoredContract).sendTransaction({ from: accounts[1].address, value: 1e18 }))
    }


    if (!contractAddrs.sponsoredUnaffordContract) {
        let slc = cfx.Contract({ abi: TestRosettaMeta.abi, bytecode: TestRosettaMeta.bytecode })
        const slcHash = await slc.constructor().sendTransaction({ from: accounts[0].address })
        let receipt = await waitReceipt(slcHash)
        contractAddrs.sponsoredUnaffordContract = receipt.contractCreated;
        console.log("deploy sponsoredUnaffordContract done on epoch", receipt.epochNumber)

        // accounts1 sponsor for it
        console.log("sponsor gas for sponsoredUnaffordContract", await SponsorControl.setSponsorForGas(contractAddrs.sponsoredUnaffordContract, 1e6).sendTransaction({ from: accounts[1].address, value: 1e9 }).then(waitReceipt).then(shortReceipt))
        console.log("sponsor storage for sponsoredUnaffordContract", await SponsorControl.setSponsorForCollateral(contractAddrs.sponsoredUnaffordContract).sendTransaction({ from: accounts[1].address, value: 1e18 / 1024 * 0x140 }).then(waitReceipt).then(shortReceipt))
    }

    // deploy contract will test for destroy
    let dc = cfx.Contract({ abi: TestRosettaMeta.abi, bytecode: TestRosettaMeta.bytecode })
    const dcHash = await dc.constructor().sendTransaction({ from: accounts[0].address })
    let receipt = await waitReceipt(dcHash)
    contractAddrs.destroyContract = receipt.contractCreated;
    console.log("deploy contract for testing destroy sponsored contract done on epoch", receipt.epochNumber)

    // accounts1 sponsor for it
    console.log("sponsor gas for destroyContract done", await SponsorControl.setSponsorForGas(contractAddrs.destroyContract, 1e10).sendTransaction({ from: accounts[1].address, value: 1e17 }))
    console.log("sponsor storage for destroyContract done", await SponsorControl.setSponsorForCollateral(contractAddrs.destroyContract).sendTransaction({ from: accounts[1].address, value: 1e18 }))

    saveConfig(config)
}

async function transCfxToUser(count) {
    for (let i = 0; i < count; i++) {
        console.log("send normal tx:", await cfx.cfx.sendTransaction({ from: accounts[0], to: accounts[2].address, value: 10 }).then(waitReceipt).then(shortReceipt))
    }
    console.log(`transCfxToUser ${count} times done\n`)
}

async function transCfxToContract(count) {
    for (let i = 0; i < count; i++) {
        console.log("send cfx to normal contract:", await cfx.cfx.sendTransaction({ from: accounts[0], to: contracts.normalContract.address, value: 60000 }).then(waitReceipt).then(shortReceipt))
    }
    console.log(`transCfxToContract ${count} times done\n`)
}

async function transCfxToInternalContract(count) {
    for (let i = 0; i < count; i++) {
        console.log("send cfx to internal contract:", await cfx.cfx.sendTransaction({ from: accounts[0], to: format.address("0x0888000000000000000000000000000000000001", cfx.networkId), value: 70000 }).then(waitReceipt).then(shortReceipt))
    }
    console.log(`transCfxToInternalContract ${count} times done\n`)
}

async function transCfxToNullAddress(count) {
    for (let i = 0; i < count; i++) {
        console.log("send cfx to null address:", await cfx.cfx.sendTransaction({ from: accounts[0], to: format.address("0x0000000000000000000000000000000000000000", cfx.networkId), value: 80000 }).then(waitReceipt).then(shortReceipt))
    }
    console.log(`transCfxToNullAddress ${count} times done\n`)
}

async function invokeContractLeadStorageRelease() {
    console.log("\nuse storage of normal contract", await contracts.normalContract.setSlots([1, 2, 3, 4]).sendTransaction({ from: accounts[0] }).then(waitReceipt).then(shortReceipt))
    console.log("release storage of normal contract", await contracts.normalContract.setSlots([]).sendTransaction({ from: accounts[0] }).then(waitReceipt).then(shortReceipt))
}

async function invokeContractSponsored() {
    // TODO: full-node bug: 合约有先使用存储又释放存储的操作时，存储抵押预估值为整体使用的值；实际应该为使用的最大值
    // TODO: 为避免错误，暂时手动设置storageLimit
    console.log("\ninvoke sponsored contract", await contracts.sponsoredContract.newAndDestoryContract().sendTransaction({ from: accounts[0], storageLimit: 1000 }).then(waitReceipt).then(shortReceipt))
}

async function invokeSpnsoneredContractLeadStorageRelease() {
    console.log("\nuse storage of sponored contract", await contracts.sponsoredContract.setSlots([1, 2, 3, 4]).sendTransaction({ from: accounts[0] }).then(waitReceipt).then(shortReceipt))
    const { gasLimit, storageCollateralized } = await contracts.sponsoredContract.setSlots([]).estimateGasAndCollateral()
    console.log("release storage of sponsored contract and will success", await contracts.sponsoredContract.setSlots([]).sendTransaction({ from: accounts[0], storageLimit: storageCollateralized }).catch(console.error).then(waitReceipt).then(shortReceipt))
}

async function failedToinvokeSpnsoneredContractLeadStorageRelease() {
    console.log("\nuse storage of sponored contract", await contracts.sponsoredContract.setSlots([1, 2, 3, 4]).sendTransaction({ from: accounts[0] }).then(waitReceipt).then(shortReceipt))
    // FIXME: 有时estimate错误X
    const { gasLimit, storageCollateralized } = await contracts.sponsoredContract.setSlots([]).estimateGasAndCollateral()
    console.log("release storage of sponsored contract and will fail due to out of gas", await contracts.sponsoredContract.setSlots([]).sendTransaction({ from: accounts[0], gas: 22000, storageLimit: storageCollateralized }).then(waitReceipt).then(shortReceipt))
}

async function invokeSponsoredUnaffordContract() {
    await showSponsorState(contractAddrs.sponsoredUnaffordContract)
    console.log("\nuse storage and gas of sponored un-afford contract which can afford both gas and storage", await contracts.sponsoredUnaffordContract.setSlots([1, 2, 3, 4]).sendTransaction({ from: accounts[0], gasPrice: 1, storageLimit: 0x140 }).then(waitReceipt).then(getSponsorResult))
    await showSponsorState(contractAddrs.sponsoredUnaffordContract)
    console.log("use storage and gas of sponored un-afford contract which can afford gas but not storage", await contracts.sponsoredUnaffordContract.setSlots([1, 2, 3, 4, 5, 6, 7, 8]).sendTransaction({ from: accounts[0], gasPrice: 1 }).then(waitReceipt).then(getSponsorResult))
    console.log("release storage", await contracts.sponsoredUnaffordContract.setSlots([]).sendTransaction({ from: accounts[0], gasPrice: 1 }).then(waitReceipt).then(shortReceipt))
    await showSponsorState(contractAddrs.sponsoredUnaffordContract)
    console.log("use storage and gas of sponored un-afford contract which can afford storage but not gas", await contracts.sponsoredUnaffordContract.setSlots([1, 2, 3, 4]).sendTransaction({ from: accounts[0], gasPrice: 10000, storageLimit: 0x140 }).then(waitReceipt).then(getSponsorResult))
    await showSponsorState(contractAddrs.sponsoredUnaffordContract)
    console.log("use storage and gas of sponored un-afford contract which un-afford both", await contracts.sponsoredUnaffordContract.setSlots([1, 2, 3, 4, 5, 6, 7, 8]).sendTransaction({ from: accounts[0], gasPrice: 10000, storageLimit: 0x140 }).then(waitReceipt).then(getSponsorResult))
}

async function replaceSponsor() {
    console.log("\nreplace gas sponsor for sponsoredContract done", await SponsorControl.setSponsorForGas(contractAddrs.sponsoredContract, 1e10).sendTransaction({ from: accounts[2].address, value: 1e18 }))
    console.log("replace storage sponsor for sponsoredContract done", await SponsorControl.setSponsorForCollateral(contractAddrs.sponsoredContract).sendTransaction({ from: accounts[2].address, value: 1e19 }))
}

async function destroyContract() {
    console.log("\ndestroy sponsored contract", await contracts.sponsoredContract.destroy(accounts[1].address).sendTransaction({ from: accounts[0] }).then(waitReceipt).then(shortReceipt))
}

async function stakeUnstake() {
    let receipt = await Staking.deposit(Drip.fromCFX(5)).sendTransaction({ from: accounts[0].address }).executed();
    console.log("\nStake result", shortReceipt(receipt));

    receipt = await Staking.withdraw(Drip.fromCFX(3)).sendTransaction({ from: accounts[0].address }).executed();
    console.log("Unstake result", shortReceipt(receipt));
}

async function internalTransferCfx() {
    let receipt = await contracts.normalContract.mustInternalOk().sendTransaction({ from: accounts[0].address, value: Drip.fromCFX(1) }).executed();
    console.log('\nInternal transfer success', shortReceipt(receipt));

    receipt = await contracts.normalContract.mustInternalFail().sendTransaction({ from: accounts[0].address }).executed();
    console.log('Internal transfer fail', shortReceipt(receipt));
}

async function gasRefund(count) {
    console.log("send normal tx and will lead gas refund:", await cfx.cfx.sendTransaction({ from: accounts[1], to: accounts[2].address, value: 10, gas: 30000 }).then(waitReceipt).then(shortReceipt))
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

function shortReceipt(receipt) {
    const { transactionHash, outcomeStatus, txExecErrorMsg } = receipt
    return `${transactionHash} ${outcomeStatus == "0x1" ? txExecErrorMsg : "ok"}`
}

async function showSponsorState(target) {
    const gasSponsored = await SponsorControl.getSponsoredBalanceForGas(target)
    const storageSponsored = await SponsorControl.getSponsoredBalanceForCollateral(target)
    console.log(`sponsor state of ${target}`, gasSponsored, storageSponsored)
}

function getSponsorResult(receipt) {
    const { storageCollateralized, storageCoveredBySponsor, storageReleased, gasCoveredBySponsor, gasFee } = receipt
    return shortReceipt(receipt) + ` storageCollateralized: ${storageCollateralized}, storageCoveredBySponsor: ${storageCoveredBySponsor}, gasCoveredBySponsor: ${gasCoveredBySponsor}, gasFee: ${gasFee}`
}

function saveConfig(config) {
    if (cfx.networkId == 1 || cfx.networkId == 1029) {
        fs.writeFileSync("./config.json", JSON.stringify(config, null, 2))
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
