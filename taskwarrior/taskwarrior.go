package taskwarrior

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"
)

type Task struct {
	UUID        string
	Description string
	Project     string
	Status      string
	Entry       int64
	Modified    int64
	End         int64
}

type TaskWarrior struct {
	dataDir string
}

func New() (*TaskWarrior, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dataDir := filepath.Join(homeDir, ".task")

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}

	return &TaskWarrior{dataDir: dataDir}, nil
}

func (tw *TaskWarrior) LoadPendingTasks() ([]*Task, error) {
	return tw.loadTasksFromCommand("status:pending")
}

func (tw *TaskWarrior) LoadCompletedTasks() ([]*Task, error) {
	return tw.loadTasksFromCommand("status:completed")
}

func (tw *TaskWarrior) loadTasksFromCommand(filter string) ([]*Task, error) {
	cmd := exec.Command("task", "rc.data.location="+tw.dataDir, filter, "export")
	output, err := cmd.Output()
	if err != nil {
		return []*Task{}, nil
	}

	var taskData []map[string]any
	if err := json.Unmarshal(output, &taskData); err != nil {
		return nil, err
	}

	var tasks []*Task
	for _, data := range taskData {
		task := &Task{}

		if uuid, ok := data["uuid"].(string); ok {
			task.UUID = uuid
		}
		if desc, ok := data["description"].(string); ok {
			task.Description = desc
		}
		if project, ok := data["project"].(string); ok {
			task.Project = project
		}
		if status, ok := data["status"].(string); ok {
			task.Status = status
		}
		if entry, ok := data["entry"].(string); ok {
			if timestamp, err := time.Parse("20060102T150405Z", entry); err == nil {
				task.Entry = timestamp.Unix()
			}
		}
		if modified, ok := data["modified"].(string); ok {
			if timestamp, err := time.Parse("20060102T150405Z", modified); err == nil {
				task.Modified = timestamp.Unix()
			}
		}
		if end, ok := data["end"].(string); ok {
			if timestamp, err := time.Parse("20060102T150405Z", end); err == nil {
				task.End = timestamp.Unix()
			}
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (tw *TaskWarrior) SaveTask(task *Task) error {
	return tw.saveTaskWithCommand(task)
}

func (tw *TaskWarrior) saveTaskWithCommand(task *Task) error {
	var cmd *exec.Cmd

	if task.UUID == "" {
		args := []string{"rc.data.location=" + tw.dataDir, "rc.confirmation=off", "add"}
		if task.Project != "" {
			args = append(args, "project:"+task.Project)
		}
		args = append(args, task.Description)
		cmd = exec.Command("task", args...)

		output, err := cmd.Output()
		if err != nil {
			return err
		}

		outputStr := string(output)
		re := regexp.MustCompile(`Created task (\S+)\.`)
		matches := re.FindStringSubmatch(outputStr)
		if len(matches) > 1 {
			task.UUID = matches[1]
		}
	} else {
		args := []string{"rc.data.location=" + tw.dataDir, "rc.confirmation=off", task.UUID, "modify"}
		if task.Project != "" {
			args = append(args, "project:"+task.Project)
		}
		args = append(args, task.Description)
		cmd = exec.Command("task", args...)

		if _, err := cmd.Output(); err != nil {
			return err
		}
	}

	if task.Status == "completed" {
		cmd = exec.Command("task", "rc.data.location="+tw.dataDir, "rc.confirmation=off", task.UUID, "done")
		if _, err := cmd.Output(); err != nil {
			return err
		}
	}

	return nil
}

func (tw *TaskWarrior) deleteTaskWithCommand(uuid string) error {
	cmd := exec.Command("task", "rc.data.location="+tw.dataDir, "rc.confirmation=off", uuid, "delete")
	_, err := cmd.Output()
	return err
}

func (tw *TaskWarrior) DeleteTask(uuid string) error {
	return tw.deleteTaskWithCommand(uuid)
}
