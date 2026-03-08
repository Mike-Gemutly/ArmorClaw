package eventlog

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"sync"
)

// Segment represents a single log file
type Segment struct {
	mu   sync.Mutex
	file *os.File
	path string
	size int64
}

func openSegment(dir string, baseOffset uint64) (*Segment, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("%016d.log", baseOffset)
	path := filepath.Join(dir, filename)

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	return &Segment{
		file: file,
		path: path,
		size: info.Size(),
	}, nil
}

// Write appends a record to the segment in binary format
func (s *Segment) Write(rec *Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Serialize payload
	payload, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	// 2. Prepare header (length + checksum)
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[0:4], uint32(len(payload)))

	checksum := crc32.ChecksumIEEE(payload)
	binary.LittleEndian.PutUint32(header[4:8], checksum)

	// 3. Write to file
	if _, err := s.file.Write(header); err != nil {
		return err
	}
	if _, err := s.file.Write(payload); err != nil {
		return err
	}

	s.size += int64(len(header) + len(payload))
	return nil
}

func (s *Segment) Sync() error {
	return s.file.Sync()
}

func (s *Segment) Close() error {
	return s.file.Close()
}

func (s *Segment) Size() int64 {
	return s.size
}
