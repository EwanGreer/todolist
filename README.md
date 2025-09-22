# TodoList

A beautiful terminal-based todo list application built with Go, featuring integration with TaskWarrior and an interactive TUI powered by Bubble Tea.

## Features

- **Beautiful Terminal Interface**: Clean, colorful table-based UI with Catppuccin color scheme
- **TaskWarrior Integration**: Full compatibility with TaskWarrior for task persistence and management
- **Project Organization**: Support for organizing tasks into projects using TaskWarrior syntax
- **Interactive Navigation**: Vim-like keybindings for efficient task management
- **Search & Filter**: Real-time text search and project-based filtering
- **Task Status Management**: Toggle task completion status with visual indicators

## Installation

### Prerequisites

- Go 1.25.0 or later
- TaskWarrior (for task persistence)

### Build from Source

```bash
git clone https://github.com/EwanGreer/todolist
cd todolist
go build -o todolist
```

## Usage

### Starting the Application

```bash
./todolist
```

### Keyboard Controls

| Key | Action |
|-----|--------|
| `‘/“` or `j/k` | Navigate through tasks |
| `Space` or `Enter` | Toggle task completion |
| `a` | Add new task |
| `d` | Delete selected task |
| `f` | Open project filter menu |
| `F` | Cycle to previous project filter |
| `/` | Search tasks |
| `Esc` | Clear search or cancel current action |
| `q` or `Ctrl+C` | Quit application |

### Adding Tasks

When adding a new task, you can use TaskWarrior syntax:

- **Simple task**: `Fix the bug`
- **Task with project**: `Write tests project:myapp`

### Project Management

Tasks are organized into projects. Use the filter menu (`f`) to:
- View all tasks across projects
- Filter by specific project
- Navigate between different project views

### Search

Use `/` to search through task descriptions and project names in real-time.

## TaskWarrior Integration

This application uses TaskWarrior as its backend for task storage and management. Tasks are stored in `~/.task/` directory and are fully compatible with the TaskWarrior command-line tool.

### TaskWarrior Features Supported

- Task creation and modification
- Project assignment
- Task completion status
- Task deletion
- Timestamp tracking (creation, modification, completion)

## Architecture

- **Backend**: TaskWarrior integration via CLI commands
- **Frontend**: Bubble Tea TUI framework with Lipgloss styling
- **Data Structure**: In-memory todo representation with TaskWarrior synchronization
- **Navigation**: Table-based interface with cursor navigation

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions for terminal apps
- [Bubbles](https://github.com/charmbracelet/bubbles) - Common Bubble Tea components
- [Cobra](https://github.com/spf13/cobra) - CLI framework

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source. Please check the repository for license details.