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

// # 测试结果
// 1. 调用被赞助的合约且导致 storage release的，estimate gas 有一倍误差
// 2. 调用被赞助的合约且失败时，receipt中 gas/storage 是否被赞助的信息是错误的 （full node bug）

const { Conflux, Contract, format, Drip } = require('js-conflux-sdk');
const { SponsorWhitelistControl } = require('js-conflux-sdk/src/contract/internal')
const path = require('path')
const DestroyableContract = require(path.join(__dirname, './Destroyable.json'));
const testTraceMaterial = require("/Users/wangdayong/myspace/mytemp/demo-truffle/build/contracts/TestTrace")

const cfx = new Conflux({
    url: 'http://127.0.0.1:12537',
    networkId: 1037,

    // url: 'https://test.confluxrpc.com',
    // networkId: 1,
    // logger: console, // for debug
});
const accounts = [];
const contracts = {
    normalContract: undefined,
    sponsoredContract: undefined,
    sponsoredUnaffordContract: undefined,
    sponsorWhitelistControl: undefined,
}
const contractAddrs = {
    normalContract: undefined,
    sponsoredContract: undefined,
    sponsoredUnaffordContract: undefined,
}

const Staking = cfx.InternalContract('Staking');
const AdminControl = cfx.InternalContract('AdminControl');
const SponsorControl = cfx.InternalContract('SponsorWhitelistControl');

async function main() {
    try {
        await init();
        await showSponsorState(contractAddrs.sponsoredUnaffordContract);
        // await transCfxToUser(2);
        // await transCfxToContract(2);
        // return
        // await transCfxToInternalContract(2);
        // await transCfxToNullAddress(2);
        // await invokeContractLeadStorageRelease();
        // await invokeContractSponsored();
        // await invokeSpnsoneredContractLeadStorageRelease();
        await invokeSponsoredUnaffordContract();
        // await stake();
        // await unstake();
        // await destroyContract();
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
    contractAddrs.sponsorWhitelistControl = format.address(SponsorWhitelistControl.address, cfx.networkId)
    contracts.sponsorWhitelistControl = cfx.Contract({ abi: SponsorWhitelistControl.abi, address: contractAddrs.sponsorWhitelistControl })
    if (!contractAddrs.normalContract && !contractAddrs.sponsoredContract) {
        await deploy();
    }
    contracts.normalContract = cfx.Contract({ abi: testTraceMaterial.abi, address: contractAddrs.normalContract });
    contracts.sponsoredContract = cfx.Contract({ abi: testTraceMaterial.abi, address: contractAddrs.sponsoredContract });
    contracts.sponsoredUnaffordContract = cfx.Contract({ abi: testTraceMaterial.abi, address: contractAddrs.sponsoredUnaffordContract });

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

    // accounts1 sponsor for it
    await contracts.sponsorWhitelistControl.setSponsorForGas(contractAddrs.sponsoredContract, 1e15).sendTransaction({ from: accounts[1].address, value: "100000000000000000000" })
    await contracts.sponsorWhitelistControl.setSponsorForCollateral(contractAddrs.sponsoredContract).sendTransaction({ from: accounts[1].address, value: "100000000000000000000" })
    console.log("sponsor for sponsoredContract done")

    let slc = cfx.Contract({ abi: testTraceMaterial.abi, bytecode: testTraceMaterial.bytecode })
    const slcHash = await slc.constructor().sendTransaction({ from: accounts[0].address })
    receipt = await waitReceipt(slcHash)
    contractAddrs.sponsoredUnaffordContract = receipt.contractCreated;
    console.log("deploy sponsoredUnaffordContract done on epoch", receipt.epochNumber)


    // accounts1 sponsor for it
    await contracts.sponsorWhitelistControl.setSponsorForGas(contractAddrs.sponsoredUnaffordContract, 1e6).sendTransaction({ from: accounts[1].address, value: 1e9 }).then(waitReceipt).then(short)
    await contracts.sponsorWhitelistControl.setSponsorForCollateral(contractAddrs.sponsoredUnaffordContract).sendTransaction({ from: accounts[1].address, value: 1e18 / 1024 * 0x140 }).then(waitReceipt).then(short)
    console.log("sponsor for sponsoredUnaffordContract done")

    console.log("deploy done", contractAddrs)
}

async function transCfxToUser(count) {
    for (let i = 0; i < count; i++) {
        console.log("send normal tx:", await cfx.cfx.sendTransaction({ from: accounts[0], to: accounts[2].address, value: 10 }).then(waitReceipt).then(short))
    }
    console.log(`transCfxToUser ${count} times done`)
}

async function transCfxToContract(count) {
    for (let i = 0; i < count; i++) {
        console.log("send cfx to normal contract:", await cfx.cfx.sendTransaction({ from: accounts[0], to: contracts.normalContract.address, value: 60000 }).then(waitReceipt).then(short))
    }
    console.log(`transCfxToContract ${count} times done`)
}

async function transCfxToInternalContract(count) {
    for (let i = 0; i < count; i++) {
        console.log("send cfx to internal contract:", await cfx.cfx.sendTransaction({ from: accounts[0], to: contractAddrs.sponsorWhitelistControl, value: 70000 }).then(waitReceipt).then(short))
    }
    console.log(`transCfxToInternalContract ${count} times done`)
}

async function transCfxToNullAddress(count) {
    for (let i = 0; i < count; i++) {
        console.log("send cfx to null address:", await cfx.cfx.sendTransaction({ from: accounts[0], to: format.address("0x0000000000000000000000000000000000000000", cfx.networkId), value: 80000 }).then(waitReceipt).then(short))
    }
    console.log(`transCfxToNullAddress ${count} times done`)
}

async function invokeContractLeadStorageRelease() {
    console.log("use storage of normal contract", await contracts.normalContract.setSlots([1, 2, 3, 4]).sendTransaction({ from: accounts[0] }).then(waitReceipt).then(short))
    console.log("release storage of normal contract", await contracts.normalContract.setSlots([]).sendTransaction({ from: accounts[0] }).then(waitReceipt).then(short))
}

async function invokeContractSponsored() {
    // 合约有先使用存储又释放存储的操作时，存储抵押预估值为整体使用的值；实际应该为使用的最大值
    console.log("invoke sponsored contract", await contracts.sponsoredContract.newAndDestoryContract().sendTransaction({ from: accounts[0] }).then(waitReceipt).then(short))
}

async function invokeSpnsoneredContractLeadStorageRelease() {
    console.log("use storage of sponored contract", await contracts.sponsoredContract.setSlots([1, 2, 3, 4]).sendTransaction({ from: accounts[0] }).then(waitReceipt).then(short))
    console.log("release storage of sponsored contract and will fail due to out of gas", await contracts.sponsoredContract.setSlots([]).sendTransaction({ from: accounts[0], gas: 22000 }).then(waitReceipt).then(short))
    console.log("release storage of sponsored contract and will success", await contracts.sponsoredContract.setSlots([]).sendTransaction({ from: accounts[0] }).catch(console.error).then(waitReceipt).then(short))
}

async function invokeSponsoredUnaffordContract() {
    await showSponsorState(contractAddrs.sponsoredUnaffordContract)
    console.log("use storage and gas of sponored little contract which can afford both gas and storage", await contracts.sponsoredUnaffordContract.setSlots([1, 2, 3, 4]).sendTransaction({ from: accounts[0], gasPrice: 1, storageLimit: 0x140 }).then(waitReceipt).then(sponsorResult))
    await showSponsorState(contractAddrs.sponsoredUnaffordContract)
    console.log("use storage and gas of sponored little contract which can afford gas but not storage", await contracts.sponsoredUnaffordContract.setSlots([1, 2, 3, 4, 5, 6, 7, 8]).sendTransaction({ from: accounts[0], gasPrice: 1 }).then(waitReceipt).then(sponsorResult))
    console.log("release storage", await contracts.sponsoredUnaffordContract.setSlots([]).sendTransaction({ from: accounts[0], gasPrice: 1 }).then(waitReceipt).then(short))
    await showSponsorState(contractAddrs.sponsoredUnaffordContract)
    console.log("use storage and gas of sponored little contract which can afford storage but not gas", await contracts.sponsoredUnaffordContract.setSlots([1, 2, 3, 4]).sendTransaction({ from: accounts[0], gasPrice: 10000, storageLimit: 0x140 }).then(waitReceipt).then(sponsorResult))
    await showSponsorState(contractAddrs.sponsoredUnaffordContract)
    console.log("use storage and gas of sponored little contract which un-afford both", await contracts.sponsoredUnaffordContract.setSlots([1, 2, 3, 4, 5, 6, 7, 8]).sendTransaction({ from: accounts[0], gasPrice: 10000, storageLimit: 0x140 }).then(waitReceipt).then(sponsorResult))
}

async function showSponsorState(target) {
    const gasSponsored = await contracts.sponsorWhitelistControl.getSponsoredBalanceForGas(target)
    const storageSponsored = await contracts.sponsorWhitelistControl.getSponsoredBalanceForCollateral(target)
    console.log(`sponsor state of ${target}`, gasSponsored, storageSponsored)
}

async function stake() {
    const receipt = await Staking
        .deposit(Drip.fromCFX(5))
        .sendTransaction({
            from: accounts[0].address,
        }).executed();

    console.log("Stake result", short(receipt));
}

async function unstake() {
    const receipt = await Staking
        .withdraw(Drip.fromCFX(3))
        .sendTransaction({
            from: accounts[0].address,
        })
        .executed();
    console.log("Unstake result", short(receipt));
}

async function destroyContract() {
    // deploy contract
    let contract = cfx.Contract(DestroyableContract);
    let receipt = await contract.constructor().sendTransaction({
        from: accounts[0].address,
    }).executed();
    console.log("deploy DestroyableContract", short(receipt));

    //
    const address = receipt.contractCreated;
    contract = cfx.Contract({
        abi: DestroyableContract.abi,
        address,
    });
    // set sponsor
    receipt = await SponsorControl.setSponsorForGas(address, 1e15).sendTransaction({
        from: accounts[1].address,
        value: Drip.fromCFX(1),
    }).executed();
    console.log("set gas sponsor for DestroyableContract", short(receipt));

    receipt = await SponsorControl.setSponsorForCollateral(address).sendTransaction({
        from: accounts[1].address,
        value: Drip.fromCFX(10),
    }).executed();
    console.log("set collateral sponsor for DestroyableContract", short(receipt));

    // use storage
    await contract.setVal(123).sendTransaction({
        from: accounts[0].address,
    }).executed();
    console.log("use storage of DestroyableContract", short(receipt));

    // destroy
    await AdminControl.destroy(address).sendTransaction({
        from: accounts[0].address,
    }).executed();
    console.log("destroy DestroyableContract", short(receipt));
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

function short(receipt) {
    const { transactionHash, outcomeStatus, txExecErrorMsg } = receipt
    return `${transactionHash} ${outcomeStatus == "0x1" ? txExecErrorMsg : "ok"}`
}

function sponsorResult(receipt) {
    const { storageCollateralized,
        storageCoveredBySponsor,
        storageReleased,
        gasCoveredBySponsor,
        gasFee } = receipt
    return short(receipt) + ` storageCollateralized: ${storageCollateralized}, storageCoveredBySponsor: ${storageCoveredBySponsor}, storageReleased: ${storageReleased}, gasCoveredBySponsor: ${gasCoveredBySponsor}, gasFee: ${gasFee}`
}

process.on('unhandledRejection', (reason, p) => {
    console.log('Unhandled Rejection at: Promise', p, 'reason:', reason)
    // application specific logging, throwing an error, or other logic here
}).on('uncaughtException', err => {
    console.log('Uncaught Exception thrown:', err)
    // application specific logging, throwing an error, or other logic here
})

main()
