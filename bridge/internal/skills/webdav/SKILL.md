---
name: webdav
description: Perform WebDAV protocol operations (list, get, put, delete) on WebDAV servers
homepage: https://tools.ietf.org/html/rfc4918
domain: webdav
risk: medium
approval_policy: auto
---

# WebDAV Skill

Perform WebDAV protocol operations on remote WebDAV servers.

## Operations

### List (PROPFIND)

List the contents of a WebDAV directory.

**Parameters:**
- `url` (required): The WebDAV server URL
- `operation`: "list"
- `username` (optional): Authentication username
- `password` (optional): Authentication password

**Example:**
```json
{
  "url": "https://dav.example.com/remote.php/webdav",
  "operation": "list"
}
```

**Response:**
```json
{
  "url": "https://dav.example.com/remote.php/webdav",
  "entries": [
    {
      "name": "documents",
      "is_directory": true
    },
    {
      "name": "notes.txt",
      "is_directory": false,
      "content_length": 1024,
      "content_type": "text/plain"
    }
  ],
  "total": 2,
  "success": true,
  "message": "Successfully listed 2 items"
}
```

### Get (GET)

Download content from a WebDAV resource.

**Parameters:**
- `url` (required): The WebDAV resource URL
- `operation`: "get"
- `username` (optional): Authentication username
- `password` (optional): Authentication password

**Example:**
```json
{
  "url": "https://dav.example.com/remote.php/webdav/notes.txt",
  "operation": "get"
}
```

**Response:**
```json
{
  "url": "https://dav.example.com/remote.php/webdav/notes.txt",
  "content": "Hello World",
  "size": 11,
  "success": true,
  "content_type": "text/plain",
  "metadata": {
    "url": "https://dav.example.com/remote.php/webdav/notes.txt",
    "size_bytes": 11,
    "content_type": "text/plain",
    "operation": "get"
  }
}
```

### Put (PUT)

Upload content to a WebDAV resource.

**Parameters:**
- `url` (required): The WebDAV target URL
- `operation`: "put"
- `content` (required): Content bytes to upload
- `content_length` (required): Length of content in bytes
- `content_type` (optional): Content type header
- `username` (optional): Authentication username
- `password` (optional): Authentication password

**Example:**
```json
{
  "url": "https://dav.example.com/remote.php/webdav/newfile.txt",
  "operation": "put",
  "content": [72, 101, 108, 108, 111],
  "content_length": 5,
  "content_type": "text/plain"
}
```

**Response:**
```json
{
  "url": "https://dav.example.com/remote.php/webdav/newfile.txt",
  "success": true,
  "message": "Successfully uploaded content",
  "new_url": "https://dav.example.com/remote.php/webdav/newfile.txt",
  "metadata": {
    "original_url": "https://dav.example.com/remote.php/webdav/newfile.txt",
    "new_url": "https://dav.example.com/remote.php/webdav/newfile.txt",
    "content_size": 5,
    "operation": "put"
  }
}
```

### Delete (DELETE)

Delete a WebDAV resource.

**Parameters:**
- `url` (required): The WebDAV resource URL
- `operation`: "delete"
- `username` (optional): Authentication username
- `password` (optional): Authentication password

**Example:**
```json
{
  "url": "https://dav.example.com/remote.php/webdav/oldfile.txt",
  "operation": "delete"
}
```

**Response:**
```json
{
  "url": "https://dav.example.com/remote.php/webdav/oldfile.txt",
  "success": true,
  "message": "Successfully deleted item",
  "metadata": {
    "url": "https://dav.example.com/remote.php/webdav/oldfile.txt",
    "operation": "delete"
  }
}
```

## Security Considerations

- **SSL/TLS Required**: Only HTTPS URLs are accepted
- **SSRF Protection**: All URLs are validated against private networks and cloud metadata endpoints
- **Authentication**: Basic authentication is supported for server authentication
- **Content Size Limit**: Maximum 100MB per operation

## Common Use Cases

1. **Cloud Storage Access**: Access WebDAV-based cloud storage (Nextcloud, ownCloud, etc.)
2. **Document Management**: Upload and retrieve documents from WebDAV servers
3. **Backup Operations**: Perform backups to WebDAV-compatible storage
4. **File Synchronization**: Synchronize files between servers
5. **WebDAV Publishing**: Manage content on WebDAV publishing systems

