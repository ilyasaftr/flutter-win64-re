package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// FolderFile is the shared in-memory shape of Reqable list configs: a
// named item array plus a top-level enabled switch. The JSON key differs
// per file ("rewrites", "breakpoints", "collections"), so loadFolder and
// saveFolder take it as a parameter.
type FolderFile[T any] struct {
	Items     []T
	IsEnabled bool
}

// loadFolder reads {"<key>": [...], "isEnabled": bool}.
func loadFolder[T any](path, key string) (FolderFile[T], error) {
	var raw map[string]json.RawMessage
	if err := loadFile(path, &raw); err != nil {
		return FolderFile[T]{}, err
	}
	var f FolderFile[T]
	if b, ok := raw[key]; ok {
		if err := json.Unmarshal(b, &f.Items); err != nil {
			return FolderFile[T]{}, fmt.Errorf("%s: %w", path, err)
		}
	}
	if b, ok := raw["isEnabled"]; ok {
		if err := json.Unmarshal(b, &f.IsEnabled); err != nil {
			return FolderFile[T]{}, fmt.Errorf("%s: %w", path, err)
		}
	}
	return f, nil
}

// saveFolder writes {"rewrites"|"breakpoints"|"collections": [...],
// "isEnabled": bool} atomically, preserving the app's key order.
func saveFolder[T any](path, key string, f FolderFile[T]) error {
	items, err := json.Marshal(f.Items)
	if err != nil {
		return err
	}
	on, err := json.Marshal(f.IsEnabled)
	if err != nil {
		return err
	}
	return saveFile(path, orderedDoc{key: key, items: items, on: on})
}

func indexByID[T any](items []T, id string, idOf func(T) string) int {
	for i, e := range items {
		if idOf(e) == id {
			return i
		}
	}
	return -1
}

// GetByID returns the item with id or a "<kind> <id> not found" error.
func GetByID[T any](items []T, kind, id string, idOf func(T) string) (T, error) {
	var zero T
	if i := indexByID(items, id, idOf); i >= 0 {
		return items[i], nil
	}
	return zero, fmt.Errorf("%s %s not found", kind, id)
}

// Delete removes the item with id, returning the kept slice and found.
func Delete[T any](items []T, id string, idOf func(T) string) ([]T, bool) {
	i := indexByID(items, id, idOf)
	if i < 0 {
		return items, false
	}
	return append(items[:i], items[i+1:]...), true
}

// SetEnabled flips the top-level switch, preserving the app's key order
// (items array first, isEnabled last) via an order-preserving doc.
func SetEnabled(path, key string, enabled bool) error {
	raw, err := loadRaw(path)
	if err != nil {
		return err
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	items, ok := doc[key]
	if !ok {
		items = json.RawMessage("[]")
	}
	on, err := json.Marshal(enabled)
	if err != nil {
		return err
	}
	return saveFile(path, orderedDoc{key: key, items: items, on: on})
}

// orderedDoc marshals as {"<key>": [...], "isEnabled": bool}.
type orderedDoc struct {
	key   string
	items json.RawMessage
	on    json.RawMessage
}

// MarshalJSON preserves app key order (items first, isEnabled last).
func (d orderedDoc) MarshalJSON() ([]byte, error) {
	var sb bytes.Buffer
	sb.WriteString(`{"` + d.key + `":`)
	sb.Write(d.items)
	sb.WriteString(`,"isEnabled":`)
	sb.Write(d.on)
	sb.WriteString(`}`)
	return sb.Bytes(), nil
}

// loadRaw reads a config file's exact bytes.
func loadRaw(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// SetItemEnabled flips one item's switch via set; missing ids error.
func SetItemEnabled[T any](items []T, kind, id string, enabled bool, idOf func(T) string, set func(*T, bool)) error {
	i := indexByID(items, id, idOf)
	if i < 0 {
		return fmt.Errorf("%s %s not found", kind, id)
	}
	set(&items[i], enabled)
	return nil
}
