# filetree

A CLI tool for organizing files using a tag-based system. `filetree` helps you categorize and manage your files by creating virtual organization structures through tags, without moving the original files.

## Features

- **Tag-based Organization** - Assign multiple tags to files for flexible categorization
- **Interactive Tagging** - Review and tag files one-by-one with preview capability
- **Smart Export** - Export tagged files as symlinks or copies into organized folder structures
- **Auto-refresh** - Automatically sync map files with actual directory contents
- **Nested Folders** - Support for folder hierarchies with nested map files

## Installation

### Build from Source

```bash
git clone https://github.com/<username>/filetree.git
cd filetree
go build -o filetree
```

### Requirements

- Go 1.24.3 or later
- Linux/Unix environment (for symlink support)

## Usage

### Initialize and Tag Files

Start by creating a `.map` file for your directory and interactively tag your files:

```bash
# Interactive tagging mode
./filetree interactive [path]

# Only show untagged files
./filetree interactive -u [path]
```

During interactive tagging:
- Press `p` to preview the file
- Enter comma-separated tags (e.g., `work, important, 2026`)
- Press `s` to skip
- Press `q` to quit

### Refresh Map Files

Sync map files with actual directory contents and remove stale entries:

```bash
./filetree refresh [path]
```

### Export Tagged Files

Organize files into tag-based folders:

```bash
# Export as symlinks (default)
./filetree export ./organized

# Export as copies
./filetree export -t copy ./organized
```

This creates a folder structure like:
```
organized/
├── work/
│   ├── document1.txt
│   └── report.pdf
├── important/
│   └── document1.txt
└── 2026/
    └── report.pdf
```

## Map File Format

`filetree` uses `.map` files (named after the directory, e.g., `documents.map`) to track files and their tags.

### Syntax

```
> filename.txt [tag1, tag2]
>> foldername [tag1]
> another.pdf [work]
```

- `>` - File entry
- `>>` - Folder entry (for nested organization)
- `[tags]` - Comma-separated tags in square brackets (optional)

### Example

```map
> resume.pdf [work, documents]
> photo.jpg [personal, memories]
>> archive [old]
> notes.txt [work, todo]
```

## Workflow

1. **Navigate** to the directory you want to organize
2. **Tag files** interactively: `./filetree interactive`
3. **Refresh** before exporting: `./filetree refresh`
4. **Export** to an organized folder: `./filetree export ./sorted`

## Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `interactive [path]` | `i` | Interactively tag files one by one |
| `refresh [path]` | `r` | Update all map files and cleanup deleted entries |
| `export [folder]` | `x` | Export tagged files to organized folders |

### Flags

- `-u, --untagged` - Only show files without tags (interactive mode)
- `-t, --type` - Export type: `copy` or `symlink` (default: symlink)

## Project Structure

```
filetree/
├── filetree.go      # Main application
├── go.mod           # Go module definition
├── README.md        # This file
└── test/            # Test directory
```

## License

This project is open source and available under the MIT License.

---

**Coded with Qwen** (Alibaba Cloud's AI Assistant)

*Built using Go and the [urfave/cli/v3](https://github.com/urfave/cli) library*
