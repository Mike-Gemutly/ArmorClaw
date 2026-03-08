package eventlog

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ReadFrom reads records starting from a specific offset
func (l *Log) ReadFrom(offset uint64, limit int) ([]*Record, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.readFromLocked(offset, limit)
}

func (l *Log) readFromLocked(offset uint64, limit int) ([]*Record, error) {
	if limit <= 0 {
		limit = 100
	}

	// 1. Find segments that might contain the offset
	segments, err := l.listSegments()
	if err != nil {
		return nil, err
	}

	// 2. Find the segment where baseOffset <= offset
	var targetSegment string
	for i := len(segments) - 1; i >= 0; i-- {
		if segments[i].baseOffset <= offset {
			targetSegment = segments[i].path
			break
		}
	}

	if targetSegment == "" {
		return nil, nil // Offset not found (too old or too new)
	}

	// 3. Read from segments until limit reached
	var results []*Record
	currentPath := targetSegment

	for {
		records, err := l.readSegment(currentPath, offset, limit-len(results))
		if err != nil {
			return nil, err
		}

		results = append(results, records...)
		if len(results) >= limit {
			break
		}

		// Move to next segment if available
		nextPath := l.getNextSegment(currentPath, segments)
		if nextPath == "" {
			break
		}
		currentPath = nextPath
	}

	return results, nil
}

type segmentInfo struct {
	baseOffset uint64
	path       string
}

func (l *Log) listSegments() ([]segmentInfo, error) {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return nil, err
	}

	var segments []segmentInfo
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			baseStr := strings.TrimSuffix(entry.Name(), ".log")
			base, err := strconv.ParseUint(baseStr, 10, 64)
			if err != nil {
				continue
			}
			segments = append(segments, segmentInfo{
				baseOffset: base,
				path:       filepath.Join(l.dir, entry.Name()),
			})
		}
	}

	sort.Slice(segments, func(i, j int) bool {
		return segments[i].baseOffset < segments[j].baseOffset
	})

	return segments, nil
}

func (l *Log) getNextSegment(currentPath string, segments []segmentInfo) string {
	for i := 0; i < len(segments)-1; i++ {
		if segments[i].path == currentPath {
			return segments[i+1].path
		}
	}
	return ""
}

func (l *Log) readSegment(path string, startOffset uint64, limit int) ([]*Record, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []*Record
	header := make([]byte, 8)

	for {
		if len(records) >= limit {
			break
		}

		_, err := io.ReadFull(file, header)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		length := binary.LittleEndian.Uint32(header[0:4])
		checksum := binary.LittleEndian.Uint32(header[4:8])

		payload := make([]byte, length)
		if _, err := io.ReadFull(file, payload); err != nil {
			return nil, err
		}

		if crc32.ChecksumIEEE(payload) != checksum {
			return nil, fmt.Errorf("corruption detected at offset in segment %s", path)
		}

		var rec Record
		if err := json.Unmarshal(payload, &rec); err != nil {
			return nil, err
		}

		if rec.Offset >= startOffset {
			records = append(records, &rec)
		}
	}

	return records, nil
}
