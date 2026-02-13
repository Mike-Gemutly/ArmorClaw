# WebSocket Client Guide for ArmorClaw Event Bus

> **Last Updated:** 2026-02-08
> **Version:** 1.0.0
> **Status:** Production Ready

## Overview

The ArmorClaw Event Bus provides real-time Matrix event distribution via WebSocket. This guide explains how to connect to the event bus, subscribe to filtered events, and handle real-time event streams.

## What is the Event Bus?

The Event Bus is a publish-subscribe system that pushes Matrix events to subscribed clients in real-time, eliminating the need for polling. This provides:

- **Real-time event delivery** - No polling delay
- **Reduced bandwidth** - Only relevant events are delivered
- **Flexible filtering** - Subscribe by room, sender, or event type
- **WebSocket transport** - Standard WebSocket protocol (RFC 6455)
- **Automatic cleanup** - Inactive subscribers are disconnected

## Architecture

```
Matrix Homeserver
        ↓
  Bridge (Matrix Adapter)
        ↓
   Event Bus (Publish)
        ↓
   WebSocket Server
        ↓
  WebSocket Clients
   (Subscribers)
```

## Configuration

Enable the event bus WebSocket server in your `config.toml`:

```toml
[eventbus]
# Enable WebSocket server for event push
websocket_enabled = true

# WebSocket listen address
websocket_addr = "0.0.0.0:8444"

# WebSocket path for event subscriptions
websocket_path = "/events"

# Maximum concurrent subscribers
max_subscribers = 100

# Inactivity timeout for subscribers
inactivity_timeout = "30m"
```

**Restart the bridge** after enabling the event bus.

## WebSocket Endpoint

**URL:** `ws://localhost:8444/events` (or `wss://` if TLS is configured)

**Connection:**
```bash
# Using websocat (recommended for testing)
websocat ws://localhost:8444/events

# Using wscat
wscat -c ws://localhost:8444/events

# Using Python websockets
python -m websockets ws://localhost:8444/events
```

## WebSocket Protocol

### Message Format

All WebSocket messages are JSON-encoded:

```json
{
  "type": "message_type",
  "data": { ... }
}
```

### Message Types

#### 1. Client → Server: Subscribe

Subscribe to filtered events:

```json
{
  "type": "subscribe",
  "data": {
    "filter": {
      "room_id": "!roomid:example.com",      // Optional: specific room
      "sender_id": "@user:example.com",       // Optional: specific user
      "event_types": [                       // Optional: event types
        "m.room.message",
        "m.room.member"
      ]
    }
  }
}
```

**Response:**

```json
{
  "type": "subscribed",
  "data": {
    "subscriber_id": "sub-1736294400000000000",
    "message": "Subscribed successfully"
  }
}
```

#### 2. Server → Client: Event

Received Matrix events:

```json
{
  "type": "event",
  "data": {
    "event": {
      "type": "m.room.message",
      "room_id": "!roomid:example.com",
      "sender": "@user:example.com",
      "content": {
        "msgtype": "m.text",
        "body": "Hello, world!"
      },
      "event_id": "$event_id:example.com"
    },
    "received": "2026-02-08T12:34:56Z",
    "sequence": 1736294096000000000
  }
}
```

#### 3. Client → Server: Unsubscribe

Unsubscribe from events:

```json
{
  "type": "unsubscribe",
  "data": {
    "subscriber_id": "sub-1736294400000000000"
  }
}
```

**Response:**

```json
{
  "type": "unsubscribed",
  "data": {
    "message": "Unsubscribed successfully"
  }
}
```

#### 4. Client → Server: Ping

Keep-alive ping:

```json
{
  "type": "ping"
}
```

**Response:**

```json
{
  "type": "pong"
}
```

#### 5. Server → Client: Error

Error messages:

```json
{
  "type": "error",
  "data": {
    "code": "invalid_filter",
    "message": "Invalid filter parameters"
  }
}
```

## Event Filtering

### Filter Options

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `room_id` | string | Only events from this room | `"!roomid:example.com"` |
| `sender_id` | string | Only events from this sender | `"@user:example.com"` |
| `event_types` | array | Only these event types | `["m.room.message", "m.room.member"]` |

### Filter Behavior

- **Empty filter** (`{}`): Receive all events
- **Partial filter**: Only specified fields are filtered
- **Multiple filters**: All filters must match (AND logic)

### Examples

#### Subscribe to all events:

```json
{
  "type": "subscribe",
  "data": {
    "filter": {}
  }
}
```

#### Subscribe to specific room:

```json
{
  "type": "subscribe",
  "data": {
    "filter": {
      "room_id": "!myroom:example.com"
    }
  }
}
```

#### Subscribe to specific user:

```json
{
  "type": "subscribe",
  "data": {
    "filter": {
      "sender_id": "@alice:example.com"
    }
  }
}
```

#### Subscribe to specific event types:

```json
{
  "type": "subscribe",
  "data": {
    "filter": {
      "event_types": ["m.room.message", "m.room.member"]
    }
  }
}
```

#### Combine multiple filters:

```json
{
  "type": "subscribe",
  "data": {
    "filter": {
      "room_id": "!myroom:example.com",
      "event_types": ["m.room.message"]
    }
  }
}
```

## Client Examples

### Python Client

```python
import asyncio
import json
import websockets
from datetime import datetime

async def subscribe_to_events():
    uri = "ws://localhost:8444/events"

    async with websockets.connect(uri) as websocket:
        # Subscribe to events from a specific room
        subscribe_msg = {
            "type": "subscribe",
            "data": {
                "filter": {
                    "room_id": "!myroom:example.com",
                    "event_types": ["m.room.message"]
                }
            }
        }

        await websocket.send(json.dumps(subscribe_msg))
        print(f"Subscribed: {await websocket.recv()}")

        # Receive events
        while True:
            try:
                message = await websocket.recv()
                data = json.loads(message)

                if data["type"] == "event":
                    event = data["data"]["event"]
                    received = data["data"]["received"]

                    print(f"[{received}] {event['sender']}: {event['content'].get('body', '')}")

                elif data["type"] == "error":
                    print(f"Error: {data['data']['message']}")
                    break

            except websockets.exceptions.ConnectionClosed:
                print("Connection closed")
                break

asyncio.run(subscribe_to_events())
```

### JavaScript Client (Browser)

```javascript
const ws = new WebSocket('ws://localhost:8444/events');

ws.onopen = () => {
  // Subscribe to all events
  ws.send(JSON.stringify({
    type: 'subscribe',
    data: {
      filter: {
        event_types: ['m.room.message']
      }
    }
  }));
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);

  switch (message.type) {
    case 'subscribed':
      console.log('Subscribed:', message.data.subscriber_id);
      break;

    case 'event':
      const eventData = message.data;
      console.log('Event received:', {
        type: eventData.event.type,
        sender: eventData.event.sender,
        room: eventData.event.room_id,
        content: eventData.event.content,
        received: eventData.received
      });
      break;

    case 'error':
      console.error('Error:', message.data.message);
      break;
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Connection closed');
};

// Unsubscribe before closing
function unsubscribe() {
  ws.send(JSON.stringify({
    type: 'unsubscribe',
    data: {
      subscriber_id: 'your-subscriber-id'
    }
  }));
}
```

### Go Client

```go
package main

import (
    "encoding/json"
    "log"
    "net/http"

    "github.com/gorilla/websocket"
)

type WSMessage struct {
    Type string                 `json:"type"`
    Data map[string]interface{} `json:"data"`
}

func main() {
    url := "ws://localhost:8444/events"

    conn, _, err := websocket.DefaultDialer.Dial(url, nil)
    if err != nil {
        log.Fatal("Dial error:", err)
    }
    defer conn.Close()

    // Subscribe to events
    subscribe := WSMessage{
        Type: "subscribe",
        Data: map[string]interface{}{
            "filter": map[string]interface{}{
                "room_id": "!myroom:example.com",
            },
        },
    }

    if err := conn.WriteJSON(subscribe); err != nil {
        log.Fatal("Write error:", err)
    }

    // Receive events
    for {
        var message WSMessage
        if err := conn.ReadJSON(&message); err != nil {
            log.Println("Read error:", err)
            break
        }

        switch message.Type {
        case "subscribed":
            log.Printf("Subscribed: %v\n", message.Data)
        case "event":
            event := message.Data["event"].(map[string]interface{})
            log.Printf("Event: %s from %s\n", event["type"], event["sender"])
        case "error":
            log.Printf("Error: %v\n", message.Data["message"])
        }
    }
}
```

### Bash Client (websocat)

```bash
#!/bin/bash
# Subscribe to events from a specific room

websocat ws://localhost:8444/events <<EOF
{"type":"subscribe","data":{"filter":{"room_id":"!myroom:example.com"}}}
EOF

# Or use interactive mode
echo '{"type":"subscribe","data":{"filter":{"room_id":"!myroom:example.com"}}}' | \
  websocat -n - ws://localhost:8444/events
```

## Inactivity Handling

The event bus automatically disconnects inactive subscribers:

- **Default timeout:** 30 minutes
- **Keep-alive:** Send ping messages every 30 seconds
- **Reconnection:** Implement automatic reconnection in clients

### Example: Keep-alive in Python

```python
import asyncio

async def keep_alive(websocket):
    while True:
        try:
            await websocket.send('{"type":"ping"}')
            await asyncio.sleep(30)
        except:
            break
```

## Error Handling

### Common Errors

| Error Code | Description | Solution |
|------------|-------------|----------|
| `invalid_filter` | Invalid filter parameters | Check filter syntax |
| `max_subscribers` | Too many subscribers | Increase `max_subscribers` or wait |
| `unauthorized` | Authentication required | Check WebSocket headers |
| `server_error` | Internal server error | Check bridge logs |

### Reconnection Strategy

```python
async def connect_with_retry():
    while True:
        try:
            await subscribe_to_events()
        except (ConnectionRefused, ConnectionClosed):
            print("Connection lost, retrying in 5 seconds...")
            await asyncio.sleep(5)
```

## Security Considerations

1. **TLS Encryption:** Use `wss://` in production
2. **Authentication:** Implement authentication headers if needed
3. **Filter Validation:** Server validates all filters
4. **Rate Limiting:** Server enforces subscription limits
5. **PII Scrubbing:** Events are automatically scrubbed of PII

## Testing

### Manual Testing

```bash
# 1. Start the bridge with event bus enabled
./bridge/build/armorclaw-bridge --config config.toml

# 2. Connect with websocat
websocat ws://localhost:8444/events

# 3. Send subscription message
{"type":"subscribe","data":{"filter":{"room_id":"!testroom:example.com"}}}

# 4. Observe events in real-time
```

### Automated Testing

```bash
# Run event bus filtering tests
./tests/test-eventbus-filtering.sh
```

## Performance

### Benchmarks

- **Event latency:** < 10ms from Matrix to WebSocket
- **Throughput:** 1000+ events/second
- **Concurrent clients:** 100+ simultaneous connections

### Optimization Tips

1. **Use specific filters** to reduce bandwidth
2. **Implement client-side buffering** for burst events
3. **Use JSON streaming** for large event batches
4. **Monitor inactivity** to avoid unnecessary reconnections

## Troubleshooting

### Connection Issues

**Problem:** Cannot connect to WebSocket server

**Solutions:**
1. Verify event bus is enabled in config
2. Check firewall allows port 8444
3. Verify bridge is running: `./bridge/build/armorclaw-bridge status`

### No Events Received

**Problem:** Connected but no events

**Solutions:**
1. Check Matrix adapter is connected
2. Verify filter matches events
3. Check bridge logs for errors
4. Test with empty filter (all events)

### Frequent Disconnections

**Problem:** Connection drops repeatedly

**Solutions:**
1. Implement keep-alive ping (every 30s)
2. Check network stability
3. Increase `inactivity_timeout` in config
4. Implement automatic reconnection

## Advanced Usage

### Multiple Subscriptions

A single WebSocket connection can have multiple subscriptions by sending multiple subscribe messages with different filters.

### Event Aggregation

Aggregate events from multiple rooms by not specifying `room_id` in the filter.

### Real-time Analytics

Use the event stream for real-time analytics, monitoring, or dashboard updates.

## Next Steps

1. **Enable the event bus** in your configuration
2. **Test with websocat** to verify connectivity
3. **Implement your client** using the examples above
4. **Configure filters** based on your needs
5. **Monitor performance** and adjust timeouts as needed

## Additional Resources

- **Configuration Guide:** [configuration.md](configuration.md)
- **RPC API Reference:** [rpc-api.md](rpc-api.md)
- **Communication Flow:** [communication-flow-analysis.md](../output/communication-flow-analysis.md)
- **Error Catalog:** [error-catalog.md](error-catalog.md)

---

**Document Version:** 1.0.0
**Last Updated:** 2026-02-08
**Status:** Production Ready
