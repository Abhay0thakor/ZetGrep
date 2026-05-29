package utils

import (
	"bytes"
	"os"

	"github.com/edsrzf/mmap-go"
)

type MmapFile struct {
	Data mmap.MMap
	File *os.File
}

func OpenMmap(path string) (*MmapFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	m, err := mmap.Map(f, mmap.RDONLY, 0)
	if err != nil {
		f.Close()
		return nil, err
	}

	return &MmapFile{
		Data: m,
		File: f,
	}, nil
}

func (m *MmapFile) Close() error {
	_ = m.Data.Unmap()
	return m.File.Close()
}

// LineIterator provides a fast way to iterate over lines in a byte slice
type LineIterator struct {
	data []byte
	pos  int
}

func NewLineIterator(data []byte) *LineIterator {
	return &LineIterator{data: data}
}

func (li *LineIterator) Next() ([]byte, bool) {
	if li.pos >= len(li.data) {
		return nil, false
	}

	start := li.pos
	end := bytes.IndexByte(li.data[start:], '\n')
	if end == -1 {
		li.pos = len(li.data)
		return li.data[start:], true
	}

	li.pos += end + 1
	// Trim \r if present (Windows line endings)
	line := li.data[start : start+end]
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return line, true
}
