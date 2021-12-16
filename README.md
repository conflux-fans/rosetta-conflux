# 测试用例
1. 普通转账
2. 合约内转账
3. 调用 gas storgae 均被代付的合约
4. 调用合约且导致 storage release
   1. release 目标地址是代付的
   2. release 目标地址是未代付的
5. 内置合约
   1. stake/unstake
   2. kill 合约
   3. 内置合约调用失败