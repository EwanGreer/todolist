package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/EwanGreer/todolist/taskwarrior"
)

type todo struct {
	uuid      string
	text      string
	project   string
	completed bool
	createdAt int64
}

type App struct {
	todos                []todo
	table                table.Model
	selected             map[int]struct{}
	currentFilter        string
	projects             []string
	searchMode           bool
	searchText           string
	projectSelectionMode bool
	projectCursor        int
	tw                   *taskwarrior.TaskWarrior
	width                int
	height               int
	addMode              bool
	addText              string
}

func sortTodosByCreatedAt(todos []todo) {
	sort.Slice(todos, func(i, j int) bool {
		// Most recent first (higher timestamp first)
		return todos[i].createdAt > todos[j].createdAt
	})
}

func parseTaskWarriorInput(input string) (description string, project string) {
	// Look for project: pattern
	projectRegex := regexp.MustCompile(`project:(\w+)`)
	projectMatch := projectRegex.FindStringSubmatch(input)

	if len(projectMatch) > 1 {
		project = projectMatch[1]
		// Remove the project: part from the description
		description = strings.TrimSpace(projectRegex.ReplaceAllString(input, ""))
	} else {
		description = strings.TrimSpace(input)
		project = "default"
	}

	return description, project
}

func NewApp() *App {
	tw, err := taskwarrior.New()
	if err != nil {
		fmt.Printf("Error initializing Taskwarrior: %v\n", err)
		os.Exit(1)
	}

	todos := loadTodosFromTaskwarrior(tw)

	projects := getUniqueProjects(todos)
	projects = append([]string{"all"}, projects...)

	columns := []table.Column{
		{Title: "Status", Width: 8},
		{Title: "Task", Width: 30},
		{Title: "Project", Width: 15},
	}

	rows := make([]table.Row, len(todos))
	for i, todo := range todos {
		status := "[ ]"
		if todo.completed {
			status = "[✓]"
		}
		rows[i] = table.Row{status, todo.text, todo.project}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#6c7086")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("#fab387")).
		PaddingLeft(1).
		PaddingRight(1)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#1e1e2e")).
		Background(lipgloss.Color("#f38ba8")).
		Bold(true)
	s.Cell = s.Cell.
		Foreground(lipgloss.Color("#cdd6f4")).
		PaddingLeft(1).
		PaddingRight(1)
	t.SetStyles(s)

	return &App{
		todos:                todos,
		table:                t,
		selected:             make(map[int]struct{}),
		currentFilter:        "all",
		projects:             projects,
		searchMode:           false,
		searchText:           "",
		projectSelectionMode: false,
		projectCursor:        0,
		tw:                   tw,
		width:                80,
		height:               24,
		addMode:              false,
		addText:              "",
	}
}

func loadTodosFromTaskwarrior(tw *taskwarrior.TaskWarrior) []todo {
	var todos []todo

	// Load pending tasks
	pendingTasks, err := tw.LoadPendingTasks()
	if err != nil {
		fmt.Printf("Warning: Could not load pending tasks: %v\n", err)
	} else {
		for _, task := range pendingTasks {
			project := task.Project
			if project == "" {
				project = "default"
			}
			todos = append(todos, todo{
				uuid:      task.UUID,
				text:      task.Description,
				project:   project,
				completed: false,
				createdAt: task.Entry,
			})
		}
	}

	// Load completed tasks
	completedTasks, err := tw.LoadCompletedTasks()
	if err != nil {
		fmt.Printf("Warning: Could not load completed tasks: %v\n", err)
	} else {
		for _, task := range completedTasks {
			project := task.Project
			if project == "" {
				project = "default"
			}
			todos = append(todos, todo{
				uuid:      task.UUID,
				text:      task.Description,
				project:   project,
				completed: true,
				createdAt: task.Entry,
			})
		}
	}

	// Sort the combined list by creation date (most recent first)
	sortTodosByCreatedAt(todos)

	return todos
}

func (m *App) saveTodoToTaskwarrior(t *todo) error {
	task := &taskwarrior.Task{
		UUID:        t.uuid,
		Description: t.text,
		Project:     t.project,
		Status:      "pending",
	}

	if t.completed {
		task.Status = "completed"
	}

	err := m.tw.SaveTask(task)
	if err == nil && t.uuid == "" {
		// Update the todo with the generated UUID
		t.uuid = task.UUID
	}
	return err
}

func (m *App) deleteTodoFromTaskwarrior(t todo) error {
	if t.uuid == "" {
		return nil // Can't delete without UUID
	}
	return m.tw.DeleteTask(t.uuid)
}

func (m *App) Init() tea.Cmd {
	return nil
}

func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.addMode {
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.addText) != "" {
					// Parse the input using TaskWarrior syntax
					description, project := parseTaskWarriorInput(m.addText)

					newTodo := todo{
						text:      description,
						completed: false,
						project:   project,
						createdAt: time.Now().Unix(),
					}

					// Save to TaskWarrior
					if err := m.saveTodoToTaskwarrior(&newTodo); err != nil {
						fmt.Printf("Error saving new task: %v\n", err)
					} else {
						// Reload todos from Taskwarrior to ensure consistency
						m.reloadTodos()
						m.updateTable()
					}
				}

				// Exit add mode
				m.addMode = false
				m.addText = ""
				return m, nil
			case "esc":
				m.addMode = false
				m.addText = ""
				return m, nil
			case "backspace":
				if len(m.addText) > 0 {
					m.addText = m.addText[:len(m.addText)-1]
				}
				return m, nil
			case "ctrl+c", "q":
				return m, tea.Quit
			default:
				if len(msg.String()) == 1 {
					m.addText += msg.String()
				}
				return m, nil
			}
		}

		if m.projectSelectionMode {
			switch msg.String() {
			case "up", "k":
				if m.projectCursor > 0 {
					m.projectCursor--
				}
				return m, nil
			case "down", "j":
				if m.projectCursor < len(m.projects)-1 {
					m.projectCursor++
				}
				return m, nil
			case "enter", " ":
				m.currentFilter = m.projects[m.projectCursor]
				m.projectSelectionMode = false
				m.updateTable()
				return m, nil
			case "esc":
				m.projectSelectionMode = false
				return m, nil
			case "ctrl+c", "q":
				return m, tea.Quit
			}
			return m, nil
		}

		if m.searchMode {
			switch msg.String() {
			case "enter", "esc":
				m.searchMode = false
				m.updateTable()
				return m, nil
			case "backspace":
				if len(m.searchText) > 0 {
					m.searchText = m.searchText[:len(m.searchText)-1]
					m.updateTable()
				}
				return m, nil
			case "ctrl+c", "q":
				return m, tea.Quit
			default:
				if len(msg.String()) == 1 {
					m.searchText += msg.String()
					m.updateTable()
				}
				return m, nil
			}
		}

		// Normal mode key handling
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "/":
			m.searchMode = true
			return m, nil
		case "esc":
			if m.searchText != "" {
				m.searchText = ""
				m.updateTable()
			}
			return m, nil
		case "enter", " ":
			filtered := m.getFilteredTodos()
			if len(filtered) > 0 {
				cursor := m.table.Cursor()
				if cursor < len(filtered) {
					targetTodo := filtered[cursor]
					for i := range m.todos {
						if m.todos[i].uuid == targetTodo.uuid || (m.todos[i].uuid == "" && m.todos[i].text == targetTodo.text && m.todos[i].project == targetTodo.project) {
							// Toggle completion status
							originalStatus := m.todos[i].completed
							m.todos[i].completed = !m.todos[i].completed

							// Save to Taskwarrior
							if err := m.saveTodoToTaskwarrior(&m.todos[i]); err != nil {
								fmt.Printf("Error saving task: %v\n", err)
								// Revert the change if save failed
								m.todos[i].completed = originalStatus
							}
							// Don't reload - just update the table with current state
							break
						}
					}
					m.updateTable()
				}
			}
		case "d":
			filtered := m.getFilteredTodos()
			if len(filtered) > 0 {
				cursor := m.table.Cursor()
				if cursor < len(filtered) {
					targetTodo := filtered[cursor]
					for i := range m.todos {
						if m.todos[i].uuid == targetTodo.uuid || (m.todos[i].uuid == "" && m.todos[i].text == targetTodo.text && m.todos[i].project == targetTodo.project) {
							// Delete from Taskwarrior
							if err := m.deleteTodoFromTaskwarrior(m.todos[i]); err != nil {
								fmt.Printf("Error deleting task: %v\n", err)
							} else {
								// Reload todos from Taskwarrior to ensure consistency
								m.reloadTodos()
							}
							break
						}
					}
					m.updateTable()
				}
			}
		case "a":
			m.addMode = true
			m.addText = ""
		case "f":
			for i, project := range m.projects {
				if project == m.currentFilter {
					m.projectCursor = i
					break
				}
			}
			m.projectSelectionMode = true
		case "F":
			m.prevFilter()
			m.updateTable()
		default:
			m.table, cmd = m.table.Update(msg)
		}
	}
	return m, cmd
}

func (m App) View() string {
	mainView := m.renderMainView()

	// Always center the main view
	centeredMainView := lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		mainView,
	)

	if m.projectSelectionMode {
		overlay := m.renderProjectSelection()

		// Place the overlay on top of the centered main view
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			overlay,
		)
	}

	if m.addMode {
		overlay := m.renderAddForm()

		// Place the overlay on top of the centered main view
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			overlay,
		)
	}

	return centeredMainView
}

func (m App) renderMainView() string {
	baseStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6c7086")).
		Padding(0, 1)

	filterInfo := fmt.Sprintf("Filter: %s", m.currentFilter)
	if m.currentFilter == "all" {
		filterInfo = "Filter: all projects"
	}

	searchInfo := ""
	if m.searchMode {
		searchInfo = fmt.Sprintf("Search: %s_", m.searchText)
	} else if m.searchText != "" {
		searchInfo = fmt.Sprintf("Search: %s", m.searchText)
	}

	headerInfo := filterInfo
	if searchInfo != "" {
		headerInfo += " • " + searchInfo
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fab387")).
		Bold(true).
		Margin(0, 0, 1, 0)

	helpText := "q: quit • ↑/↓: navigate • space/enter: toggle • a: add task • d: delete • f: filter • F: prev filter • /: search • esc: clear search"

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086")).
		Margin(1, 0)

	return headerStyle.Render(headerInfo) + "\n" +
		baseStyle.Render(m.table.View()) + "\n" +
		helpStyle.Render(helpText)
}

func (m *App) updateTable() {
	filtered := m.getFilteredTodos()

	// Sort filtered todos by creation date (most recent first)
	sortTodosByCreatedAt(filtered)

	rows := make([]table.Row, len(filtered))
	for i, todo := range filtered {
		status := "[ ]"
		if todo.completed {
			status = "[✓]"
		}
		rows[i] = table.Row{status, todo.text, todo.project}
	}

	// Preserve cursor position and focus state
	currentCursor := m.table.Cursor()
	m.table.SetRows(rows)

	// Always maintain focus
	m.table.Focus()

	// Ensure cursor is within bounds after updating rows
	if len(rows) > 0 {
		if currentCursor >= len(rows) {
			m.table.SetCursor(len(rows) - 1)
		} else if currentCursor < 0 {
			m.table.SetCursor(0)
		} else {
			m.table.SetCursor(currentCursor)
		}
	} else {
		// No rows, reset cursor
		m.table.SetCursor(0)
	}
}

func getUniqueProjects(todos []todo) []string {
	projectMap := make(map[string]bool)
	for _, todo := range todos {
		projectMap[todo.project] = true
	}

	var projects []string
	for project := range projectMap {
		projects = append(projects, project)
	}
	return projects
}

func (m *App) matchesSearch(todo todo) bool {
	if m.searchText == "" {
		return true
	}
	searchLower := strings.ToLower(m.searchText)
	textLower := strings.ToLower(todo.text)
	projectLower := strings.ToLower(todo.project)

	return strings.Contains(textLower, searchLower) || strings.Contains(projectLower, searchLower)
}

func (m *App) getFilteredTodos() []todo {
	var filtered []todo

	for _, todo := range m.todos {
		projectMatch := m.currentFilter == "all" || todo.project == m.currentFilter
		textMatch := m.matchesSearch(todo)

		if projectMatch && textMatch {
			filtered = append(filtered, todo)
		}
	}
	return filtered
}

func (m *App) updateProjects() {
	m.projects = getUniqueProjects(m.todos)
	m.projects = append([]string{"all"}, m.projects...)
}

func (m *App) reloadTodos() {
	m.todos = loadTodosFromTaskwarrior(m.tw)
	m.updateProjects()
}

func (m *App) nextFilter() {
	for i, project := range m.projects {
		if project == m.currentFilter {
			m.currentFilter = m.projects[(i+1)%len(m.projects)]
			break
		}
	}
}

func (m *App) prevFilter() {
	for i, project := range m.projects {
		if project == m.currentFilter {
			if i == 0 {
				m.currentFilter = m.projects[len(m.projects)-1]
			} else {
				m.currentFilter = m.projects[i-1]
			}
			break
		}
	}
}

func (m *App) renderProjectSelection() string {
	var items []string
	for i, project := range m.projects {
		cursor := "  "
		if i == m.projectCursor {
			cursor = "❯ "
		}

		selected := " "
		if project == m.currentFilter {
			selected = "✓"
		}

		displayName := project
		if project == "all" {
			displayName = "all projects"
		}

		line := fmt.Sprintf("%s[%s] %s", cursor, selected, displayName)
		if i == m.projectCursor {
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#1e1e2e")).
				Background(lipgloss.Color("#f38ba8")).
				Bold(true).
				Render(line)
		}

		items = append(items, line)
	}

	content := strings.Join(items, "\n")

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fab387")).
		Bold(true).
		Render("Select Project Filter:")

	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086")).
		Render("Press enter to select, esc to cancel")

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6c7086")).
		Padding(1, 2).
		Width(35)

	return style.Render(title + "\n\n" + content + "\n\n" + instructions)
}

func (m *App) renderAddForm() string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fab387")).
		Bold(true).
		Render("Add New Task")

	// Input field
	inputStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#fab387")).
		Padding(0, 1).
		Width(50)

	inputContent := m.addText + "_"
	inputField := inputStyle.Render(inputContent)

	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086")).
		Render("Enter task description (use project:name for projects) • enter to save • esc to cancel")

	// Examples
	examples := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086")).
		Italic(true).
		Render("Examples: \"Fix the bug\" or \"Write tests project:myapp\"")

	// Main container
	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6c7086")).
		Padding(1, 2).
		Width(60)

	return style.Render(title + "\n\n" + inputField + "\n\n" + instructions + "\n\n" + examples)
}

var rootCmd = &cobra.Command{
	Use:   "todolist",
	Short: "A beautiful terminal-based todo list application",
	Long: `A beautiful and interactive terminal-based todo list application built with bubbletea.
Features include project filtering, text search, and an intuitive table interface.`,
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := tea.NewProgram(NewApp(), tea.WithAltScreen()).Run(); err != nil {
			fmt.Printf("Error: %v", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
