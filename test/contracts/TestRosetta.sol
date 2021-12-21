// SPDX-License-Identifier: MIT
pragma solidity >=0.4.22 <0.9.0;
import "@confluxfans/contracts/internalContracts/InternalContractsLib.sol";

contract TestTraceA {
    function callB(TestTraceB b) public {
        b.foo();
    }
}

contract TestTraceB {
    function fooPure() public pure {}

    function foo() public {}

    function destory() public {
        selfdestruct(payable(msg.sender));
    }
}

contract TestRosetta {
    constructor() {
        address[] memory whitelist = new address[](1);
        whitelist[0] = address(0);
        InternalContracts.SPONSOR_CONTROL.addPrivilege(whitelist);
    }

    receive() external payable {}

    mapping(address => uint256[]) slots;

    function newContractTwiceAndCall() public {
        TestTraceB b = new TestTraceB();
        TestTraceA a = new TestTraceA();
        a.callB(b);
    }

    function newAndDestoryContract() public {
        TestTraceB b = new TestTraceB();
        b.destory();
    }

    function setSlots(uint256[] memory value) public {
        slots[msg.sender] = value;
    }

    function destroy(address receiver) public {
        address payable target = payable(receiver);
        selfdestruct(target);
    }

    function transferCfx(address a, uint amount) public payable {
        address payable target = payable(a);
        //maybe fail but not revert
        target.send(amount);
    }
}
