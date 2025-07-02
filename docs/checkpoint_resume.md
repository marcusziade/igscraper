# Checkpoint and Resume Capability

The Instagram Scraper now supports checkpoint-based resume functionality, allowing you to continue interrupted downloads without starting from scratch.

## Overview

The checkpoint system automatically tracks download progress and can resume from where it left off in case of:
- Network interruptions
- Rate limit cooldowns
- Manual stops (Ctrl+C)
- System crashes
- Power outages

## How It Works

1. **Automatic Checkpoint Creation**: When a download starts, a checkpoint file is created to track progress
2. **Progress Tracking**: The system records:
   - Last processed page/cursor position
   - Successfully downloaded photos
   - Overall progress statistics
3. **Resume Detection**: When running the scraper again, it detects existing checkpoints and can resume
4. **Completion Cleanup**: Checkpoints are automatically deleted after successful completion

## Usage

### Resume an Interrupted Download

If a download was interrupted, simply add the `--resume` flag:

```bash
igscraper scrape username --resume
```

### Force Restart

To ignore an existing checkpoint and start fresh:

```bash
igscraper scrape username --force-restart
```

### Check for Existing Checkpoint

If you run the scraper without flags and a checkpoint exists, you'll see:

```
[WARNING] Existing checkpoint found. Use --resume to continue or --force-restart to start over.
[INFO] Checkpoint info: Downloaded: 150 photos, Last updated: 5m30s ago
```

## Checkpoint Storage Locations

Checkpoints are stored in platform-specific data directories:

- **Linux**: `~/.local/share/igscraper/checkpoints/`
- **macOS**: `~/Library/Application Support/igscraper/checkpoints/`
- **Windows**: `%APPDATA%/igscraper/checkpoints/`

Checkpoint files are named: `{username}.checkpoint.json`

## Features

### Atomic Writes
Checkpoints are written atomically to prevent corruption during system failures.

### Duplicate Prevention
The system tracks downloaded photos to avoid re-downloading when resuming.

### Progress Preservation
All progress counters and statistics are preserved across resume operations.

### Automatic Cleanup
Checkpoints are automatically deleted after successful completion to avoid clutter.

## Example Scenarios

### Scenario 1: Rate Limit Hit

```bash
# Start download
$ igscraper scrape photographer_jane

[INITIATING EXTRACTION SEQUENCE]
[EXTRACTED] Total: 500 | Batch: [████████████████████] 100/100
[COOLING DOWN FOR 1 HOUR]

# User stops with Ctrl+C
^C

# Resume after cooldown period
$ igscraper scrape photographer_jane --resume
[INFO] Resuming from checkpoint: Downloaded: 500 photos
[INITIATING EXTRACTION SEQUENCE]
[EXTRACTED] Total: 750 | Batch: [████████░░░░░░░░░░░░] 50/100
```

### Scenario 2: Network Interruption

```bash
# Download interrupted by network issue
$ igscraper scrape travel_blogger
[ERROR] Network timeout

# Resume when network is restored
$ igscraper scrape travel_blogger --resume
[INFO] Resuming from checkpoint: Downloaded: 1250 photos
```

### Scenario 3: Starting Fresh

```bash
# Previous incomplete download exists
$ igscraper scrape artist_profile
[WARNING] Existing checkpoint found. Use --resume to continue or --force-restart to start over.

# Force a fresh start
$ igscraper scrape artist_profile --force-restart
[INFO] Force restart: Ignoring existing checkpoint
[INITIATING EXTRACTION SEQUENCE]
```

## Technical Details

### Checkpoint Structure

```json
{
  "username": "photographer_jane",
  "user_id": "123456789",
  "last_processed_page": 5,
  "end_cursor": "QVFBc2R...",
  "downloaded_photos": {
    "ABC123": "ABC123.jpg",
    "DEF456": "DEF456.jpg"
  },
  "total_queued": 750,
  "total_downloaded": 500,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T11:45:30Z",
  "version": 1
}
```

### Checkpoint Update Frequency

- After each batch of photos is processed
- After each successful photo download
- Before entering rate limit cooldown

### Error Handling

- Corrupted checkpoints are backed up and a fresh start is initiated
- Missing checkpoint files are handled gracefully
- File system errors are logged but don't stop the download

## Best Practices

1. **Let Downloads Complete**: When possible, allow downloads to complete naturally for automatic cleanup
2. **Use Resume for Large Profiles**: For profiles with thousands of photos, resume capability ensures progress isn't lost
3. **Monitor Checkpoint Age**: Very old checkpoints might indicate issues - consider using `--force-restart`
4. **Check Disk Space**: Ensure sufficient disk space for both photos and checkpoint files

## Troubleshooting

### "Failed to load checkpoint"
- Check file permissions in the checkpoint directory
- Ensure the checkpoint file isn't corrupted
- Try `--force-restart` to start fresh

### "Checkpoint exists but resume not requested"
- Use `--resume` to continue from checkpoint
- Use `--force-restart` to ignore checkpoint and start over

### Resume seems to re-download photos
- This might happen if photos were deleted from the output directory
- The checkpoint tracks what was downloaded, not what currently exists
- Use `--force-restart` for a complete fresh start