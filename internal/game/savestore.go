package game

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// SaveStore holds the files the game writes: saved games and LANDS.CFG. The original
// wrote them into the game folder, case-insensitively as DOS does.
type SaveStore interface {
	Read(name string) ([]byte, error)
	Write(name string, b []byte) error
	Remove(name string) error
	Rename(from, to string) error
	Exists(name string) bool
}

// dirStore keeps the files in a folder, matching names case-insensitively.
type dirStore struct{ dir string }

func (d dirStore) path(name string) string {
	entries, err := os.ReadDir(d.dir)
	if err == nil {
		for _, e := range entries {
			if strings.EqualFold(e.Name(), name) {
				return filepath.Join(d.dir, e.Name())
			}
		}
	}
	return filepath.Join(d.dir, name)
}

func (d dirStore) Read(name string) ([]byte, error) { return os.ReadFile(d.path(name)) }
func (d dirStore) Write(name string, b []byte) error {
	return os.WriteFile(d.path(name), b, 0o644)
}
func (d dirStore) Remove(name string) error { return os.Remove(d.path(name)) }
func (d dirStore) Rename(from, to string) error {
	return os.Rename(d.path(from), d.path(to))
}
func (d dirStore) Exists(name string) bool {
	_, err := os.Stat(d.path(name))
	return err == nil
}

// MemoryStore keeps the files in memory, for tests and the browser.
type MemoryStore map[string][]byte

var errNoFile = errors.New("no such file")

func (m MemoryStore) Read(name string) ([]byte, error) {
	b, ok := m[strings.ToUpper(name)]
	if !ok {
		return nil, errNoFile
	}
	return b, nil
}
func (m MemoryStore) Write(name string, b []byte) error {
	m[strings.ToUpper(name)] = append([]byte(nil), b...)
	return nil
}
func (m MemoryStore) Remove(name string) error { delete(m, strings.ToUpper(name)); return nil }
func (m MemoryStore) Rename(from, to string) error {
	b, ok := m[strings.ToUpper(from)]
	if !ok {
		return errNoFile
	}
	delete(m, strings.ToUpper(from))
	m[strings.ToUpper(to)] = b
	return nil
}
func (m MemoryStore) Exists(name string) bool { _, ok := m[strings.ToUpper(name)]; return ok }
