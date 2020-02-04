package todoist

import (
	"log"
	"os"
	"testing"
)

func TestTask(t *testing.T) {
	c := NewClient(os.Getenv("TODOIST_TOKEN"))
	c.Logger = log.New(os.Stdout, "[DEBUG] ", log.LstdFlags)

	item, err := c.AddTask(&map[string]interface{}{"content": "created"})
	if err != nil {
		t.Fatalf("Failed to create a task: %s", err)
	}

	list, err := c.ListTasks(nil)
	if err != nil {
		t.Fatalf("Failed to list tasks: %s", err)
	}

	found := false
	for _, v := range list {
		if v.ID == item.ID && v.Content == "created" {
			found = true
		}
	}
	if !found {
		t.Fatal("Failed to list the created task")
	}

	err = c.UpdateTask(item.ID, &map[string]interface{}{"content": "modified"})
	if err != nil {
		t.Fatalf("Failed to update the task: %s", err)
	}

	item, err = c.GetTask(item.ID)
	if err != nil {
		t.Fatalf("Failed to get the task: %s", err)
	}
	if item.Content != "modified" {
		t.Fatalf("Failed to update the task: task unmodified")
	}

	err = c.CloseTask(item.ID)
	if err != nil {
		t.Fatalf("Failed to close the task: %s", err)
	}

	err = c.ReopenTask(item.ID)
	if err != nil {
		t.Fatalf("Failed to reopen the task: %s", err)
	}

	err = c.DeleteTask(item.ID)
	if err != nil {
		t.Fatalf("Failed to delete the task: %s", err)
	}
}
