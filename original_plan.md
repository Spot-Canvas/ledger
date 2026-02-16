I have a trading bot running external to this repository. It's trading crypto using daily/swing trade strategies. I need a ledger for this bot which will provide me with the following:

- portfolio tracking
- order history
- tax reporting

The bot is now able to produce a dashboard page like so: https://spot-arb-dashboard.s3.eu-north-1.amazonaws.com/paper_trading.html
I want to be able to produce that kind of page using the data this ledger module will provide. The bot will no longer keep track of the positions and trades.

## Some additional requirements:

- Ingest the trades using a NATS subscription. The bot will publish the trades to NATS.
- Support multiple trading accounts (like for live and paper trading)
- Support spot and futures trading
- Leveraged futures
- A REST endpoint for querying the porfolio, open trades, order history
- Ability to produce a report for the Finnish tax authorities

## Priorities:

- The tax reporting will be done in a later phase. We just need to be prepared for that so that we have all the data for it.

## Non goals:

- No UI or HTML views. This module will just provide data and the UI/dashboard will be generated elsewhere.

Ask if you need more details!
