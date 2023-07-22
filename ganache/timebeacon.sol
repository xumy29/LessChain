// SPDX-License-Identifier: GPL-3.0

pragma solidity >=0.8.2 <0.9.0;

/**
 * @title Storage
 * @dev Store & retrieve value in a variable
 * @custom:dev-run-script ./scripts/deploy_with_ethers.ts
 */
struct TB {
    uint32 shardID;
    uint64 height;
    string blockHash;
    string txHash;
    string statusHash;
}

contract TBStorage {
    mapping(uint32 => mapping(uint64 => TB)) public tbs;

    event LogMessage(string message, uint32 shardID, uint64 height);

    // 写入创世区块的信标
    constructor(TB[] memory genesisTBs) {
        for (uint32 shardID = 0; shardID < genesisTBs.length; shardID++) {
            tbs[shardID][0] = genesisTBs[shardID];
        }
    }

    /**
     * @dev Store TimeBeacon
     * @param tb TimeBeacon to store
     */
    function addTB(TB calldata tb) public {
        tbs[tb.shardID][tb.height] = tb;
        emit LogMessage("addTB", tb.shardID, tb.height);
    }

    /**
     * @dev Return timebeacon of specific height in specific shard 
     */
    function getTB(uint32 shardID, uint64 height) public returns (TB memory tb){
        emit LogMessage("getTB", shardID, height);
        return tbs[shardID][height];
    }
}
