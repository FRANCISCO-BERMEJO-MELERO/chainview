# vhs script that records chainview's demo GIF.
#
# Requirements: vhs (https://github.com/charmbracelet/vhs), ttyd and ffmpeg.
# Usage, from the repo root:
#
#   make build
#   vhs .github/assets/demo.tape
#
# Records with REAL data (vitalik.eth) in an isolated, temporary XDG environment,
# so it never touches your config or wallets. Limited to the 4 main networks
# (ETH/ARB/Base/OP) to keep network loads fast.

Output ".github/assets/demo.gif"

Set Shell "bash"
Set FontSize 15
Set Width 1320
Set Height 780
Set Padding 18
Set Theme "Catppuccin Mocha"
Set Framerate 24

# --- Setup (hidden): isolated XDG environment and clear screen ---
Hide
Type "D=$(mktemp -d)"
Enter
Type "mkdir -p $D/cfg/chainview $D/data/chainview"
Enter
Type `printf 'theme = "dark"\nfiat_currency = "usd"\nrefresh_seconds = 120\n' > "$D/cfg/chainview/config.toml"`
Enter
Type `printf '%s' '{"enabled_chains":[1,42161,8453,10]}' > "$D/data/chainview/prefs.json"`
Enter
Type "export XDG_CONFIG_HOME=$D/cfg XDG_DATA_HOME=$D/data"
Enter
Type "clear"
Enter
Show

# --- Welcome screen ---
Type "./bin/chainview"
Enter
Sleep 2.5s

# enter and skip the tour
Enter
Sleep 1.2s
Escape
Sleep 800ms

# --- Accounts: add a wallet by ENS ---
Type@120ms "vitalik.eth"
Sleep 900ms
Enter
Sleep 4.5s

# --- Balances: multi-network table + fiat value + total ---
Tab
Sleep 17s

# --- Transactions: decoded history with badges ---
Tab
Sleep 12s

# --- Command palette (ctrl+k) ---
Ctrl+K
Sleep 1s
Type@120ms "balances"
Sleep 1.5s
Escape
Sleep 600ms

# --- Help ---
Type "?"
Sleep 2s
Escape
Sleep 700ms

# quit
Type "q"
Sleep 1s
