package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type todo struct {
	text      string
	project   string
	completed bool
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
}

func NewApp() *App {
	todos := []todo{
		{text: "Buy groceries", completed: false, project: "life"},
		{text: "Walk the dog", completed: false, project: "life"},
		{text: "Write some code", completed: false, project: "programming"},
	}

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
			status = "[x]"
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
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
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
	}
}

func (m *App) Init() tea.Cmd {
	return nil
}

func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
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
						if m.todos[i].text == targetTodo.text && m.todos[i].project == targetTodo.project {
							m.todos[i].completed = !m.todos[i].completed
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
						if m.todos[i].text == targetTodo.text && m.todos[i].project == targetTodo.project {
							m.todos = append(m.todos[:i], m.todos[i+1:]...)
							break
						}
					}
					m.updateProjects()
					m.updateTable()
				}
			}
		case "a":
			newTodo := todo{text: "New todo item", completed: false, project: "default"}
			m.todos = append(m.todos, newTodo)
			m.updateProjects()
			m.updateTable()
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
	if m.projectSelectionMode {
		baseView := m.renderMainView()
		overlay := m.renderProjectSelection()

		return lipgloss.Place(
			lipgloss.Width(baseView), lipgloss.Height(baseView),
			lipgloss.Center, lipgloss.Center,
			overlay,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("238")),
		)
	}

	return m.renderMainView()
}

func (m App) renderMainView() string {
	baseStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

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

	helpText := "q: quit • ↑/↓: navigate • space/enter: toggle • a: add • d: delete • f: filter • F: prev filter • /: search • esc: clear search"

	return lipgloss.NewStyle().Margin(0, 0, 1, 0).Render(headerInfo) + "\n" +
		baseStyle.Render(m.table.View()) + "\n" +
		lipgloss.NewStyle().Margin(1, 0).Render(helpText)
}

func (m *App) updateTable() {
	filtered := m.getFilteredTodos()
	rows := make([]table.Row, len(filtered))
	for i, todo := range filtered {
		status := "[ ]"
		if todo.completed {
			status = "[x]"
		}
		rows[i] = table.Row{status, todo.text, todo.project}
	}
	m.table.SetRows(rows)
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
			cursor = "> "
		}

		selected := " "
		if project == m.currentFilter {
			selected = "✓"
		}

		displayName := project
		if project == "all" {
			displayName = "all projects"
		}

		items = append(items, fmt.Sprintf("%s[%s] %s", cursor, selected, displayName))
	}

	content := strings.Join(items, "\n")

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(30)

	return style.Render("Select Project Filter:\n\n" + content + "\n\nPress enter to select, esc to cancel")
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