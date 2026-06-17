package storage

import (
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const (
	vitalik = "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"
	zero    = "0x0000000000000000000000000000000000000000"
)

func TestLoadFromMissingFileIsEmpty(t *testing.T) {
	w, err := loadFrom(filepath.Join(t.TempDir(), "wallets.json"))
	if err != nil {
		t.Fatalf("archivo inexistente no debería dar error: %v", err)
	}
	if w.Len() != 0 {
		t.Fatalf("esperaba lista vacía, hay %d", w.Len())
	}
}

func TestAddValidatesAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallets.json")
	w, _ := loadFrom(path)

	if err := w.Add("no-es-una-address"); err == nil {
		t.Fatal("esperaba error con address inválida")
	}
	if err := w.Add(vitalik); err != nil {
		t.Fatalf("Add válida: %v", err)
	}
	// Duplicado: idempotente, no crece.
	if err := w.Add(vitalik); err != nil {
		t.Fatalf("Add duplicada: %v", err)
	}
	if w.Len() != 1 {
		t.Fatalf("esperaba 1 wallet, hay %d", w.Len())
	}

	// Recargar desde disco debe ver la wallet persistida.
	reloaded, err := loadFrom(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Len() != 1 || reloaded.List()[0] != common.HexToAddress(vitalik) {
		t.Fatalf("la wallet no persistió correctamente: %+v", reloaded.List())
	}
}

func TestRemovePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallets.json")
	w, _ := loadFrom(path)
	_ = w.Add(vitalik)
	_ = w.Add(zero)

	if err := w.Remove(common.HexToAddress(vitalik)); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if w.Len() != 1 {
		t.Fatalf("esperaba 1 wallet tras borrar, hay %d", w.Len())
	}

	reloaded, _ := loadFrom(path)
	if reloaded.Len() != 1 || reloaded.List()[0] != common.HexToAddress(zero) {
		t.Fatalf("el borrado no persistió: %+v", reloaded.List())
	}
}
