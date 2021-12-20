//SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;

contract TransferUtil {
    
    receive() external payable {}

    function transfer(address a, uint amount) public payable {
        address payable target = payable(a);
        target.transfer(amount);
    }
    
}