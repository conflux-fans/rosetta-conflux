// SPDX-License-Identifier: MIT
pragma solidity >=0.4.22 <0.9.0;
import "@confluxfans/contracts/internalContracts/InternalContractsLib.sol";

contract TestTraceA {
    function callB(TestTraceB b) public {
        b.foo();
    }

    function mustRevert() public {
        revert();
    }

    function mustOk() public payable {}
}

contract TestTraceB {
    function fooPure() public pure {}

    function foo() public {}

    function destory() public {
        selfdestruct(payable(msg.sender));
    }
}

contract TestRosetta {
    event InteralCalled(bool success);

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

    function transferCfx(address a, uint256 amount) public payable {
        address payable target = payable(a);
        //maybe fail but not revert
        target.send(amount);
    }

    function mustInternalFail() public {
        TestTraceA a = new TestTraceA();
        (bool callResult, ) = address(a).call(
            abi.encodeWithSignature("mustRevert()")
        );
        (callResult, ) = address(a).call{value: 1 ether}(
            abi.encodeWithSignature("mustRevert()")
        );
        emit InteralCalled(callResult);
    }

    function mustInternalOk() public payable {
        require(msg.value >= 1 ether, "must pay at least 1 ether");
        TestTraceA a = new TestTraceA();
        (bool callResult, ) = address(a).call{value: 1 ether}(
            abi.encodeWithSignature("mustOk()")
        );
        emit InteralCalled(callResult);
    }
}
