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

// # 测试结果
// 1. 调用被赞助的合约且导致 storage release的，estimate gas 有一倍误差
// 2. hardfork前；调用被赞助的合约且失败时，receipt中 gas/storage 是否被赞助的信息是错误的

const { Conflux, Contract, format, Drip } = require('js-conflux-sdk');
const { SponsorWhitelistControl } = require('js-conflux-sdk/src/contract/internal')
const path = require('path')
// const testTraceMaterial = require("/Users/wangdayong/myspace/mytemp/demo-truffle/build/contracts/TestTrace")
const DestroyableContractMeta = require(path.join(__dirname, './contracts/Destroyable.json'));
const EmptyContractMeta = require(path.join(__dirname, './contracts/Empty.json'));
const TransferUtilMeta = require(path.join(__dirname, './contracts/TransferUtil.json'));

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

const Staking = cfx.InternalContract('Staking');
const AdminControl = cfx.InternalContract('AdminControl');
const SponsorControl = cfx.InternalContract('SponsorWhitelistControl');

async function main() {
    try {
        await init();
        await transCfxToUser(10);
        await transCfxToContract(10);
        await invokeContractLeadStorageRelease();
        await invokeContractSponsored();
        await invokeSpnsoneredContractLeadStorageRelease();
        await stake();
        await unstake();
        // await internalTransferCfx();
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
        console.log("send normal tx:", await cfx.cfx.sendTransaction({ from: accounts[0], to: contracts.normalContract.address, value: 60000 }))
    }
    console.log(`transCfxToContract ${count} times done`)
}

async function invokeContractLeadStorageRelease() {
    console.log("use storage of normal contract", await contracts.normalContract.setSlots([1, 2, 3, 4]).sendTransaction({ from: accounts[0] }).then(waitReceipt).then(short))
    console.log("release storage of normal contract", await contracts.normalContract.setSlots([]).sendTransaction({ from: accounts[0] }).then(waitReceipt).then(short))
}

async function invokeContractSponsored() {
    console.log("invoke sponsored contract", await contracts.sponsoredContract.newAndDestoryContract().sendTransaction({ from: accounts[0] }).then(waitReceipt).then(short))
}

async function invokeSpnsoneredContractLeadStorageRelease() {
    console.log("use storage of sponored contract", await contracts.sponsoredContract.setSlots([1, 2, 3, 4]).sendTransaction({ from: accounts[0] }).then(waitReceipt).then(short))
    console.log("release storage of sponsored contract and will fail due to out of gas", await contracts.sponsoredContract.setSlots([]).sendTransaction({ from: accounts[0], gas: 22000, storageLimit: 100 }).then(waitReceipt).then(short))
    console.log("release storage of sponsored contract and will success", await contracts.sponsoredContract.setSlots([]).sendTransaction({ from: accounts[0], gas: 220000 }).catch(console.error).then(waitReceipt).then(short))
}

async function stake() {
    const receipt = await Staking
        .deposit(Drip.fromCFX(5))
        .sendTransaction({
            from: accounts[0].address,
        }).executed();

    console.log("Stake result:", short(receipt));
}

async function unstake() {
    const receipt = await Staking
        .withdraw(Drip.fromCFX(3))
        .sendTransaction({
            from: accounts[0].address,
        })
        .executed();
    console.log("Unstake result:", short(receipt));
}

async function destroyContract() {
    // deploy contract
    let contract = cfx.Contract(DestroyableContractMeta);
    let receipt = await contract.constructor().sendTransaction({
        from: accounts[0].address,
    }).executed();
    console.log(`result: ${receipt.outcomeStatus == 0 ? 'success' : 'fail'}`);
    console.log(receipt.transactionHash);

    //
    const address = receipt.contractCreated;
    contract = cfx.Contract({
        abi: DestroyableContractMeta.abi,
        address,
    });
    // set sponsor
    receipt = await SponsorControl.setSponsorForGas(address, 1e15).sendTransaction({
        from: accounts[1].address,
        value: Drip.fromCFX(1),
    }).executed();

    receipt = await SponsorControl.setSponsorForCollateral(address).sendTransaction({
        from: accounts[1].address,
        value: Drip.fromCFX(10),
    }).executed();

    // use storage
    await contract.setVal(123).sendTransaction({
        from: accounts[0].address,
    }).executed();

    // destroy
    await AdminControl.destroy(address).sendTransaction({
        from: accounts[0].address,
    }).executed();
}

async function deployInternalTransferContracts() {
  console.log('Deploying TransferUtil...');
  const transferReceipt = await cfx.Contract(TransferUtilMeta).constructor().sendTransaction({
    from: accounts[0].address,
  }).executed();

  console.log('Deploying Destroyable...');
  const destroyableReceipt = await cfx.Contract(DestroyableContractMeta).constructor().sendTransaction({
    from: accounts[0].address,
  }).executed();

  console.log('Deploying Empty...');
  const emptyReceipt = await cfx.Contract(EmptyContractMeta).constructor().sendTransaction({
    from: accounts[0].address,
  }).executed();

  return {
    emptyContract: emptyReceipt.contractCreated,
    transferContract: transferReceipt.contractCreated,
    destroyableContract: destroyableReceipt.contractCreated,
  }
}

async function internalTransferCfx() {
  let addresses = await deployInternalTransferContracts();
  let transferContract = cfx.Contract({
    address: addresses.transferContract,
    abi: TransferUtilMeta.abi,
  });

  let receipt = await transferContract.transfer(addresses.destroyableContract, Drip.fromCFX(1)).sendTransaction({
    from: accounts[0].address,
    value: Drip.fromCFX(1),
  }).executed();
  console.log('Internal transfer', receipt.outcomeStatus == 0 ? 'success' : 'fail', receipt.transactionHash);

  receipt = await transferContract.transfer(addresses.emptyContract, Drip.fromCFX(1)).sendTransaction({
    gas: 500000,
    storageLimit: 100,
    from: accounts[0].address,
    value: Drip.fromCFX(1),
  }).executed();
  console.log('Internal transfer', receipt.outcomeStatus == 0 ? 'success' : 'fail');
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


process.on('unhandledRejection', (reason, p) => {
    console.log('Unhandled Rejection at: Promise', p, 'reason:', reason)
    // application specific logging, throwing an error, or other logic here
}).on('uncaughtException', err => {
    console.log('Uncaught Exception thrown:', err)
    // application specific logging, throwing an error, or other logic here
})

main()
