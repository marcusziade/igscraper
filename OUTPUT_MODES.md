# Instagram Scraper Output Modes

The scraper now supports multiple output modes to control verbosity:

## 1. Progress Mode (DEFAULT)
```bash
./igscraper username
```
Shows:
- Only the progress bar: `username [━━──────] 45/523 • 12.5/min • 25.3 MB • 2m15s`
- Final summary when complete
- Errors if they occur
- No logo, no logs, no other output

## 2. Verbose Mode (`--verbose` or `-v`)
```bash
./igscraper username --verbose
```
Shows:
- ASCII logo
- Target profile info
- All INFO/WARN/ERROR logs
- Progress bar with full details
- Download completion messages

## 3. Quiet Mode (`--quiet` or `-q`)
```bash
./igscraper username --quiet
```
Shows:
- Logger output only (INFO/WARN/ERROR)
- No UI elements (no logo, no progress bar)
- Good for logging to files

## 4. Silent Mode (`--log-level error`)
```bash
./igscraper username --log-level error
```
Shows:
- Only errors
- Completely silent otherwise
- Perfect for scripts and automation

## Examples

### Download with just progress bar
```bash
./igscraper johndoe --progress
```

### Resume download silently
```bash
./igscraper johndoe --resume --log-level error
```

### Debug mode with all details
```bash
./igscraper johndoe --log-level debug
```

### Quiet mode for cron jobs
```bash
./igscraper johndoe --quiet >> scraper.log 2>&1
```

## Environment Variables

You can also set quiet mode via environment variable:
```bash
export IGSCRAPER_QUIET=true
./igscraper johndoe
```