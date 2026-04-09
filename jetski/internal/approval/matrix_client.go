package approval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type OperationType string

const (
	OpSessionCreate OperationType = "session_create"
	OpNavigation    OperationType = "navigation"
	OpFileDownload  OperationType = "file_download"
)

type ApprovalRequest struct {
	ID          string        `json:"id"`
	Operation   OperationType `json:"operation"`
	Detail      string        `json:"detail"`
	RoomID      string        `json:"room_id"`
	RequestedAt time.Time     `json:"requested_at"`
	Status      string        `json:"status"`
}

type ApprovalClient struct {
	baseURL string
	roomID  string
	timeout time.Duration
	pending map[string]chan string
	mu      sync.RWMutex
	cancel  context.CancelFunc
}

func NewApprovalClient(baseURL, roomID string, timeout time.Duration) *ApprovalClient {
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	ac := &ApprovalClient{
		baseURL: baseURL,
		roomID:  roomID,
		timeout: timeout,
		pending: make(map[string]chan string),
		cancel:  cancel,
	}
	go ac.cleanupLoop(ctx)
	return ac
}

func (ac *ApprovalClient) RequestApproval(ctx context.Context, op OperationType, detail string) (bool, error) {
	id := fmt.Sprintf("apr-%d", time.Now().UnixNano())
	ch := make(chan string, 1)

	ac.mu.Lock()
	ac.pending[id] = ch
	ac.mu.Unlock()

	defer func() {
		ac.mu.Lock()
		delete(ac.pending, id)
		ac.mu.Unlock()
	}()

	req := ApprovalRequest{
		ID:          id,
		Operation:   op,
		Detail:      detail,
		RoomID:      ac.roomID,
		RequestedAt: time.Now().UTC(),
		Status:      "pending",
	}

	body, _ := json.Marshal(req)
	_, _ = http.Post(ac.baseURL+"/rpc/approval/request", "application/json", bytes.NewReader(body))

	timer := time.NewTimer(ac.timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case result := <-ch:
		return result == "approved", nil
	case <-timer.C:
		return false, nil
	}
}

func (ac *ApprovalClient) HandleApprovalResponse(id string, approved bool) {
	ac.mu.RLock()
	ch, exists := ac.pending[id]
	ac.mu.RUnlock()

	if !exists {
		return
	}

	result := "denied"
	if approved {
		result = "approved"
	}

	select {
	case ch <- result:
	default:
	}
}

func (ac *ApprovalClient) PendingCount() int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return len(ac.pending)
}

func (ac *ApprovalClient) Close() {
	ac.cancel()
	ac.mu.Lock()
	for id, ch := range ac.pending {
		close(ch)
		delete(ac.pending, id)
	}
	ac.mu.Unlock()
}

func (ac *ApprovalClient) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ac.mu.Lock()
			for id, ch := range ac.pending {
				select {
				case <-ch:
				default:
					_ = id
				}
			}
			ac.mu.Unlock()
		}
	}
}

func RegisterApprovalHandlers(mux *http.ServeMux, ac *ApprovalClient) {
	mux.HandleFunc("/rpc/approval/response", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			ID       string `json:"id"`
			Approved bool   `json:"approved"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}

		ac.HandleApprovalResponse(req.ID, req.Approved)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/rpc/approval/pending", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"count":   ac.PendingCount(),
			"pending": ac.pendingIDs(),
		})
	})
}

func (ac *ApprovalClient) pendingIDs() []string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	ids := make([]string, 0, len(ac.pending))
	for id := range ac.pending {
		ids = append(ids, id)
	}
	return ids
}
