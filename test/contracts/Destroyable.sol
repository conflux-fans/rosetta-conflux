//SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;

contract SelfDestroyable {
    uint public val;
    
    function setVal(uint _val) public {
        val = _val;
    }

    function alwaysFail() public {
        val = 3;
        revert();
    }
    
    function destroy(address receiver) public {
        address payable target = payable(receiver);
        selfdestruct(target);
    }

    receive() external payable {}
}