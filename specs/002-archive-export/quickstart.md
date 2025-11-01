# Quickstart Guide: Exporting Your Archive

**Feature**: Archive Export
**Audience**: End Users

## Overview

The export feature allows you to download your complete Bluesky archive in JSON or CSV format with all associated media files. Exports are saved locally in the `./exports/` directory.

## Basic Export

### 1. Navigate to Export Page

- Log in to your local bskyarchive instance
- Click "Export" in the navigation menu

### 2. Choose Export Format

**JSON Export** (recommended for backup):
- Preserves complete post metadata
- Includes all fields (engagement metrics, embed data, labels)
- Best for programmatic access or migration

**CSV Export** (recommended for analysis):
- Spreadsheet-compatible format
- Opens directly in Excel or Google Sheets
- Ideal for analyzing posting patterns, engagement trends

### 3. Configure Options

- **Include Media**: Check to copy image/video files to export directory (default: checked)
- **Date Range** (optional): Filter posts by creation date
  - Leave blank to export all posts
  - Specify start/end dates for targeted export

### 4. Start Export

- Click "Start Export" button
- Progress indicator shows posts processed and media copied
- Export completes in under 10 seconds for typical archives (1000 posts)

### 5. Download Export

- When complete, click "Download" or navigate to `./exports/{timestamp}/` directory
- Export directory contains:
  - `posts.json` or `posts.csv` - Your post data
  - `manifest.json` - Export metadata
  - `media/` - All media files (if included)

## Export Directory Structure

```
exports/
└── 2025-10-30_14-30-45/          # Timestamp when export was created
    ├── manifest.json              # What's in this export
    ├── posts.json                 # Your posts (JSON format)
    └── media/                     # Media files
        ├── abc123def456...jpg     # Original images/videos
        └── ...
```

## Using Exported Data

### JSON Export

**View in browser**:
```bash
python3 -m json.tool posts.json | less
```

**Parse programmatically**:
```python
import json
with open('posts.json') as f:
    posts = json.load(f)
print(f"Total posts: {len(posts)}")
```

### CSV Export

**Open in Excel/Google Sheets**:
- Double-click `posts.csv`
- Opens with proper UTF-8 encoding (emoji/Unicode preserved)

**Analyze with command line**:
```bash
# Count posts by month
cut -d',' -f5 posts.csv | cut -d'T' -f1 | sort | uniq -c

# Find posts with most likes
sort -t',' -k7 -nr posts.csv | head -10
```

## Advanced: Date Range Filtering

Export only posts from specific time periods:

1. **Last Year's Posts**:
   - Start: `2024-01-01`
   - End: `2024-12-31`

2. **Recent Posts** (last 6 months):
   - Start: `2024-05-01`
   - End: Leave blank (defaults to today)

3. **Historical Posts** (before 2024):
   - Start: Leave blank (defaults to earliest post)
   - End: `2023-12-31`

## Troubleshooting

### "Insufficient disk space" error

Exports require roughly 2x your current archive size. Free up disk space or export without media:
- Uncheck "Include Media"
- Or use date range filter to export smaller subset

### "No posts match criteria" error

Your date range doesn't match any posts:
- Check start date is before end date
- Verify posts exist in that time period (check Browse page)

### CSV opens with garbled characters

Excel on Windows may need manual encoding selection:
1. Open Excel
2. File → Import → CSV file
3. Select "UTF-8" encoding
4. Import

Alternative: Use Google Sheets (handles UTF-8 automatically)

### Media files not in export

- Verify "Include Media" was checked
- Some posts may reference media that wasn't downloaded during archival
- Check `manifest.json` for actual media count

## Tips

- **Backup Strategy**: Export to JSON monthly, store in cloud backup
- **Analysis Workflow**: Export to CSV, analyze in spreadsheet, re-export filtered results
- **Disk Space**: Exports are cumulative - old exports aren't deleted automatically
- **Multiple Exports**: Each export gets unique timestamp directory - no overwriting

## Need Help?

- Check export manifest.json for details about what was exported
- Review application logs for detailed error messages
- Verify archive database is intact (browse posts still works)
