package chain

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ensRegistry es el contrato registry de ENS. Tiene la misma dirección en todas
// las redes donde ENS está desplegado; nosotros solo resolvemos en mainnet.
var ensRegistry = common.HexToAddress("0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e")

// errENSNotFound indica que no hay resolución (sin resolver, sin registro o sin
// nombre inverso). Es un "no encontrado" normal, no un fallo de red.
var errENSNotFound = errors.New("ENS: no resolution")

// namehash implementa el algoritmo de EIP-137: convierte un nombre como
// "vitalik.eth" en el nodo de 32 bytes con el que se le pregunta a ENS. Se
// construye de derecha a izquierda, hasheando cada label sobre el acumulado:
//
//	node = keccak256(node_padre || keccak256(label))
//
// partiendo de 32 bytes a cero para la raíz.
func namehash(name string) common.Hash {
	node := common.Hash{} // raíz: 32 bytes a cero
	if name == "" {
		return node
	}
	labels := strings.Split(name, ".")
	for i := len(labels) - 1; i >= 0; i-- {
		labelHash := crypto.Keccak256Hash([]byte(labels[i]))
		node = crypto.Keccak256Hash(node.Bytes(), labelHash.Bytes())
	}
	return node
}

// ensABIJSON cubre los tres métodos que necesitamos: resolver() en el registry,
// y addr()/name() en los resolvers (directo e inverso).
const ensABIJSON = `[
	{"name":"resolver","type":"function","stateMutability":"view","inputs":[{"name":"node","type":"bytes32"}],"outputs":[{"type":"address"}]},
	{"name":"addr","type":"function","stateMutability":"view","inputs":[{"name":"node","type":"bytes32"}],"outputs":[{"type":"address"}]},
	{"name":"name","type":"function","stateMutability":"view","inputs":[{"name":"node","type":"bytes32"}],"outputs":[{"type":"string"}]}
]`

var ensABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(ensABIJSON))
	if err != nil {
		panic("chain: ABI de ENS inválida: " + err.Error())
	}
	return a
}()

// resolverFor consulta al registry qué resolver gestiona un nodo. Devuelve
// errENSNotFound si el nodo no tiene resolver (dirección cero).
func resolverFor(ctx context.Context, caller contractCaller, node common.Hash) (common.Address, error) {
	out, err := ensCall(ctx, caller, ensRegistry, "resolver", [32]byte(node))
	if err != nil {
		return common.Address{}, err
	}
	var resolver common.Address
	if err := ensABI.UnpackIntoInterface(&resolver, "resolver", out); err != nil {
		return common.Address{}, err
	}
	if resolver == (common.Address{}) {
		return common.Address{}, errENSNotFound
	}
	return resolver, nil
}

// resolveENS traduce un nombre ENS a su dirección (resolución directa).
func resolveENS(ctx context.Context, caller contractCaller, name string) (common.Address, error) {
	node := namehash(name)
	resolver, err := resolverFor(ctx, caller, node)
	if err != nil {
		return common.Address{}, err
	}
	out, err := ensCall(ctx, caller, resolver, "addr", [32]byte(node))
	if err != nil {
		return common.Address{}, err
	}
	var addr common.Address
	if err := ensABI.UnpackIntoInterface(&addr, "addr", out); err != nil {
		return common.Address{}, err
	}
	if addr == (common.Address{}) {
		return common.Address{}, errENSNotFound
	}
	return addr, nil
}

// reverseENS traduce una dirección a su nombre ENS (resolución inversa). El
// registro inverso vive bajo "<addr-en-hex-sin-0x>.addr.reverse". Como el nombre
// inverso lo fija quien controla la dirección y NO es de fiar por sí solo, lo
// verificamos resolviéndolo hacia delante: solo aceptamos el nombre si resuelve
// de vuelta a la misma dirección (anti-spoofing, como recomienda el estándar).
func reverseENS(ctx context.Context, caller contractCaller, addr common.Address) (string, error) {
	reverseName := strings.ToLower(addr.Hex()[2:]) + ".addr.reverse"
	node := namehash(reverseName)

	resolver, err := resolverFor(ctx, caller, node)
	if err != nil {
		return "", err
	}
	out, err := ensCall(ctx, caller, resolver, "name", [32]byte(node))
	if err != nil {
		return "", err
	}
	var name string
	if err := ensABI.UnpackIntoInterface(&name, "name", out); err != nil {
		return "", err
	}
	if name == "" {
		return "", errENSNotFound
	}

	// Verificación directa: el nombre debe resolver de vuelta a addr.
	forward, err := resolveENS(ctx, caller, name)
	if err != nil {
		return "", err
	}
	if forward != addr {
		return "", errENSNotFound
	}
	return name, nil
}

// ensCall empaqueta y ejecuta una llamada view de ENS contra un contrato.
func ensCall(ctx context.Context, caller contractCaller, contract common.Address, method string, args ...interface{}) ([]byte, error) {
	data, err := ensABI.Pack(method, args...)
	if err != nil {
		return nil, err
	}
	return caller.CallContract(ctx, ethereum.CallMsg{To: &contract, Data: data}, nil)
}

// ensTTL es cuánto vale una resolución cacheada. Los registros ENS cambian poco,
// así que un TTL largo basta y evita repetir RPC; al reiniciar la app la caché se
// vacía igualmente (decisión D6: caché en memoria, sin persistencia).
const ensTTL = 6 * time.Hour

// ENSResolver resuelve nombres ENS con una caché en memoria con TTL (D6). Es
// thread-safe: varios tea.Cmd pueden resolver a la vez. Cachea tanto los aciertos
// como los "sin nombre" (negativos), pero NO los fallos de red, para no quedarse
// pegado a un vacío por una caída puntual de conexión.
type ENSResolver struct {
	dial func(context.Context) (contractCaller, error) // obtiene la conexión a mainnet
	ttl  time.Duration
	now  func() time.Time // inyectable para testear el vencimiento

	mu  sync.RWMutex
	rev map[common.Address]revEntry // inversa: address -> nombre
	fwd map[string]fwdEntry         // directa: nombre -> address
}

type revEntry struct {
	name string // "" = la dirección no tiene nombre inverso válido
	at   time.Time
}

type fwdEntry struct {
	addr common.Address // cero = el nombre no resuelve
	at   time.Time
}

// NewENSResolver crea el resolver sobre la conexión de Ethereum mainnet del
// cliente (ENS solo se resuelve en mainnet). La conexión se obtiene de forma
// perezosa en la primera resolución, no al construir.
func NewENSResolver(client *Client) *ENSResolver {
	return &ENSResolver{
		dial: func(ctx context.Context) (contractCaller, error) {
			return client.connect(ChainEthereum)
		},
		ttl: ensTTL,
		now: time.Now,
		rev: make(map[common.Address]revEntry),
		fwd: make(map[string]fwdEntry),
	}
}

// Lookup resuelve la inversa (address -> nombre). El segundo valor es false si la
// dirección no tiene nombre ENS verificable. Cachea el resultado.
func (r *ENSResolver) Lookup(ctx context.Context, addr common.Address) (string, bool) {
	r.mu.RLock()
	e, ok := r.rev[addr]
	r.mu.RUnlock()
	if ok && r.now().Sub(e.at) < r.ttl {
		return e.name, e.name != ""
	}

	caller, err := r.dial(ctx)
	if err != nil {
		return "", false // fallo de conexión: no cacheamos
	}

	name, err := reverseENS(ctx, caller, addr)
	switch {
	case err == nil:
		r.put(func() { r.rev[addr] = revEntry{name: name, at: r.now()} })
		return name, true
	case errors.Is(err, errENSNotFound):
		r.put(func() { r.rev[addr] = revEntry{name: "", at: r.now()} })
		return "", false
	default:
		return "", false // error de red: no cacheamos
	}
}

// Resolve resuelve la directa (nombre -> address). El segundo valor es false si
// el nombre no resuelve. Cachea el resultado.
func (r *ENSResolver) Resolve(ctx context.Context, name string) (common.Address, bool) {
	name = strings.ToLower(strings.TrimSpace(name))

	r.mu.RLock()
	e, ok := r.fwd[name]
	r.mu.RUnlock()
	if ok && r.now().Sub(e.at) < r.ttl {
		return e.addr, e.addr != (common.Address{})
	}

	caller, err := r.dial(ctx)
	if err != nil {
		return common.Address{}, false
	}

	addr, err := resolveENS(ctx, caller, name)
	switch {
	case err == nil:
		r.put(func() { r.fwd[name] = fwdEntry{addr: addr, at: r.now()} })
		return addr, true
	case errors.Is(err, errENSNotFound):
		r.put(func() { r.fwd[name] = fwdEntry{addr: common.Address{}, at: r.now()} })
		return common.Address{}, false
	default:
		return common.Address{}, false
	}
}

// put ejecuta una escritura en la caché bajo el lock de escritura.
func (r *ENSResolver) put(write func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	write()
}
