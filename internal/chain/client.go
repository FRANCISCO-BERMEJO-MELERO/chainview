package chain

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/sync/singleflight"
)

// Client es un cliente multi-red sobre go-ethereum. Abre las conexiones RPC
// bajo demanda (lazy) y las cachea por chain ID, de modo que arrancar la app no
// dispara ninguna conexión: solo se conecta a una red la primera vez que se
// consulta algo en ella.
//
// Es seguro para uso concurrente: varios tea.Cmd pueden pedir conexiones a la
// vez (p.ej. al refrescar balances de 4 redes en paralelo) sin abrir conexiones
// duplicadas ni provocar data races.
type Client struct {
	mu       sync.RWMutex                 // protege el mapa conns
	networks map[uint64]Network           // redes conocidas, indexadas por chain ID
	conns    map[uint64]*ethclient.Client // conexiones ya abiertas (cache lazy)

	tokenMu    sync.RWMutex          // protege tokenCache
	tokenCache map[string]tokenEntry // metadatos de token resueltos por "chainID:address"

	rpcMu    sync.Mutex           // protege rpcCache y cooldown
	rpcCache map[string]rpcEntry  // balances/gas cacheados con TTL, por clave
	cooldown map[uint64]time.Time // chain ID -> hasta cuándo está en cooldown por 429
	sf       singleflight.Group   // coalescing de lecturas RPC idénticas concurrentes
}

// tokenEntry es una entrada de la caché de metadatos de token. Guardamos también
// los fallos (ok=false) para no reintentar RPC ante un token no estándar.
type tokenEntry struct {
	meta TokenMeta
	ok   bool
}

// NewClient construye el cliente a partir de una lista de redes. No abre
// ninguna conexión todavía.
func NewClient(networks []Network) *Client {
	nets := make(map[uint64]Network, len(networks))
	for _, n := range networks {
		nets[n.ChainID] = n
	}
	return &Client{
		networks:   nets,
		conns:      make(map[uint64]*ethclient.Client),
		tokenCache: make(map[string]tokenEntry),
		rpcCache:   make(map[string]rpcEntry),
		cooldown:   make(map[uint64]time.Time),
	}
}

// connect devuelve la conexión de la red indicada, abriéndola solo la primera
// vez. Usa el patrón "double-checked locking" con un RWMutex:
//
//  1. Camino rápido: tomamos un read-lock (varios lectores en paralelo) y, si la
//     conexión ya existe en cache, la devolvemos sin bloquear a nadie.
//  2. Si no existe, tomamos el write-lock exclusivo para crearla.
//  3. Volvemos a comprobar dentro del write-lock ("double check"): entre soltar
//     el read-lock y tomar el write-lock, otra goroutine pudo haberla creado ya.
//     Sin esta segunda comprobación abriríamos conexiones duplicadas.
func (c *Client) connect(chainID uint64) (*ethclient.Client, error) {
	// (1) Camino rápido, solo lectura.
	c.mu.RLock()
	conn, ok := c.conns[chainID]
	c.mu.RUnlock()
	if ok {
		return conn, nil
	}

	// (2) Hay que crearla: lock de escritura exclusivo.
	c.mu.Lock()
	defer c.mu.Unlock()

	// (3) Double-check: ¿la creó otra goroutine mientras esperábamos el lock?
	if conn, ok := c.conns[chainID]; ok {
		return conn, nil
	}

	net, ok := c.networks[chainID]
	if !ok {
		return nil, fmt.Errorf("red desconocida: chain id %d", chainID)
	}

	// ethclient.Dial no hace I/O de red para endpoints HTTP: solo prepara el
	// cliente. La conexión real se establece en la primera llamada RPC.
	conn, err := ethclient.Dial(net.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("conectando a %s (chain id %d): %w", net.Name, chainID, err)
	}
	c.conns[chainID] = conn
	return conn, nil
}

// Networks devuelve las redes configuradas, en orden de chain ID no garantizado.
// Pensado para que la UI sepa qué redes existen sin tocar el mapa interno.
func (c *Client) Networks() []Network {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Network, 0, len(c.networks))
	for _, n := range c.networks {
		out = append(out, n)
	}
	return out
}

// Close cierra todas las conexiones abiertas. Conviene llamarlo al salir de la
// app para no dejar conexiones colgando.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, conn := range c.conns {
		conn.Close()
		delete(c.conns, id)
	}
}
