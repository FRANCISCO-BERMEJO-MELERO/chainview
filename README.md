# chainview

[![CI](https://github.com/FRANCISCO-BERMEJO-MELERO/chainview/actions/workflows/ci.yml/badge.svg)](https://github.com/FRANCISCO-BERMEJO-MELERO/chainview/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/FRANCISCO-BERMEJO-MELERO/chainview/branch/main/graph/badge.svg)](https://codecov.io/gh/FRANCISCO-BERMEJO-MELERO/chainview)
[![Go Report Card](https://goreportcard.com/badge/github.com/FRANCISCO-BERMEJO-MELERO/chainview)](https://goreportcard.com/report/github.com/FRANCISCO-BERMEJO-MELERO/chainview)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

A watch-only EVM wallet monitor for your terminal (TUI). Tracks balances,
tokens, fiat value, transaction history and gas in real time across Ethereum,
Arbitrum, Base, Optimism, Polygon, Scroll and Linea (and any EVM network you
add) — **with no sign-up and zero configuration**.

<p align="center">
  <img src=".github/assets/demo.gif" alt="chainview demo: multi-network balances with fiat value, transaction history and the command palette" width="100%">
</p>

## Features

- **Multi-account, multi-network balances** in one table, auto-refreshed.
- **ERC-20 tokens**: discovers and shows each wallet's tokens, not just the
  native balance.
- **Fiat valuation**: USD price per asset and **total portfolio value**, via
  DefiLlama (keyless).
- **Transaction history** with the decoded action (`Transfer 100 USDC → 0x…`)
  and a per-transaction detail modal.
- **ENS names**: shows `vitalik.eth` instead of the address, and you can add
  accounts by address or ENS name.
- **Live gas tracker** with per-network trend.
- **Command palette** (`Ctrl+K`): fuzzy-search wallets, networks and actions
  without memorizing shortcuts.
- **Themes** light/dark with automatic terminal-background detection
  (configurable and hot-swappable).
- **Productivity shortcuts**: copy address/hash (`y`), open in the explorer
  (`o`), sort columns (`s`) and an aggregated per-wallet detail across networks.
- **Configurable networks**: add any EVM chain from the TOML (chainID + RPC +
  explorer) without recompiling.
- **Keyless**: prices (DefiLlama), tokens and history (Blockscout) and balances
  (public RPCs) — zero API keys to get started.

## Installation

```sh
go install github.com/FRANCISCO-BERMEJO-MELERO/chainview/cmd/chainview@latest
```

Or from the repo:

```sh
make build   # binary at ./bin/chainview
make run
```

## Usage

Launch `chainview`. In the **Accounts** tab, type an address (`0x…`) or an ENS
name (`vitalik.eth`) and press Enter. Switch tabs with `tab`.

`chainview --version` prints the build version and `chainview --help` lists the
options. To diagnose, start with `chainview --debug` (or `CHAINVIEW_DEBUG=1`) or
press `ctrl+g` to see internal metrics: RPC calls, cache hits and rate-limits.

### Shortcuts

| Key              | Action                                          |
| ---------------- | ----------------------------------------------- |
| `tab` / `↹⇧`     | Switch tab                                      |
| `ctrl+k`         | Command palette (search/go/actions)             |
| `ctrl+g`         | Metrics / debug mode (RPCs, cache, rate-limits) |
| `?`              | Toggle help                                     |
| `↑` `↓`          | Move through lists and tables                   |
| `enter`          | Add wallet / wallet or tx detail                |
| `ctrl+d`         | Delete the selected wallet (confirm with another `ctrl+d`) |
| `y` / `o`        | Copy address/hash / open in the explorer        |
| `s` / `S`        | Sort column / reverse order                     |
| `f`              | Filter (wallet in Balances, network in Txs)     |
| `n`              | Choose visible networks                         |
| `e`              | Export to CSV                                   |
| `r`              | Reload balances / transactions                  |
| `esc`            | Close modal / overlay                           |
| `q` / `ctrl+c`   | Quit                                            |

## Configuration

chainview works with no configuration. To customize, copy the example:

```sh
cp config.example.toml ~/.config/chainview/config.toml
```

See [`config.example.toml`](./config.example.toml) for all options: refresh
interval, fiat currency, per-network RPC overrides, custom-network definitions
(`[[network]]`), and the optional Etherscan API key (the `ETHERSCAN_API_KEY`
environment variable takes precedence).

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](./CONTRIBUTING.md).

## License

MIT — see [LICENSE](./LICENSE).
