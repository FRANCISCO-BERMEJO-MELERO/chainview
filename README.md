# chainview

Monitor de wallets EVM en el terminal (TUI), watch-only. Sigue balances,
tokens, valor en fiat, historial de transacciones y gas en tiempo real sobre
Ethereum, Arbitrum, Base, Optimism, Polygon, Scroll y Linea (y cualquier red EVM
que añadas) — **sin registrarte en ningún sitio ni configurar nada**.

## Características

- **Balances multi-cuenta y multi-red** en una tabla, con refresco automático.
- **Tokens ERC-20**: descubre y muestra los tokens de cada wallet, no solo el
  saldo nativo.
- **Valoración en fiat**: precio en USD por activo y **total de cartera**, vía
  DefiLlama (keyless).
- **Historial de transacciones** con la acción decodificada (`Transfer 100 USDC
  → 0x…`) y un modal de detalle por transacción.
- **Nombres ENS**: muestra `vitalik.eth` en vez de la address, y puedes añadir
  cuentas tanto por address como por nombre ENS.
- **Gas tracker** en vivo con tendencia por red.
- **Paleta de comandos** (`Ctrl+K`): busca wallets, redes y acciones con filtrado
  difuso, sin recordar atajos.
- **Temas** claro/oscuro con detección automática del fondo del terminal
  (configurable y alternable en caliente).
- **Atajos de productividad**: copiar address/hash (`y`), abrir en el explorador
  (`o`), ordenar columnas (`s`) y detalle agregado de wallet entre redes.
- **Redes configurables**: añade cualquier cadena EVM desde el TOML (chainID +
  RPC + explorador) sin recompilar.
- **Keyless**: precios (DefiLlama), tokens e historial (Blockscout) y balances
  (RPC públicos); cero API keys para empezar.

## Instalación

```sh
go install github.com/FRANCISCO-BERMEJO-MELERO/chainview/cmd/chainview@latest
```

O desde el repo:

```sh
make build   # binario en ./bin/chainview
make run
```

## Uso

Lanza `chainview`. En la pestaña **Cuentas**, escribe una address (`0x…`) o un
nombre ENS (`vitalik.eth`) y pulsa Enter. Cambia de pestaña con `tab`.

Para diagnosticar, arranca con `chainview --debug` (o `CHAINVIEW_DEBUG=1`) o pulsa
`ctrl+g` para ver métricas internas: llamadas RPC, aciertos de caché y
rate-limits.

### Atajos

| Tecla            | Acción                                          |
| ---------------- | ----------------------------------------------- |
| `tab` / `↹⇧`     | Cambiar de pestaña                              |
| `ctrl+k`         | Paleta de comandos (buscar/ir/acciones)         |
| `ctrl+g`         | Métricas / modo debug (RPCs, caché, rate-limits) |
| `?`              | Mostrar/ocultar la ayuda                        |
| `↑` `↓`          | Moverse por listas y tablas                     |
| `enter`          | Añadir wallet / detalle de wallet o de tx       |
| `ctrl+d`         | Borrar la wallet seleccionada (confirma con otro `ctrl+d`) |
| `y` / `o`        | Copiar address/hash / abrir en el explorador    |
| `s` / `S`        | Ordenar columna / invertir el orden             |
| `f`              | Filtrar (wallet en Balances, red en Txs)        |
| `n`              | Elegir redes visibles                           |
| `e`              | Exportar a CSV                                  |
| `r`              | Recargar balances / transacciones               |
| `esc`            | Cerrar modal / overlay                          |
| `q` / `ctrl+c`   | Salir                                           |

## Configuración

chainview funciona sin configuración. Para personalizar, copia el ejemplo:

```sh
cp config.example.toml ~/.config/chainview/config.toml
```

Ver [`config.example.toml`](./config.example.toml) para todas las opciones:
intervalo de refresco, moneda fiat, overrides de RPC por red, alta de redes
personalizadas (`[[network]]`), y la API key opcional de Etherscan
(`ETHERSCAN_API_KEY` como variable de entorno tiene prioridad).

## Licencia

MIT — ver [LICENSE](./LICENSE).
