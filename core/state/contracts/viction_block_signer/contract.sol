pragma solidity ^0.4.21;

/*
 * This contract has been modified and amalgamated for reference purpose only.
 * The orginal contract and compile instruction can be found here:
 * https://github.com/BuildOnViction/viction-contracts
 */

// solc --allow-paths ., --abi --bin --overwrite --optimize -o core/state/contracts/build core/state/contracts/viction_block_signer.sol
// abigen --abi core/state/contracts/build/VictionBlockSigner.abi --bin core/state/contracts/build/VictionBlockSigner.bin -pkg viction_block_signer -out core/state/contracts/viction_block_signer/bindings.go --v2

/**
 * @title SafeMath
 * @dev Math operations with safety checks that throw on error
 */
library SafeMath {

  /**
  * @dev Multiplies two numbers, throws on overflow.
  */
  function mul(uint256 a, uint256 b) internal pure returns (uint256) {
    if (a == 0) {
      return 0;
    }
    uint256 c = a * b;
    assert(c / a == b);
    return c;
  }

  /**
  * @dev Integer division of two numbers, truncating the quotient.
  */
  function div(uint256 a, uint256 b) internal pure returns (uint256) {
    // assert(b > 0); // Solidity automatically throws when dividing by 0
    // uint256 c = a / b;
    // assert(a == b * c + a % b); // There is no case in which this doesn't hold
    return a / b;
  }

  /**
  * @dev Subtracts two numbers, throws on overflow (i.e. if subtrahend is greater than minuend).
  */
  function sub(uint256 a, uint256 b) internal pure returns (uint256) {
    assert(b <= a);
    return a - b;
  }

  /**
  * @dev Adds two numbers, throws on overflow.
  */
  function add(uint256 a, uint256 b) internal pure returns (uint256) {
    uint256 c = a + b;
    assert(c >= a);
    return c;
  }
}


contract VictionBlockSigner {
    using SafeMath for uint256;

    event Sign(address _signer, uint256 _blockNumber, bytes32 _blockHash);

    mapping(bytes32 => address[]) blockSigners;
    mapping(uint256 => bytes32[]) blocks;
    uint256 public epochNumber;

    function BlockSigner(uint256 _epochNumber) public {
        epochNumber = _epochNumber;
    }

    function sign(uint256 _blockNumber, bytes32 _blockHash) external {
        // consensus should validate all senders are validators, gas = 0
        require(block.number >= _blockNumber);
        require(block.number <= _blockNumber.add(epochNumber * 2));
        blocks[_blockNumber].push(_blockHash);
        blockSigners[_blockHash].push(msg.sender);

        emit Sign(msg.sender, _blockNumber, _blockHash);
    }

    function getSigners(bytes32 _blockHash) public view returns(address[]) {
        return blockSigners[_blockHash];
    }
}
