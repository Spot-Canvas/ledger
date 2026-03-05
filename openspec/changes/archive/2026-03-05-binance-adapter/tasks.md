## 1. Fix leverage guard in `BinanceFuturesExchange.OpenPosition`

- [x] 1.1 In `internal/engine/exchange.go`, replace the `if req.Leverage > 0` guard with an unconditional `setLeverage` call that defaults to `1` when `req.Leverage <= 0`
- [x] 1.2 Add a `mockBinanceFuturesClient` struct in `exchange_test.go` (new file, `package engine`) that records calls to `setLeverage`, `newOrder`, `getBalance`, and `getPositionQty`
- [x] 1.3 Write `TestOpenPosition_SpotSignalSetsLeverageOne`: call `OpenPosition` with `Leverage=0`, assert `mockClient.leverageSet == 1`
- [x] 1.4 Write `TestOpenPosition_FuturesSignalSetsConfiguredLeverage`: call `OpenPosition` with `Leverage=5`, assert `mockClient.leverageSet == 5`

## 2. Fix fill quantity (`executedQty`)

- [x] 2.1 In `internal/engine/exchange.go`, rename the `newOrder` response struct field from `OrigQty` to `ExecutedQty` and update the JSON tag to `"executedQty"`
- [x] 2.2 Update the `Sscanf` call that reads quantity to use `body.ExecutedQty`
- [x] 2.3 Write `TestOpenPosition_FillQuantityFromExecutedQty`: mock returns `executedQty="0.00123"` and `origQty="0.00124"`, assert `OrderResult.Quantity == 0.00123`

## 3. Verify and run tests

- [x] 3.1 Run `go test ./internal/engine/...` — all existing tests plus new tests must pass
- [x] 3.2 Run `go build ./...` — no compile errors
