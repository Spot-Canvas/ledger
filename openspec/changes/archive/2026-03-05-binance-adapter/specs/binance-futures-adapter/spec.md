## ADDED Requirements

### Requirement: Leverage always set before open
Before placing any MARKET open order, `BinanceFuturesExchange` SHALL call `setLeverage` on the target symbol. When `req.Leverage` is 0 or negative, it SHALL use 1 as the effective leverage. This ensures spot-like signals (which carry no explicit leverage) always open at 1× regardless of any leverage previously configured on the account.

#### Scenario: Spot-like signal opens at 1× leverage
- **WHEN** `OpenPosition` is called with `Leverage = 0`
- **THEN** `setLeverage` SHALL be called with leverage value `1` before the order is placed

#### Scenario: Futures signal opens at configured leverage
- **WHEN** `OpenPosition` is called with `Leverage = 5`
- **THEN** `setLeverage` SHALL be called with leverage value `5` before the order is placed

#### Scenario: Leverage set even when unchanged
- **WHEN** `OpenPosition` is called and the Binance account already has the correct leverage set
- **THEN** `setLeverage` SHALL still be called (Binance returns success immediately; no separate pre-check is made)

---

### Requirement: Fill quantity from executedQty
`BinanceFuturesExchange` SHALL record the fill quantity from the `executedQty` field of the Binance order response, not `origQty`. `executedQty` reflects the actual filled amount after lot-size rounding; `origQty` reflects the originally requested amount and may differ.

#### Scenario: Fill quantity matches exchange fill
- **WHEN** a MARKET order is placed and Binance returns `executedQty = "0.001230"` and `origQty = "0.001234"`
- **THEN** `OrderResult.Quantity` SHALL be `0.001230`

---

### Requirement: Fee recorded as zero
`BinanceFuturesExchange` SHALL set `OrderResult.Fee` to `0`. The Binance Futures order endpoint does not include commission in its response; exact commission requires a separate `GET /fapi/v1/userTrades` call which is not performed at order time.

#### Scenario: Open order fee is zero
- **WHEN** `OpenPosition` completes successfully
- **THEN** `OrderResult.Fee` SHALL be `0.0`

#### Scenario: Close order fee is zero
- **WHEN** `ClosePosition` completes successfully
- **THEN** `OrderResult.Fee` SHALL be `0.0`
