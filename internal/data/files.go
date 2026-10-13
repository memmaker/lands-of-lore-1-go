// Package data reads the game's files: the archives (.PAK) and their index FILEDATA.FDT,
// and the Westwood formats inside them.
package data

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// indexFile lists every file of the game: its size, and for the files inside a .PAK which
// archive holds it and where.
const indexFile = "FILEDATA.FDT"

// indexEntrySize is the size of one FILEDATA.FDT record:
// name offset (4), size (4), cache pointer (4), offset in the archive (4), archive (1),
// open count (1), flags (2).
const indexEntrySize = 20

// insideArchive marks an index entry whose file lives inside an archive.
const insideArchive = 0x1000

// entry is one file of the index.
type entry struct {
	name    string
	size    int
	offset  int
	archive int // index of the archive's own entry
	packed  bool
}

// Files finds the game's files by name, as the original's file layer (s034, s035): a loose
// file in the game folder wins, else the index says which archive holds it.
type Files struct {
	dir     string
	entries []entry
	byName  map[string]int
	cache   map[string][]byte // whole archives, read once
}

// Open reads the index from the game folder dir.
func Open(dir string) (*Files, error) {
	f := &Files{dir: dir, byName: map[string]int{}, cache: map[string][]byte{}}
	raw, err := os.ReadFile(f.path(indexFile))
	if err != nil {
		return nil, err
	}
	for at := 0; at+indexEntrySize <= len(raw); at += indexEntrySize {
		nameAt := int(binary.LittleEndian.Uint32(raw[at:]))
		if nameAt == 0 {
			break
		}
		end := bytes.IndexByte(raw[nameAt:], 0)
		e := entry{
			name:    strings.ToUpper(string(raw[nameAt : nameAt+end])),
			size:    int(binary.LittleEndian.Uint32(raw[at+4:])),
			offset:  int(binary.LittleEndian.Uint32(raw[at+12:])),
			archive: int(raw[at+16]),
			packed:  binary.LittleEndian.Uint16(raw[at+18:])&insideArchive != 0,
		}
		f.byName[e.name] = len(f.entries)
		f.entries = append(f.entries, e)
	}
	return f, nil
}

// path is the name of a loose file in the game folder, matching its case-insensitively.
func (f *Files) path(name string) string {
	p := filepath.Join(f.dir, name)
	if _, err := os.Stat(p); err == nil {
		return p
	}
	if list, err := os.ReadDir(f.dir); err == nil {
		for _, d := range list {
			if strings.EqualFold(d.Name(), name) {
				return filepath.Join(f.dir, d.Name())
			}
		}
	}
	return p
}

// Exists says whether the file can be read (C: fn_1981_000a).
func (f *Files) Exists(name string) bool {
	if i, ok := f.byName[strings.ToUpper(name)]; ok && f.entries[i].size != 0 {
		return true
	}
	_, err := os.Stat(f.path(name))
	return err == nil
}

// Read returns a file's contents.
func (f *Files) Read(name string) ([]byte, error) {
	name = strings.ToUpper(name)
	if b, err := os.ReadFile(f.path(name)); err == nil {
		return b, nil
	}
	i, ok := f.byName[name]
	if !ok || !f.entries[i].packed {
		return nil, fmt.Errorf("%s: not found", name)
	}
	e := f.entries[i]
	archive := f.entries[e.archive].name
	whole, ok := f.cache[archive]
	if !ok {
		var err error
		if whole, err = os.ReadFile(f.path(archive)); err != nil {
			return nil, err
		}
		f.cache[archive] = whole
		f.readDirectory(e.archive, whole)
		e = f.entries[i]
	}
	if e.offset+e.size > len(whole) {
		return nil, fmt.Errorf("%s: beyond the end of %s", name, archive)
	}
	return whole[e.offset : e.offset+e.size : e.offset+e.size], nil
}

// readDirectory replaces the index's offsets and sizes of an archive's members by those of
// the archive's own directory, as the original does when it first opens an archive: the
// index's values are stale. The directory is a list of (offset (4), name, 0) ending with
// an offset of 0; each member runs to the next one's offset, the last to the archive's end.
func (f *Files) readDirectory(archive int, whole []byte) {
	prev := -1
	for at := 0; at+directoryOffsetSize <= len(whole); {
		offset := int(binary.LittleEndian.Uint32(whole[at:]))
		if offset == 0 {
			break
		}
		at += directoryOffsetSize
		end := bytes.IndexByte(whole[at:], 0)
		if end < 0 {
			break
		}
		name := strings.ToUpper(string(whole[at : at+end]))
		at += end + 1
		i, ok := f.byName[name]
		if !ok || !f.entries[i].packed || f.entries[i].archive != archive {
			continue
		}
		f.entries[i].offset = offset
		if prev >= 0 {
			f.entries[prev].size = offset - f.entries[prev].offset
		}
		prev = i
	}
	if prev >= 0 {
		f.entries[prev].size = len(whole) - f.entries[prev].offset
	}
}

// directoryOffsetSize is the size of a member's offset in an archive's directory.
const directoryOffsetSize = 4

// Size returns a file's length, 0 if it is missing (C: fn_1809_120c on a fresh handle).
func (f *Files) Size(name string) int {
	if fi, err := os.Stat(f.path(name)); err == nil {
		return int(fi.Size())
	}
	if b, err := f.Read(name); err == nil {
		return len(b)
	}
	return 0
}
