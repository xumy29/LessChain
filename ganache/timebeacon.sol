// SPDX-License-Identifier: GPL-3.0

pragma solidity >=0.8.2 <0.9.0;

/**
 * @title Storage
 * @dev Store & retrieve value in a variable
 * @custom:dev-run-script ./scripts/deploy_with_ethers.ts
 */
contract TBStorage {

    uint64 height;
    uint16 shardID;
    bytes32 blockHash;
    bytes32 txHash;
    bytes32 statusHash;
    uint32 number;

    /**
     * @dev Store value in variable
     * @param num value to store
     */
    function addTB(uint32 num) public {
        number = num;
    }

    /**
     * @dev Return value 
     * @return value of 'number'
     */
    function getTB() public view returns (uint32){
        return number;
    }
}