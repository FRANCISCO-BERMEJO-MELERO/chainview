# Flujo de mensajes: UI ↔ chain

chainview sigue la **Elm Architecture** de Bubble Tea: el estado vive en el
`Model` y solo cambia dentro de `Update`, en respuesta a mensajes (`tea.Msg`).
Toda operación lenta (una llamada RPC) se ejecuta **fuera** de `Update`, dentro
de un `tea.Cmd` que corre en su propia goroutine y devuelve un `tea.Msg` cuando
termina.

La regla de oro: **nunca hacer I/O de red de forma síncrona dentro de `Update`**.
Si bloqueas `Update`, congelas toda la TUI (no repinta, no responde a teclas).

```
                        ┌──────────────────────────────────────────┐
                        │                  Model                    │
                        │   (estado: tabs, wallets, balances, ...)  │
                        └──────────────────────────────────────────┘
                              ▲                          │
                  (4) tea.Msg │                          │ (1) View()
            balanceLoadedMsg /│                          ▼
            balanceErrMsg     │                  ┌───────────────┐
                              │                  │   terminal    │
                        ┌─────┴───────┐          │  (lo que ve   │
                        │   Update    │◀─────────│   el usuario) │
                        │  (msg) →    │  KeyMsg  └───────────────┘
                        │ (Model,Cmd) │  Tick, WindowSizeMsg
                        └─────┬───────┘
                              │ (2) devuelve un tea.Cmd
                              ▼
                   ┌──────────────────────┐
                   │   tea.Cmd (goroutine)│   <- Bubble Tea lo ejecuta
                   │  func() tea.Msg {    │      en background, sin
                   │    ctx, cancel :=    │      bloquear Update/View
                   │      context.WithTimeout
                   │    bal, err :=       │
                   │      client.BalanceAt(ctx, chainID, addr)
                   │    return msg(bal,err)│
                   │  }                   │
                   └──────────┬───────────┘
                              │ (3) llamada RPC vía chain.Client
                              ▼
                   ┌──────────────────────┐
                   │     chain.Client     │  lazy connect + cache por chainID
                   │  ethclient.Dial(RPC) │  (sync.RWMutex, double-check)
                   └──────────┬───────────┘
                              │ JSON-RPC
                              ▼
                   ┌──────────────────────┐
                   │   RPC público (red)  │  ethereum-rpc.publicnode.com, etc.
                   └──────────────────────┘
```

## El ciclo, paso a paso

1. **View()** dibuja el `Model` actual en el terminal tras cada `Update`.
2. **Update(msg)** recibe un mensaje (una tecla, un `tea.WindowSizeMsg`, un
   `tea.Tick`...). Si necesita datos de la cadena, **no** los pide ahí: devuelve
   un `tea.Cmd` (una función `func() tea.Msg`).
3. Bubble Tea ejecuta ese `tea.Cmd` **en una goroutine aparte**. Dentro se hace
   la llamada RPC a través de `chain.Client`, siempre con `context.WithTimeout`
   para no quedarse colgado.
4. Cuando la llamada termina, el `tea.Cmd` devuelve un `tea.Msg` con el resultado
   (`balanceLoadedMsg`) o el error (`balanceErrMsg`). Bubble Tea lo encola y se
   lo entrega a `Update`, que actualiza el `Model`. En el siguiente `View()` se
   ve el balance.

## Por qué importa la concurrencia del Client

Al refrescar balances de varias wallets × 4 redes se lanzan **muchos `tea.Cmd`
a la vez**, cada uno en su goroutine. Todos comparten el mismo `chain.Client`,
así que `connect()` debe ser thread-safe: el `sync.RWMutex` + double-check
garantiza que dos goroutines pidiendo la misma red **no abran dos conexiones**.
El refresco periódico (D2: polling con `tea.Tick`) reinyecta este mismo flujo
cada ~15 s.
