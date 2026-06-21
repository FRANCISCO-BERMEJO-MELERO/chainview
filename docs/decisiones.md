# Registro de decisiones técnicas

Decisiones tomadas durante el desarrollo de chainview. Formato ligero: contexto,
decisión y motivo. Las decisiones cerradas no se re-discuten salvo que cambie el
contexto.

## D1 — Provider RPC por defecto (Semana 2)

**Contexto:** hay que leer estado on-chain de 4 redes sin que el usuario tenga
que configurar nada para empezar. Opciones: publicnode.com (gratis, sin key,
rate-limited), Alchemy o Infura (más cuota, requieren registro + API key).

**Decisión:** **publicnode.com** como provider por defecto, con override por red
vía config TOML (Semana 4). Endpoints `https://<red>-rpc.publicnode.com`.

**Motivo:** experiencia "clona y arranca" sin fricción ni secretos en el repo,
ideal para un proyecto de portfolio. El precio es el rate-limit, que mitigaremos
en la Semana 9 con caché por TTL, coalescing de peticiones en vuelo y backoff
ante 429. Quien quiera más cuota apunta su Alchemy/Infura desde el TOML.

## D2 — Estrategia de refresco de balances (Semana 2)

**Contexto:** los balances deben mantenerse al día. Opciones: polling periódico
(`eth_getBalance` cada N segundos) vs WebSockets (`eth_subscribe` a nuevos
bloques y refrescar al llegar uno).

**Decisión:** **polling cada ~15 s** para la v1, implementado con `tea.Tick`
(tarea S4-7).

**Motivo:** más simple y portable. Los WebSockets exigen endpoints `wss://`
(no todos los RPC públicos los ofrecen de forma estable), reconexión y manejo de
estado de suscripción — complejidad que no aporta a un dashboard watch-only donde
un retardo de 15 s es irrelevante. El polling encaja de forma natural con el
patrón de mensajes de Bubble Tea (un `tea.Tick` dispara el mismo `tea.Cmd` de
fetch que ya usamos al abrir la pestaña). WebSockets quedan como posible mejora v2.

> Nota de implementación: con publicnode (D1) + polling (D2) es fácil rozar el
> rate-limit. Por eso D1 y D2 empujan juntas hacia la caché/backoff de la S9.

## D3 — Fuente del historial de transacciones (Semana 5)

**Contexto:** mostrar las últimas ~20 txs de una wallet. Opciones: la API de
Etherscan (V2, key gratis) vs escanear bloques nosotros vía RPC.

**Decisión:** **Etherscan API V2**. Endpoint unificado
`https://api.etherscan.io/v2/api?chainid=<id>&...`: una sola API key gratis vale
para Ethereum, Arbitrum, Base y Optimism cambiando solo el `chainid`.

**Motivo:** escanear bloques para reconstruir el historial de una address es
inviable sin un nodo indexado (habría que recorrer millones de bloques y filtrar
logs). Etherscan ya lo tiene indexado y devuelve la lista lista para mostrar. El
coste es depender de un tercero y de una API key, pero es gratis y el plan
free (5 req/s) sobra para una TUI.

**Cómo se configura la key:** vía `etherscan_api_key` en el TOML o la variable
de entorno `ETHERSCAN_API_KEY` (esta última tiene prioridad, para no dejar la
key en archivos). Sin key, la pestaña Transacciones muestra un aviso en la UI en
lugar de romper. El acceso queda detrás de la interfaz `chain.TxProvider` para
poder mockearlo en tests.

## D4 — Decodificación de calls y metadatos de token (Semana 6)

**Contexto:** las txs a contratos llegan como calldata hex (`0xa9059cbb…`). Para
mostrar "Transfer 100 USDC → 0x…" hay que (a) identificar el método por su
selector de 4 bytes y decodificar los argumentos, y (b) traducir la cantidad
cruda a unidades legibles, lo que exige el símbolo y los decimales del token.

**Decisión:**

- **Decodificación de selectores: ABIs locales** para los métodos comunes de
  ERC-20/721 (`transfer`, `transferFrom`, `safeTransferFrom`, `approve`),
  decodificando los argumentos con `accounts/abi` de go-ethereum. 4byte.directory
  queda como posible fallback de v2 para métodos arbitrarios.
- **Metadatos de token (símbolo + decimales): tabla local** de tokens conocidos
  por chain (USDC, USDT, DAI, WETH…) como camino rápido, con **fallback a RPC**
  (`symbol()` / `decimals()`) para tokens fuera de la tabla. El resultado se
  cachea en memoria por `chainID:address` (incluidos los fallos, para no repetir
  RPC ante un token no estándar).

**Motivo:** los selectores de ERC-20/721 son un conjunto pequeño y estable; una
ABI local los cubre sin dependencias de red ni de terceros, y es lo que aporta
valor inmediato en una TUI watch-only. 4byte añadiría cobertura de métodos raros
a cambio de otra dependencia externa y latencia: no compensa en v1. Para los
metadatos, la tabla local resuelve sin RPC el 99 % de los casos reales (los
tokens populares), y el fallback por RPC + caché cubre la cola larga sin castigar
el rate-limit. La resolución queda tras la interfaz `chain.TokenResolver`
(implementada por `chain.Client`) para poder mockearla en tests.
