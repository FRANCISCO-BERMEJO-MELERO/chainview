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
