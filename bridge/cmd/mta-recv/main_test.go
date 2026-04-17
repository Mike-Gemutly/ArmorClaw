package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestIsTemporaryError_NilError(t *testing.T) {
	if isTemporaryError(nil) {
		t.Error("nil error should not be temporary")
	}
}

func TestIsTemporaryError_NetTimeout(t *testing.T) {
	err := &mockNetError{timeout: true, temporary: false}
	if !isTemporaryError(err) {
		t.Error("timeout error should be temporary")
	}
}

func TestIsTemporaryError_NetTemporary(t *testing.T) {
	err := &mockNetError{timeout: false, temporary: true}
	if !isTemporaryError(err) {
		t.Error("temporary error should be temporary")
	}
}

func TestIsTemporaryError_NetNeither(t *testing.T) {
	err := &mockNetError{timeout: false, temporary: false}
	if isTemporaryError(err) {
		t.Error("non-timeout non-temporary net error should not be temporary")
	}
}

func TestIsTemporaryError_RegularError(t *testing.T) {
	if isTemporaryError(errors.New("plain error")) {
		t.Error("regular error should not be temporary")
	}
}

func TestReadStdin_NormalInput(t *testing.T) {
	input := "From: test@example.com\r\nTo: user@test.com\r\nSubject: Hi\r\n\r\nHello"
	origStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		w.WriteString(input)
		w.Close()
	}()
	defer func() { os.Stdin = origStdin }()

	data, err := readStdin()
	if err != nil {
		t.Fatalf("readStdin: %v", err)
	}
	if string(data) != input {
		t.Errorf("data mismatch: got %q, want %q", string(data), input)
	}
}

func TestReadStdin_ExceedsMaxSize(t *testing.T) {
	origStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		bigData := make([]byte, maxEmailSize+1)
		for i := range bigData {
			bigData[i] = 'X'
		}
		w.Write(bigData)
		w.Close()
	}()
	defer func() { os.Stdin = origStdin }()

	_, err := readStdin()
	if err == nil {
		t.Fatal("expected error for oversized input")
	}
	if !strings.Contains(err.Error(), "max size") {
		t.Errorf("error = %v, want max size message", err)
	}
}

func TestProcessEmail_SizeLimit(t *testing.T) {
	origStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		bigData := make([]byte, maxEmailSize+1)
		for i := range bigData {
			bigData[i] = 'X'
		}
		w.Write(bigData)
		w.Close()
	}()
	defer func() { os.Stdin = origStdin }()

	code := processEmail("from@test.com", "to@test.com", "queue-123")
	if code != exitTempFail && code != exitPermFail {
		t.Errorf("exit code = %d, expected temp or perm failure for oversized email", code)
	}
}

func TestProcessEmail_SocketConnectFails(t *testing.T) {
	input := "From: test@example.com\r\nSubject: Test\r\n\r\nBody"
	origStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		w.WriteString(input)
		w.Close()
	}()
	defer func() { os.Stdin = origStdin }()

	code := processEmail("from@test.com", "to@test.com", "queue-456")
	if code != exitTempFail && code != exitPermFail {
		t.Errorf("exit code = %d, expected temp or perm failure for missing socket", code)
	}
}

type mockNetError struct {
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return "mock net error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }
