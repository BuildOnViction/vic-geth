pragma solidity ^0.4.24;

/*
 * This contract has been modified and amalgamated for reference purpose only.
 * The orginal contract and compile instruction can be found here:
 * https://github.com/BuildOnViction/viction-contracts
 */

// solc --allow-paths ., --abi --bin --overwrite --optimize -o core/state/contracts/build core/state/contracts/viction_zero_gas.sol
// abigen --abi core/state/contracts/build/VictionZeroGas.abi --bin core/state/contracts/build/VictionZeroGas.bin --pkg viction_zero_gas -out core/state/contracts/viction_zero_gas/bindings.go --v2

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
    require(c / a == b);
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
    require(b <= a);
    return a - b;
  }

  /**
  * @dev Adds two numbers, throws on overflow.
  */
  function add(uint256 a, uint256 b) internal pure returns (uint256) {
    uint256 c = a + b;
    require(c >= a);
    return c;
  }
}

contract AbstractZeroGasToken {
    function issuer() public view returns (address);
}

contract VictionZeroGas {
    using SafeMath for uint256;
    uint256 _minCap;
    address[] _tokens;
    mapping(address => uint256) tokensState;

    event Apply(address indexed issuer, address indexed token, uint256 value);
    event Charge(address indexed supporter, address indexed token, uint256 value);

    constructor (uint256 value) public {
        _minCap = value;
    }

    function minCap() public view returns(uint256) {
        return _minCap;
    }

    function tokens() public view returns(address[]) {
        return _tokens;
    }

    function getTokenCapacity(address token) public view returns(uint256) {
        return tokensState[token];
    }

    modifier onlyValidCapacity(address token) {
        require(token != address(0));
        require(msg.value >= _minCap);
        _;
    }

    function apply(address token) public payable onlyValidCapacity(token) {
        AbstractZeroGasToken t = AbstractZeroGasToken(token);
        require(t.issuer() == msg.sender);
        _tokens.push(token);
        tokensState[token] = tokensState[token].add(msg.value);
        emit Apply(msg.sender, token, msg.value);
    }

    function charge(address token) public payable onlyValidCapacity(token) {
        tokensState[token] = tokensState[token].add(msg.value);
        emit Charge(msg.sender, token, msg.value);
    }
}
