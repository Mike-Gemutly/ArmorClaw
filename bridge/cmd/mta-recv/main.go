package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	pb "github.com/armorclaw/bridge/pkg/email/proto"
)

const (
	exitSuccess   = 0
	exitTempFail  = 75 // EX_TEMPFAIL — Postfix will retry
	exitPermFail  = 65 // EX_DATAERR — permanent failure
	defaultSocket = "/run/armorclaw/email-ingest.sock"
	readTimeout   = 30 * time.Second
	maxEmailSize  = 26 * 1024 * 1024 // 26MB matching Postfix message_size_limit
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: armorclaw-mta-recv <sender> <recipient> [queue_id]\n")
		os.Exit(exitPermFail)
	}

	envelopeFrom := os.Args[1]
	envelopeTo := os.Args[2]
	queueID := ""
	if len(os.Args) >= 4 {
		queueID = os.Args[3]
	}

	os.Exit(processEmail(envelopeFrom, envelopeTo, queueID))
}

func processEmail(from, to, queueID string) int {
	rawEmail, err := readStdin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "read stdin: %v\n", err)
		return exitTempFail
	}

	if len(rawEmail) > maxEmailSize {
		fmt.Fprintf(os.Stderr, "email exceeds max size: %d > %d\n", len(rawEmail), maxEmailSize)
		return exitPermFail
	}

	resp, err := ingestOverUnixSocket(from, to, queueID, rawEmail)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ingest email: %v\n", err)
		if isTemporaryError(err) {
			return exitTempFail
		}
		return exitPermFail
	}

	if !resp.Accepted {
		fmt.Fprintf(os.Stderr, "email rejected: %s\n", resp.RejectionReason)
		return exitPermFail
	}

	return exitSuccess
}

func readStdin() ([]byte, error) {
	limited := io.LimitReader(os.Stdin, maxEmailSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	if len(data) > maxEmailSize {
		return nil, fmt.Errorf("email exceeds max size")
	}
	return data, nil
}

func ingestOverUnixSocket(from, to, queueID string, rawEmail []byte) (*pb.IngestEmailResponse, error) {
	socketPath := os.Getenv("ARMORCLAW_INGEST_SOCKET")
	if socketPath == "" {
		socketPath = defaultSocket
	}

	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("dial unix socket: %w", err)
	}
	defer conn.Close()

	header := map[string]interface{}{
		"action":   "ingest",
		"from":     from,
		"to":       to,
		"queue_id": queueID,
		"data_len": len(rawEmail),
	}
	headerJSON, _ := json.Marshal(header)
	headerJSON = append(headerJSON, '\n')

	if _, err := conn.Write(headerJSON); err != nil {
		return nil, fmt.Errorf("write header: %w", err)
	}
	if _, err := conn.Write(rawEmail); err != nil {
		return nil, fmt.Errorf("write data: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var respBuf bytes.Buffer
	if _, err := io.Copy(&respBuf, conn); err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var resp pb.IngestEmailResponse
	if err := json.Unmarshal(respBuf.Bytes(), &resp); err != nil {
		return &pb.IngestEmailResponse{
			Accepted: true,
			EmailID:  queueID,
		}, nil
	}

	return &resp, nil
}

func isTemporaryError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}
	return false
}
