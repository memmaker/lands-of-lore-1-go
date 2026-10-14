package data

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Archives is LANDS.EXE's file layer: it runs in the language folder, where a loose file
// wins and else the first of its archives that holds the file gives it (C: LANDS.EXE
// _Find_File over the PAK table _Init_FileData_Table reads).
type Archives struct {
	dir     string
	members map[string]member
}

type member struct {
	archive     string
	offset, end int64
}

// OpenArchives reads the directories of the named archives in dir, the earlier ones
// first; missing archives are skipped.
func OpenArchives(dir string, names []string) *Archives {
	a := &Archives{dir: dir, members: map[string]member{}}
	for _, n := range names {
		p, ok := findIn(dir, n)
		if !ok {
			continue
		}
		a.readDirectory(p)
	}
	return a
}

// readDirectory adds an archive's members: offsets and names up to an offset of 0 or the
// first member's data, each member running to the next one's offset.
func (a *Archives) readDirectory(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return
	}
	var head [directoryOffsetSize]byte
	if _, err := io.ReadFull(f, head[:]); err != nil {
		return
	}
	first := int64(binary.LittleEndian.Uint32(head[:]))
	if first <= 0 || first > fi.Size() {
		return
	}
	dir := make([]byte, first)
	if _, err := f.ReadAt(dir, 0); err != nil {
		return
	}
	type ent struct {
		name   string
		offset int64
	}
	var list []ent
	for at := 0; at+directoryOffsetSize <= len(dir); {
		off := int64(binary.LittleEndian.Uint32(dir[at:]))
		if off == 0 {
			break
		}
		at += directoryOffsetSize
		end := bytes.IndexByte(dir[at:], 0)
		if end < 0 {
			break
		}
		list = append(list, ent{strings.ToUpper(string(dir[at : at+end])), off})
		at += end + 1
	}
	for i, e := range list {
		end := fi.Size()
		if i+1 < len(list) {
			end = list[i+1].offset
		}
		if _, ok := a.members[e.name]; !ok {
			a.members[e.name] = member{path, e.offset, end}
		}
	}
}

// Exists says whether a file can be read (C: _Find_File).
func (a *Archives) Exists(name string) bool {
	if _, ok := findIn(a.dir, name); ok {
		return true
	}
	_, ok := a.members[strings.ToUpper(name)]
	return ok
}

// Read returns a file's contents.
func (a *Archives) Read(name string) ([]byte, error) {
	if p, ok := findIn(a.dir, name); ok {
		return os.ReadFile(p)
	}
	m, ok := a.members[strings.ToUpper(name)]
	if !ok {
		return nil, fmt.Errorf("%s: not found", name)
	}
	f, err := os.Open(m.archive)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b := make([]byte, m.end-m.offset)
	if _, err := f.ReadAt(b, m.offset); err != nil && err != io.EOF {
		return nil, err
	}
	return b, nil
}

// LanguageDir is the language folder in use.
func (f *Files) LanguageDir() string {
	if p, ok := findIn(f.dir, f.lang); ok {
		return p
	}
	return filepath.Join(f.dir, f.lang)
}
