# chainview

Monitor de wallets EVM en el terminal (TUI), watch-only. Sigue balances,
historial de transacciones y gas en tiempo real sobre Ethereum, Arbitrum, Base y
Optimism — **sin registrarte en ningún sitio ni configurar nada**.

## Características

- **Balances multi-cuenta y multi-red** en una tabla, con refresco automático.
- **Historial de transacciones** con la acción decodificada (`Transfer 100 USDC
  → 0x…`) y un modal de detalle por transacción.
- **Nombres ENS**: muestra `vitalik.eth` en vez de la address, y puedes añadir
  cuentas tanto por address como por nombre ENS.
- **Gas tracker** en vivo con tendencia por red.
- **Keyless**: el historial usa Blockscout y los balances RPC públicos; cero
  API keys para empezar.

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

### Atajos

| Tecla            | Acción                                   |
| ---------------- | ---------------------------------------- |
| `tab` / `↹⇧`     | Cambiar de pestaña                       |
| `?`              | Mostrar/ocultar la ayuda                 |
| `↑` `↓`          | Moverse por listas y tablas              |
| `enter`          | Añadir wallet / ver detalle de tx        |
| `ctrl+d`         | Borrar la wallet seleccionada            |
| `r`              | Recargar balances / transacciones        |
| `esc`            | Cerrar modal de detalle / ayuda          |
| `q` / `ctrl+c`   | Salir                                     |

## Configuración

chainview funciona sin configuración. Para personalizar, copia el ejemplo:

```sh
cp config.example.toml ~/.config/chainview/config.toml
```

Ver [`config.example.toml`](./config.example.toml) para todas las opciones:
intervalo de refresco, overrides de RPC por red, y la API key opcional de
Etherscan (`ETHERSCAN_API_KEY` como variable de entorno tiene prioridad).

## Licencia

MIT — ver [LICENSE](./LICENSE).
