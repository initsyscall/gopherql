# gopherQl

A gopher sized SQLite IDE for your small databases and learning. Single binary, no SQLite install needed — `modernc.org/sqlite` ships it pure Go.

## Stack

- **Go** — language
- **BubbleTea** — TUI framework
- **Lipgloss** — styling
- **modernc.org/sqlite** — pure-Go SQLite driver, no CGO

## Usage

### Installation

```bash
go install codeberg.org/initsyscall/gopherql@latest
```

### Run

```bash
go build -o gopherql .
./gopherql database_name.db
```

If the database doesn't exist it's created after a `y/N` confirmation. Command history is stored next to the `.db` file as `database_name_history.json`.

## Layout

- Left (70%) — query results
- Right (30%) — command history
- Bottom — prompt line

## Commands

Type `/` to preview commands, then pick as you type.

| Command | Action |
|---|---|
| `/quit` | exit |
| `/clear` | clear query pane |
| `/burn history` | wipe command history (double confirm) |
| `/burn db` | drop all tables (double confirm) |

## Keys

| Key | Action |
|---|---|
| `Enter` | run query / command |
| `↑` / `↓` | cycle command history |
| `Ctrl+j` / `Ctrl+k` | scroll history pane |
| `Ctrl+Shift+j` / `Ctrl+Shift+k` | scroll query pane vertically |
| `Ctrl+Shift+→` / `Ctrl+Shift+←` | scroll query pane horizontally |
| `Ctrl+c` | quit |
