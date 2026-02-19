// Package appservice provides client methods for AppService operations
// This file contains methods for interacting with the Matrix homeserver
package appservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client provides methods for AppService to interact with the homeserver
type Client struct {
	homeserverURL string
	asToken       string
	httpClient    *http.Client
}

// NewClient creates a new AppService client
func NewClient(homeserverURL, asToken string) *Client {
	return &Client{
		homeserverURL: homeserverURL,
		asToken:       asToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RoomCreateRequest represents a room creation request
type RoomCreateRequest struct {
	Name        string                 `json:"name,omitempty"`
	Topic       string                 `json:"topic,omitempty"`
	Visibility  string                 `json:"visibility,omitempty"`  // public, private
	Preset      string                 `json:"preset,omitempty"`      // private_chat, public_chat, trusted_private_chat
	Invite      []string               `json:"invite,omitempty"`
	InitialState []interface{}          `json:"initial_state,omitempty"`
	CreationContent map[string]interface{} `json:"creation_content,omitempty"`
	IsDirect    bool                   `json:"is_direct,omitempty"`
	RoomAliasName string               `json:"room_alias_name,omitempty"`
}

// RoomCreateResponse represents a room creation response
type RoomCreateResponse struct {
	RoomID    string `json:"room_id"`
	RoomAlias string `json:"room_alias,omitempty"`
}

// MessageEvent represents a message event
type MessageEvent struct {
	MsgType string `json:"msgtype"`
	Body    string `json:"body"`
	// For formatted messages
	Format        string `json:"format,omitempty"`
	FormattedBody string `json:"formatted_body,omitempty"`
	// For media
	URL string `json:"url,omitempty"`
	// For replies
	InReplyTo *struct {
		EventID string `json:"event_id,omitempty"`
	} `json:"m.relates_to,omitempty"`
}

// CreateRoom creates a new Matrix room
func (c *Client) CreateRoom(ctx context.Context, req RoomCreateRequest, asUser string) (*RoomCreateResponse, error) {
	endpoint := fmt.Sprintf("%s/_matrix/client/v3/createRoom", c.homeserverURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq, asUser)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create room failed (%d): %s", resp.StatusCode, string(errBody))
	}

	var result RoomCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// SendMessage sends a message to a room
func (c *Client) SendMessage(ctx context.Context, roomID, eventID string, message MessageEvent, asUser string) (string, error) {
	// Generate transaction ID if not provided
	if eventID == "" {
		eventID = fmt.Sprintf("m%d", time.Now().UnixNano())
	}

	endpoint := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s",
		c.homeserverURL, url.PathEscape(roomID), url.PathEscape(eventID))

	body, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("marshal message: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq, asUser)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("send message failed (%d): %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		EventID string `json:"event_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.EventID, nil
}

// SendText sends a text message to a room
func (c *Client) SendText(ctx context.Context, roomID, text, asUser string) (string, error) {
	return c.SendMessage(ctx, roomID, "", MessageEvent{
		MsgType: "m.text",
		Body:    text,
	}, asUser)
}

// SendNotice sends a notice (bot message) to a room
func (c *Client) SendNotice(ctx context.Context, roomID, text, asUser string) (string, error) {
	return c.SendMessage(ctx, roomID, "", MessageEvent{
		MsgType: "m.notice",
		Body:    text,
	}, asUser)
}

// SendEmote sends an emote (action) message to a room
func (c *Client) SendEmote(ctx context.Context, roomID, text, asUser string) (string, error) {
	return c.SendMessage(ctx, roomID, "", MessageEvent{
		MsgType: "m.emote",
		Body:    text,
	}, asUser)
}

// InviteUser invites a user to a room
func (c *Client) InviteUser(ctx context.Context, roomID, userID, asUser string) error {
	endpoint := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/invite", c.homeserverURL, url.PathEscape(roomID))

	body, err := json.Marshal(map[string]string{
		"user_id": userID,
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq, asUser)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("invite failed (%d): %s", resp.StatusCode, string(errBody))
	}

	return nil
}

// JoinRoom joins a room (for ghost users)
func (c *Client) JoinRoom(ctx context.Context, roomIDOrAlias, asUser string) error {
	endpoint := fmt.Sprintf("%s/_matrix/client/v3/join/%s", c.homeserverURL, url.PathEscape(roomIDOrAlias))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq, asUser)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("join failed (%d): %s", resp.StatusCode, string(errBody))
	}

	return nil
}

// LeaveRoom leaves a room
func (c *Client) LeaveRoom(ctx context.Context, roomID, asUser string) error {
	endpoint := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/leave", c.homeserverURL, url.PathEscape(roomID))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq, asUser)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("leave failed (%d): %s", resp.StatusCode, string(errBody))
	}

	return nil
}

// SetDisplayName sets a user's display name
func (c *Client) SetDisplayName(ctx context.Context, displayName, asUser string) error {
	endpoint := fmt.Sprintf("%s/_matrix/client/v3/profile/%s/displayname",
		c.homeserverURL, url.PathEscape(asUser))

	body, err := json.Marshal(map[string]string{
		"displayname": displayName,
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq, asUser)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set displayname failed (%d): %s", resp.StatusCode, string(errBody))
	}

	return nil
}

// SetAvatarURL sets a user's avatar
func (c *Client) SetAvatarURL(ctx context.Context, avatarURL, asUser string) error {
	endpoint := fmt.Sprintf("%s/_matrix/client/v3/profile/%s/avatar_url",
		c.homeserverURL, url.PathEscape(asUser))

	body, err := json.Marshal(map[string]string{
		"avatar_url": avatarURL,
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq, asUser)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set avatar failed (%d): %s", resp.StatusCode, string(errBody))
	}

	return nil
}

// GetRoomState gets state from a room
func (c *Client) GetRoomState(ctx context.Context, roomID, eventType, stateKey, asUser string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/state/%s",
		c.homeserverURL, url.PathEscape(roomID), url.PathEscape(eventType))
	if stateKey != "" {
		endpoint += "/" + url.PathEscape(stateKey)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq, asUser)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get state failed (%d): %s", resp.StatusCode, string(errBody))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result, nil
}

// setHeaders sets common headers for requests
func (c *Client) setHeaders(req *http.Request, asUser string) {
	req.Header.Set("Authorization", "Bearer "+c.asToken)
	req.Header.Set("Content-Type", "application/json")

	// Set user to act as (ghost user)
	if asUser != "" {
		req.Header.Set("X-Matrix-User", asUser)
		// Some homeservers use query parameter
		q := req.URL.Query()
		q.Set("user_id", asUser)
		req.URL.RawQuery = q.Encode()
	}
}
