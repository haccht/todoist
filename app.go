package todoist

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type Application struct {
	ui     *UI
	config *Config
	client *Client

	tasks    []Task
	labels   map[uint]string
	projects map[uint]string
}

func NewApplication() (*Application, error) {
	config, err := NewConfig()
	if err != nil {
		return nil, err
	}

	ui := NewUI()
	client := NewClient(config.Token)

	a := &Application{ui: ui, config: config, client: client}
	a.ui.StatusMessage("[f[]Filter [v[]Detail [a[]Add [e[]Edit [d[]Due [p[]Project [1-4[]Priority [r[]Refresh [C[]Close [D[]Delete", 0)
	a.ui.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'v':
			a.ShowDescription()
		case 'a':
			a.QuickAdd()
		case 'f':
			a.QuickFilter()
		case 'e':
			a.EditContent()
		case 'd':
			a.EditDuedate()
		case 'p':
			a.EditProject()
		case 'r':
			a.Init()
		case 'C':
			a.Complete()
		case 'D':
			a.Delete()
		case '1':
			a.SetPriority(4)
		case '2':
			a.SetPriority(3)
		case '3':
			a.SetPriority(2)
		case '4':
			a.SetPriority(1)
		case 'q':
			a.ui.Stop()
		}
		return event
	})

	if err := a.Init(); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *Application) Run() error {
	return a.ui.Run()
}

func (a *Application) Init() error {
	a.labels = map[uint]string{}
	if labels, err := a.client.ListLabels(); err != nil {
		return err
	} else {
		for _, label := range *labels {
			a.labels[label.ID] = "@" + label.Name
		}
	}

	a.projects = map[uint]string{}
	if projects, err := a.client.ListProjects(); err != nil {
		return err
	} else {
		for _, project := range *projects {
			a.projects[project.ID] = "#" + project.Name
		}
	}

	if err := a.Update(); err != nil {
		return err
	}
	return nil
}

func (a *Application) ErrorMessage(err error) {
	a.ui.StatusMessage(fmt.Sprintf("[red][ERROR[] %s[-]", err), 3*time.Second)
}

func (a *Application) ShowDescription() {
	_, t := a.GetSelection()

	var b strings.Builder
	fmt.Fprintf(&b, "[::b]Project:[::-]  %s\n", a.Project(t.ProjectID))
	fmt.Fprintf(&b, "[::b]DueDate:[::-]  %s\n", t.DueString())
	fmt.Fprintf(&b, "[::b]Labels:[::-]   %s\n", strings.Join(a.Labels(t.LabelIDs), ","))
	fmt.Fprintf(&b, "[::b]Priority:[::-] P%d\n", 5-t.Priority)
	fmt.Fprintf(&b, "[::b]URL:[::-] %s\n", t.URL)
	fmt.Fprintf(&b, "\n\n%s[-]", t.Content)

	comments, _ := a.client.ListComments(&map[string]interface{}{"task_id": t.ID})
	if len(*comments) > 0 {
		fmt.Fprintf(&b, "\n\n--\n")
		for _, comment := range *comments {
			fmt.Fprintf(&b, "%s\n%s\n", comment.Posted, comment.Content)
		}
	}

	a.ui.Popup("Detail", b.String())
}

func (a *Application) QuickAdd() {
	a.ui.FormInput("Quick add", "", func(text string) {
		var err error
		if err = a.client.QuickAddTask(text, nil); err != nil {
			a.ErrorMessage(err)
			return
		}

		if err = a.Update(); err != nil {
			a.ErrorMessage(err)
		}
	})
}

func (a *Application) QuickFilter() {
	a.ui.FormInput("Quick filter", a.GetFilter(), func(text string) {
		if err := a.SetFilter(text); err != nil {
			a.ErrorMessage(err)
		}
	})
}

func (a *Application) EditContent() {
	r, t := a.GetSelection()
	a.ui.FormInput("Edit content", t.Content, func(text string) {
		var err error
		if err = a.client.UpdateTask(t.ID, &map[string]interface{}{"content": text}); err != nil {
			a.ErrorMessage(err)
			return
		}

		if t, err = a.client.GetTask(t.ID); err != nil {
			a.ErrorMessage(err)
			return
		}

		a.tasks[r] = *t
		a.RenderTableRow(r, t)
	})
}

func (a *Application) EditDuedate() {
	r, t := a.GetSelection()
	a.ui.FormInput("Edit duedate", t.Due.String, func(text string) {
		var err error
		if err = a.client.UpdateTask(t.ID, &map[string]interface{}{"due_string": text}); err != nil {
			a.ErrorMessage(err)
			return
		}

		if t, err = a.client.GetTask(t.ID); err != nil {
			a.ErrorMessage(err)
			return
		}

		a.tasks[r] = *t
		a.RenderTableRow(r, t)
	})
}

func (a *Application) EditProject() {
	r, t := a.GetSelection()
	a.ui.FormInput("Edit project", a.Project(t.ProjectID), func(text string) {
		var projectID uint
		for k, v := range a.projects {
			if strings.EqualFold(text, v) {
				projectID = k
				break
			}
		}

		if projectID == 0 {
			a.ErrorMessage(fmt.Errorf("Invalid project Name: %s", text))
			return
		}

		var err error
		if err = a.client.MoveTask(t.ID, &map[string]interface{}{"project_id": projectID}); err != nil {
			a.ErrorMessage(err)
			return
		}

		if t, err = a.client.GetTask(t.ID); err != nil {
			a.ErrorMessage(err)
			return
		}

		a.tasks[r] = *t
		a.RenderTableRow(r, t)
	})
}

func (a *Application) SetPriority(p int) {
	r, t := a.GetSelection()

	var err error
	if err = a.client.UpdateTask(t.ID, &map[string]interface{}{"priority": p}); err != nil {
		a.ErrorMessage(err)
		return
	}

	if t, err = a.client.GetTask(t.ID); err != nil {
		a.ErrorMessage(err)
		return
	}

	a.tasks[r] = *t
	a.RenderTableRow(r, t)
}

func (a *Application) Complete() {
	r, t := a.GetSelection()
	if err := a.client.CloseTask(t.ID); err != nil {
		a.ErrorMessage(err)
		return
	}

	a.tasks = append(a.tasks[:r], a.tasks[r+1:]...)
	a.ui.RemoveTableRow(r)
}

func (a *Application) Delete() {
	r, t := a.GetSelection()
	message := fmt.Sprintf("Are you sure you want to delete `%s`?", t.Content)
	a.ui.PopupConfirm(message, []string{"Delete", "Cancel"}, func(text string) {
		if text == "Delete" {
			if err := a.client.DeleteTask(t.ID); err != nil {
				a.ErrorMessage(err)
				return
			}

			a.tasks = append(a.tasks[:r], a.tasks[r+1:]...)
			a.ui.RemoveTableRow(r)
		}
	})
}

func (a *Application) GetSelection() (int, *Task) {
	r := a.ui.GetSelection()
	t := a.tasks[r]
	return r, &t
}

func (a *Application) Update() error {
	return a.SetFilter(a.GetFilter())
}

func (a *Application) SetFilter(text string) error {
	a.config.Filter = text
	a.config.Save()

	tasks, err := a.client.ListTasks(&map[string]interface{}{"filter": text})
	if err != nil {
		return err
	}

	a.tasks = *tasks
	a.ui.Init()
	for i, t := range a.tasks {
		a.RenderTableRow(i, &t)
	}

	return nil
}

func (a *Application) GetFilter() string {
	return a.config.Filter
}

func (a *Application) RenderTableRow(r int, t *Task) {
	var c *tview.TableCell
	cells := []*tview.TableCell{}

	c = tview.NewTableCell(fmt.Sprint(t.ID))
	cells = append(cells, c)

	c = tview.NewTableCell(t.DueString())
	cells = append(cells, c)
	switch {
	case t.IsOverdue():
		c.SetTextColor(tcell.ColorRed)
	case t.IsDuedate():
		c.SetTextColor(tcell.ColorFuchsia)
	}

	c = tview.NewTableCell("")
	cells = append(cells, c)
	switch t.Priority {
	case 4:
		c.SetText("P1").SetTextColor(tcell.ColorRed)
	case 3:
		c.SetText("P2").SetTextColor(tcell.ColorIndianRed)
	case 2:
		c.SetText("P3").SetTextColor(tcell.ColorDarkRed)
	}

	c = tview.NewTableCell(a.Project(t.ProjectID))
	c.SetMaxWidth(16)
	cells = append(cells, c)

	c = tview.NewTableCell(t.Content)
	cells = append(cells, c)

	a.ui.RenderTableRow(r, cells...)
}

func (a *Application) Project(projectID uint) string {
	return a.projects[projectID]
}

func (a *Application) Labels(labelIDs []uint) []string {
	labels := []string{}
	for _, labelID := range labelIDs {
		labels = append(labels, a.labels[labelID])
	}
	return labels
}
