// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/// @title Chainlink AggregatorV3 Interface (minimal)
interface AggregatorV3Interface {
    function latestRoundData()
        external
        view
        returns (
            uint80 roundId,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        );

    function decimals() external view returns (uint8);
}

/// @title ZhajinhuaChips
/// @notice Game chip purchase contract for Zhajinhua. Users send ETH and receive
///         game points at a rate of 1000 points per 1 USD worth of ETH.
///         The ETH/USD price is sourced from a Chainlink-compatible oracle with
///         an on-chain fallback price if the oracle is stale or unavailable.
contract ZhajinhuaChips {
    // -----------------------------------------------------------------------
    // Errors
    // -----------------------------------------------------------------------

    error NotOwner();
    error ZeroAddress();
    error ZeroValue();
    error WithdrawFailed();
    error NewOwnerIsZero();

    // -----------------------------------------------------------------------
    // Events
    // -----------------------------------------------------------------------

    /// @notice Emitted when a user purchases chips.
    /// @param buyer       Address of the purchaser.
    /// @param ethAmount   Wei sent by the buyer.
    /// @param pointsAwarded Number of game points credited.
    /// @param ethPriceUsd ETH/USD price (8-decimal fixed point) used for the conversion.
    event ChipsPurchased(
        address indexed buyer,
        uint256 ethAmount,
        uint256 pointsAwarded,
        uint256 ethPriceUsd
    );

    /// @notice Emitted when the owner withdraws accumulated funds.
    /// @param to     Recipient address.
    /// @param amount Wei withdrawn.
    event FundsWithdrawn(address indexed to, uint256 amount);

    /// @notice Emitted when ownership is transferred.
    /// @param previousOwner Old owner address.
    /// @param newOwner      New owner address.
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    /// @notice Emitted when the fallback price is updated.
    /// @param oldPrice Previous fallback price (8-decimal fixed point).
    /// @param newPrice New fallback price (8-decimal fixed point).
    event FallbackPriceUpdated(uint256 oldPrice, uint256 newPrice);

    /// @notice Emitted when the price feed address is updated.
    /// @param oldFeed Previous feed address.
    /// @param newFeed New feed address.
    event PriceFeedUpdated(address oldFeed, address newFeed);

    /// @notice Emitted when the staleness threshold is updated.
    /// @param oldThreshold Previous threshold in seconds.
    /// @param newThreshold New threshold in seconds.
    event StalenessThresholdUpdated(uint256 oldThreshold, uint256 newThreshold);

    // -----------------------------------------------------------------------
    // Constants
    // -----------------------------------------------------------------------

    /// @notice Points awarded per 1 USD of ETH.
    uint256 public constant POINTS_PER_USD = 1000;

    /// @notice All prices are normalised to 8 decimals (Chainlink standard).
    uint8 public constant PRICE_DECIMALS = 8;

    // -----------------------------------------------------------------------
    // State
    // -----------------------------------------------------------------------

    /// @notice Contract owner (admin).
    address public owner;

    /// @notice Chainlink-compatible ETH/USD price feed.
    AggregatorV3Interface public priceFeed;

    /// @notice Fallback ETH/USD price used when the oracle is stale or reverts.
    ///         Stored with 8 decimals (e.g. 2000_00000000 = $2 000).
    uint256 public fallbackPriceUsd;

    /// @notice Maximum age (seconds) of an oracle answer before it is considered stale.
    uint256 public stalenessThreshold;

    /// @notice Cumulative ETH deposited per user (in wei).
    mapping(address => uint256) public totalDeposited;

    /// @notice Cumulative game points earned per user.
    mapping(address => uint256) public totalPoints;

    // -----------------------------------------------------------------------
    // Modifiers
    // -----------------------------------------------------------------------

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    // -----------------------------------------------------------------------
    // Constructor
    // -----------------------------------------------------------------------

    /// @param _priceFeed          Address of the Chainlink ETH/USD price feed.
    /// @param _fallbackPriceUsd   Fallback price with 8 decimals.
    /// @param _stalenessThreshold Seconds after which an oracle answer is stale.
    constructor(
        address _priceFeed,
        uint256 _fallbackPriceUsd,
        uint256 _stalenessThreshold
    ) {
        if (_priceFeed == address(0)) revert ZeroAddress();
        if (_fallbackPriceUsd == 0) revert ZeroValue();

        owner = msg.sender;
        priceFeed = AggregatorV3Interface(_priceFeed);
        fallbackPriceUsd = _fallbackPriceUsd;
        stalenessThreshold = _stalenessThreshold;

        emit OwnershipTransferred(address(0), msg.sender);
    }

    // -----------------------------------------------------------------------
    // Public / External — chip purchase
    // -----------------------------------------------------------------------

    /// @notice Purchase game chips by sending ETH.
    /// @return points The number of game points credited.
    function buyChips() external payable returns (uint256 points) {
        if (msg.value == 0) revert ZeroValue();

        uint256 ethPriceUsd = _getEthPriceUsd();

        // points = (ethAmount * ethPriceUsd * POINTS_PER_USD) / (1e18 * 10^PRICE_DECIMALS)
        //
        // ethAmount is in wei (18 decimals), ethPriceUsd has PRICE_DECIMALS decimals.
        // The division strips both scaling factors, leaving a plain integer point count.
        points = (msg.value * ethPriceUsd * POINTS_PER_USD) / (1e18 * 10 ** PRICE_DECIMALS);

        totalDeposited[msg.sender] += msg.value;
        totalPoints[msg.sender] += points;

        emit ChipsPurchased(msg.sender, msg.value, points, ethPriceUsd);
    }

    // -----------------------------------------------------------------------
    // View helpers
    // -----------------------------------------------------------------------

    /// @notice Preview how many points a given ETH amount would yield.
    /// @param ethAmount Wei amount to quote.
    /// @return points   Estimated game points.
    function quotePoints(uint256 ethAmount) external view returns (uint256 points) {
        uint256 ethPriceUsd = _getEthPriceUsd();
        points = (ethAmount * ethPriceUsd * POINTS_PER_USD) / (1e18 * 10 ** PRICE_DECIMALS);
    }

    /// @notice Return the ETH/USD price currently in use (oracle or fallback).
    /// @return price 8-decimal fixed-point ETH/USD price.
    function currentPrice() external view returns (uint256 price) {
        price = _getEthPriceUsd();
    }

    // -----------------------------------------------------------------------
    // Owner-only — administration
    // -----------------------------------------------------------------------

    /// @notice Withdraw all accumulated ETH to a specified address.
    /// @param to Recipient of the funds.
    function withdraw(address payable to) external onlyOwner {
        if (to == address(0)) revert ZeroAddress();

        uint256 balance = address(this).balance;
        if (balance == 0) revert ZeroValue();

        (bool success, ) = to.call{value: balance}("");
        if (!success) revert WithdrawFailed();

        emit FundsWithdrawn(to, balance);
    }

    /// @notice Withdraw a specific amount of ETH.
    /// @param to     Recipient of the funds.
    /// @param amount Wei to withdraw.
    function withdrawAmount(address payable to, uint256 amount) external onlyOwner {
        if (to == address(0)) revert ZeroAddress();
        if (amount == 0 || amount > address(this).balance) revert ZeroValue();

        (bool success, ) = to.call{value: amount}("");
        if (!success) revert WithdrawFailed();

        emit FundsWithdrawn(to, amount);
    }

    /// @notice Update the Chainlink-compatible price feed address.
    /// @param newFeed New price feed address.
    function setPriceFeed(address newFeed) external onlyOwner {
        if (newFeed == address(0)) revert ZeroAddress();

        address oldFeed = address(priceFeed);
        priceFeed = AggregatorV3Interface(newFeed);

        emit PriceFeedUpdated(oldFeed, newFeed);
    }

    /// @notice Update the fallback ETH/USD price.
    /// @param newPrice New fallback price (8-decimal fixed point).
    function setFallbackPrice(uint256 newPrice) external onlyOwner {
        if (newPrice == 0) revert ZeroValue();

        uint256 oldPrice = fallbackPriceUsd;
        fallbackPriceUsd = newPrice;

        emit FallbackPriceUpdated(oldPrice, newPrice);
    }

    /// @notice Update the staleness threshold for the oracle.
    /// @param newThreshold New threshold in seconds.
    function setStalenessThreshold(uint256 newThreshold) external onlyOwner {
        uint256 oldThreshold = stalenessThreshold;
        stalenessThreshold = newThreshold;

        emit StalenessThresholdUpdated(oldThreshold, newThreshold);
    }

    /// @notice Transfer ownership to a new address.
    /// @param newOwner Address of the new owner.
    function transferOwnership(address newOwner) external onlyOwner {
        if (newOwner == address(0)) revert NewOwnerIsZero();

        address previousOwner = owner;
        owner = newOwner;

        emit OwnershipTransferred(previousOwner, newOwner);
    }

    // -----------------------------------------------------------------------
    // Receive / Fallback — allow plain ETH transfers to buy chips
    // -----------------------------------------------------------------------

    /// @notice Buying chips by sending ETH directly to the contract.
    receive() external payable {
        if (msg.value == 0) revert ZeroValue();

        uint256 ethPriceUsd = _getEthPriceUsd();
        uint256 points = (msg.value * ethPriceUsd * POINTS_PER_USD) / (1e18 * 10 ** PRICE_DECIMALS);

        totalDeposited[msg.sender] += msg.value;
        totalPoints[msg.sender] += points;

        emit ChipsPurchased(msg.sender, msg.value, points, ethPriceUsd);
    }

    // -----------------------------------------------------------------------
    // Internal
    // -----------------------------------------------------------------------

    /// @dev Fetch the ETH/USD price from the oracle. Falls back to
    ///      `fallbackPriceUsd` if the call reverts or the answer is stale/invalid.
    function _getEthPriceUsd() internal view returns (uint256) {
        try priceFeed.latestRoundData() returns (
            uint80,
            int256 answer,
            uint256,
            uint256 updatedAt,
            uint80
        ) {
            bool isStale = stalenessThreshold > 0 &&
                (block.timestamp - updatedAt) > stalenessThreshold;

            if (answer > 0 && !isStale) {
                return uint256(answer);
            }
        } catch {
            // Oracle call reverted — fall through to fallback.
        }

        return fallbackPriceUsd;
    }
}
