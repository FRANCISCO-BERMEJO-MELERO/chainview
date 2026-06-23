# Guion de grabación del GIF de demostración de chainview, para vhs.
#
# Requisitos: vhs (https://github.com/charmbracelet/vhs), ttyd y ffmpeg.
# Uso, desde la raíz del repo:
#
#   make build
#   vhs .github/assets/demo.tape
#
# Graba con datos REALES (vitalik.eth) en un entorno XDG temporal y aislado, así
# que no toca tu configuración ni tus wallets. Se limita a las 4 redes
# principales (ETH/ARB/Base/OP) para que las cargas de red sean rápidas.

Output ".github/assets/demo.gif"

Set Shell "bash"
Set FontSize 15
Set Width 1320
Set Height 780
Set Padding 18
Set Theme "Catppuccin Mocha"
Set Framerate 24

# --- Preparación (oculta): entorno XDG aislado y limpieza de pantalla ---
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

# --- Portada de bienvenida ---
Type "./bin/chainview"
Enter
Sleep 2.5s

# entrar y saltar el tour
Enter
Sleep 1.2s
Escape
Sleep 800ms

# --- Cuentas: añadir una wallet por ENS ---
Type@120ms "vitalik.eth"
Sleep 900ms
Enter
Sleep 4.5s

# --- Balances: tabla multi-red + valor en fiat + total ---
Tab
Sleep 13s

# --- Transacciones: historial decodificado con badges ---
Tab
Sleep 12s

# --- Paleta de comandos (ctrl+k) ---
Ctrl+K
Sleep 1s
Type@120ms "balances"
Sleep 1.5s
Escape
Sleep 600ms

# --- Ayuda ---
Type "?"
Sleep 2s
Escape
Sleep 700ms

# salir
Type "q"
Sleep 1s
