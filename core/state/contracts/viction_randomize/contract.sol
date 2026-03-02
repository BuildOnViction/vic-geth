pragma solidity ^0.4.21;

/*
 * This contract has been modified and amalgamated for reference purpose only.
 * The orginal contract and compile instruction can be found here:
 * https://github.com/BuildOnViction/viction-contracts
 */

// solc --allow-paths ., --abi --bin --overwrite --optimize -o core/state/contracts/build core/state/contracts/viction_randomize.sol
// abigen --abi core/state/contracts/build/VictionRandomize.abi --bin core/state/contracts/build/VictionRandomize.bin --pkg viction_randomize -out core/state/contracts/viction_randomize/bindings.go --v2

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

contract VictionRandomize {
    using SafeMath for uint256;

    mapping (address=>bytes32[]) randomSecret;
    mapping (address=>bytes32) randomOpening;

    function TomoRandomize () public {
    }

    function setSecret(bytes32[] _secret) public {
        uint secretPoint =  block.number % 900;
        require(secretPoint >= 800);
        require(secretPoint < 850);
        randomSecret[msg.sender] = _secret;
    }

    function setOpening(bytes32 _opening) public {
        uint openingPoint =  block.number % 900;
        require(openingPoint >= 850);
        randomOpening[msg.sender] = _opening;
    }

    function getSecret(address _validator) public view returns(bytes32[]) {
        return randomSecret[_validator];
    }

    function getOpening(address _validator) public view returns(bytes32) {
        return randomOpening[_validator];
    }
}
