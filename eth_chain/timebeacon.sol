// SPDX-License-Identifier: GPL-3.0

pragma solidity >=0.8.2 <0.9.0;

/**
 * @title timeBeacon
 * @dev Store & retrieve ContractTB variable
 * @custom:dev-run-script ./scripts/deploy_with_ethers.ts
 */


struct ContractTB {
    uint32 shardID;
    uint64 height;
    string blockHash;
    string txHash;
    string statusHash;
}

struct ContractSignedTB {
    ContractTB tb;
    bytes[][] sigs;
    address[] signers;
}

contract TBStorage {
    uint32 minSigCnt; // the minimum number of signatures required for multi-signature
    mapping(uint32 => mapping(uint64 => ContractTB)) public tbs;

    event LogMessage(string message, uint32 shardID, uint64 height);

    // store ContractTBs of the genesis block
    constructor(ContractTB[] memory genesisTBs, uint32 _minSigCnt) {
        for (uint32 shardID = 0; shardID < genesisTBs.length; shardID++) {
            tbs[shardID][0] = genesisTBs[shardID];
        }
        minSigCnt = _minSigCnt;
    }

    // store ContractTB of specific shard and height
    function addTB(ContractTB memory tb, bytes[] memory sigs, address[] memory signers) public {
        // verify every signature until validSigCnt reaches minSigCnt
        uint32 validSigCnt = 0;
        for (uint32 i = 0; i < sigs.length; i++) {
            if (verifySignature(tb, sigs[i], signers[i])) {
                validSigCnt++;
                if (validSigCnt >= minSigCnt) {
                    break;
                }
            }
        }

        if (validSigCnt < minSigCnt) {
            emit LogMessage("insufficient valid signatures", tb.shardID, tb.height);
        } else {
            emit LogMessage("addTB", tb.shardID, tb.height);
            tbs[tb.shardID][tb.height] = tb;
        }

        // require(validSigCnt >= minSigCnt, "Insufficient valid signatures");
    }

    // Verify signature of ContractTB
    function verifySignature(ContractTB memory tb, bytes memory sig, address signer) internal pure returns (bool) {
        // abi-encode and Keccak-256 hash
        bytes memory encoded = abi.encode(tb.shardID, tb.height, tb.blockHash, tb.txHash, tb.statusHash);
        bytes32 msgHash = keccak256(encoded);

        // Extract r, s, and v values from the signature
        bytes32 r;
        bytes32 s;
        uint8 v;

        assembly {
            r := mload(add(sig, 32))
            s := mload(add(sig, 64))
            v := byte(0, mload(add(sig, 96)))
        }
        if (v < 27) {
            v = v + 27;
        }

        // Recover the public key from the signature
        address recoveredSigner = ecrecover(msgHash, v, r, s);

        return recoveredSigner == signer;
    }

    // retrieve ContractTB of specific shard and height
    function getTB(uint32 shardID, uint64 height) public returns (ContractTB memory tb){
        emit LogMessage("getTB", shardID, height);
        return tbs[shardID][height];
    }
    
}
