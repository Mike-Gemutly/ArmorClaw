# File Preview Screen

> **Route:** `file/{fileId}`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/media/FilePreviewScreen.kt`
> **Category:** Media

## Screenshot

![File Preview Screen](../../screenshots/media/file-preview.png)

## Layout

```
┌─────────────────────────────────────┐
│ ←  File Preview                     │  ← TopAppBar
├─────────────────────────────────────┤
│                                     │
│              📄                     │  ← File icon
│                                     │
│         document.pdf                │  ← File name
│         2.4 MB                      │  ← File size
│         PDF Document                │  ← File type
│                                     │
│  ┌─────────────────────────────┐   │
│  │ File Details                │   │
│  │ • Sender: John Doe          │   │
│  │ • Received: Today, 2:30 PM  │   │
│  │ • Encryption: ✅ Verified   │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ 📥 Download File            │   │  ← Download button
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ 👁️ Open With...             │   │  ← Open button
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ 📤 Share File               │   │  ← Share button
│  └─────────────────────────────┘   │
│                                     │
└─────────────────────────────────────┘
```

## UI States

### Loading

```
┌─────────────────────────────────────┐
│ ←  File Preview                     │
├─────────────────────────────────────┤
│                                     │
│           ◠ ◠ ◠                     │
│        Loading file info...         │
│                                     │
└─────────────────────────────────────┘
```

### Loaded (Default)

```
┌─────────────────────────────────────┐
│ ←  File Preview                     │
├─────────────────────────────────────┤
│  [File icon]                        │
│  [File name, size, type]            │
│  [File details]                     │
│                                     │
│  [Download] [Open] [Share]          │
└─────────────────────────────────────┘
```

### Downloading

```
┌─────────────────────────────────────┐
│ ←  File Preview                     │
├─────────────────────────────────────┤
│                                     │
│  ┌─────────────────────────────┐   │
│  │ Downloading...              │   │
│  │ ████████░░░░░░░░  45%       │   │  ← Progress bar
│  └─────────────────────────────┘   │
│                                     │
│         [Cancel]                    │
└─────────────────────────────────────┘
```

### Download Complete

```
┌─────────────────────────────────────┐
│ ←  File Preview                     │
├─────────────────────────────────────┤
│                                     │
│              ✅                     │
│        Download Complete            │
│                                     │
│  File saved to Downloads folder     │
│                                     │
│         [Open File]                 │
└─────────────────────────────────────┘
```

### Error

```
┌─────────────────────────────────────┐
│ ←  File Preview                     │
├─────────────────────────────────────┤
│              ⚠️                     │
│     Failed to load file             │
│                                     │
│         [Retry]                     │
└─────────────────────────────────────┘
```

## File Type Icons

| Type | Icon |
|------|------|
| PDF | 📄 |
| Image | 🖼️ |
| Video | 🎬 |
| Audio | 🎵 |
| Archive | 📦 |
| Document | 📝 |
| Unknown | 📎 |

## State Flow

```
            ┌─────────────┐
            │   Loading   │
            └──────┬──────┘
                   │
        ┌──────────┼──────────┐
        ▼          ▼          ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐
   │ Loaded  │ │  Error  │ │ Back    │
   └────┬────┘ └────┬────┘ └─────────┘
        │           │
   ┌────┴────┐      ▼
   ▼    ▼    ▼  ┌─────────┐
┌─────┐┌─────┐┌─────┐│ Retry   │
│DL   ││Open ││Share│└─────────┘
└──┬──┘└─────┘└─────┘
   │
   ▼
┌──────────┐
│Download  │
│Complete  │
└──────────┘
```

## User Flow

1. **User arrives from:** Chat screen (tap file)
2. **User can:**
   - View file metadata
   - Download to device
   - Open with external app
   - Share to other apps
   - View encryption status
3. **User navigates to:**
   - Chat screen (back)
   - External app (open)

## Accessibility

- **Content descriptions:**
  - File icon: "[file type] file"
  - Download: "Download file"
  - Open: "Open file with"
  - Share: "Share file"

- **Touch targets:**
  - All buttons: 48.dp minimum

## Design Tokens

| Token | Value |
|-------|-------|
| File icon size | 80.dp |
| Button width | Full width |
| Details card | surfaceVariant |

## Notes

- Shows file metadata before download
- Encryption status clearly indicated
- Supports all common file types
- Download progress visible
- External app integration
