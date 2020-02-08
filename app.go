package todoist

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type Application struct {
	ui     *UI
	client *Client
	config *Config

	tasks    []*Task
	labels   map[uint]string
	projects map[uint]string
}

func NewApplication() (*Application, error) {
	config, err := NewConfig()
	if err != nil {
		return nil, err
	}

	a := &Application{
		ui:       NewUI(),
		client:   NewClient(config.Token),
		config:   config,
		tasks:    []*Task{},
		labels:   map[uint]string{},
		projects: map[uint]string{},
	}

	a.ui.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			a.ShowDetail()
		case tcell.KeyEscape:
			a.Stop()
		default:
			switch event.Rune() {
			case '?':
				a.ShowHelp()
			case 'v':
				a.ShowDetail()
			case 'a':
				a.QuickAdd()
			case 'f':
				a.QuickFilter()
			case 'e':
				a.EditContent()
			case 'd':
				a.EditDuedate()
			case 'p':
				a.MoveProject()
			case 'r':
				a.Refresh()
			case 'D':
				a.Delete()
			case 'C':
				a.Complete()
			case 'u':
				a.Reopen()
			case '1':
				a.SetPriority(4)
			case '2':
				a.SetPriority(3)
			case '3':
				a.SetPriority(2)
			case '4':
				a.SetPriority(1)
			case 'q':
				a.Stop()
			}
		}

		return event
	})

	if err := a.Refresh(); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *Application) Run() error {
	return a.ui.Run()
}

func (a *Application) Stop() {
	a.ui.Stop()
}

func (a *Application) Refresh() error {
	if labels, err := a.client.ListLabels(); err != nil {
		return err
	} else {
		for _, label := range labels {
			a.labels[label.ID] = "@" + label.Name
		}
	}

	if projects, err := a.client.ListProjects(); err != nil {
		return err
	} else {
		for _, project := range projects {
			a.projects[project.ID] = "#" + project.Name
		}
	}

	if err := a.Update(); err != nil {
		return err
	}
	return nil
}

func (a *Application) ShowHelp() {
	var help = `       [::b]Q :[::-] Quit
       [::b]? :[::-] Help

       [::b]F :[::-] Filter the list
       [::b]R :[::-] Refresh the lisk

       [::b]A :[::-] Quick add
       [::b]V :[::-] Task detail
   [::b]Enter :[::-] Task detail

 [::b]Shift-D :[::-] Delete a task
 [::b]Shift-C :[::-] Complete a task
       [::b]U :[::-] Uncomplete a task

       [::b]E :[::-] Edit the text
       [::b]P :[::-] Move the project
       [::b]D :[::-] Set the due date
     [::b]1-4 :[::-] Set the priority P1 to P4`

	a.ui.Popup("Help", help)
}

func (a *Application) ShowDetail() {
	_, t := a.GetSelection()

	var b strings.Builder
	fmt.Fprintf(&b, "[::b]Project:[-::-]  %s\n", a.project(t.ProjectID))
	fmt.Fprintf(&b, "[::b]DueDate:[-::-]  %s\n", t.DueString())
	fmt.Fprintf(&b, "[::b]Labels:[-::-]   %s\n", strings.Join(a.label(t.LabelIDs), ","))
	fmt.Fprintf(&b, "[::b]Priority:[-::-] P%d\n", 5-t.Priority)
	fmt.Fprintf(&b, "[::b]URL:[-::-] %s\n", t.URL)

	fmt.Fprintf(&b, "\n\n%s", tview.Escape(marginLink(t.Content)))

	comments, err := a.client.ListComments(&map[string]interface{}{"task_id": t.ID})
	if err != nil {
		a.ui.ErrorMessage(err)
	} else if len(comments) > 0 {
		fmt.Fprintf(&b, "\n\n--")
		for _, comment := range comments {
			fmt.Fprintf(&b, "\n%s\n%s", comment.Posted, tview.Escape(marginLink(comment.Content)))
		}
	}

	a.ui.Popup("Detail", b.String())
}

func (a *Application) QuickFilter() {
	a.ui.PopupInput("Quick filter", a.config.Filter, func(text string) {
		if err := a.SetFilter(text); err != nil {
			a.ui.ErrorMessage(err)
		}
	})
}

func (a *Application) QuickAdd() {
	a.ui.PopupInput("Quick add", "", func(text string) {
		var err error
		if err = a.client.QuickAddTask(text, nil); err != nil {
			a.ui.ErrorMessage(err)
			return
		}

		if err = a.Update(); err != nil {
			a.ui.ErrorMessage(err)
			return
		}
	})
}

func (a *Application) EditContent() {
	r, t := a.GetSelection()
	a.ui.PopupInput("Edit text", t.Content, func(text string) {
		var err error
		if err = a.client.UpdateTask(t.ID, &map[string]interface{}{"content": text}); err != nil {
			a.ui.ErrorMessage(err)
			return
		}

		if t, err = a.client.GetTask(t.ID); err != nil {
			a.ui.ErrorMessage(err)
			return
		}

		a.tasks[r] = t
		a.ui.RenderRow(r, a.cells(r, t)...)
	})
}

func (a *Application) EditDuedate() {
	r, t := a.GetSelection()
	a.ui.PopupInput("Edit due date", t.Due.String, func(text string) {
		var err error
		if err = a.client.UpdateTask(t.ID, &map[string]interface{}{"due_string": text}); err != nil {
			a.ui.ErrorMessage(err)
			return
		}

		if t, err = a.client.GetTask(t.ID); err != nil {
			a.ui.ErrorMessage(err)
			return
		}

		a.tasks[r] = t
		a.ui.RenderRow(r, a.cells(r, t)...)
	})
}

func (a *Application) MoveProject() {
	r, t := a.GetSelection()
	a.ui.PopupInput("Move project", a.project(t.ProjectID), func(text string) {
		var projectID uint
		for k, v := range a.projects {
			if strings.EqualFold(text, v) {
				projectID = k
				break
			}
		}

		if projectID == 0 {
			a.ui.ErrorMessage(fmt.Errorf("Invalid project name: %s", text))
			return
		}

		var err error
		if err = a.client.MoveTask(t.ID, &map[string]interface{}{"project_id": projectID}); err != nil {
			a.ui.ErrorMessage(err)
			return
		}

		if t, err = a.client.GetTask(t.ID); err != nil {
			a.ui.ErrorMessage(err)
			return
		}

		a.tasks[r] = t
		a.ui.RenderRow(r, a.cells(r, t)...)
	})
}

func (a *Application) SetPriority(p int) {
	r, t := a.GetSelection()

	var err error
	if err = a.client.UpdateTask(t.ID, &map[string]interface{}{"priority": p}); err != nil {
		a.ui.ErrorMessage(err)
		return
	}

	if t, err = a.client.GetTask(t.ID); err != nil {
		a.ui.ErrorMessage(err)
		return
	}

	a.tasks[r] = t
	a.ui.RenderRow(r, a.cells(r, t)...)
}

func (a *Application) Reopen() {
	if a.config.Closed == 0 {
		return
	}

	var err error
	if err = a.client.ReopenTask(a.config.Closed); err != nil {
		a.ui.ErrorMessage(err)
		return
	}

	if err = a.Update(); err != nil {
		a.ui.ErrorMessage(err)
		return
	}
}

func (a *Application) Complete() {
	r, t := a.GetSelection()
	if err := a.client.CloseTask(t.ID); err != nil {
		a.ui.ErrorMessage(err)
		return
	}

	a.config.Closed = t.ID
	a.config.Save()

	a.tasks = append(a.tasks[:r], a.tasks[r+1:]...)
	a.ui.RemoveRow(r)
}

func (a *Application) Delete() {
	r, t := a.GetSelection()
	message := fmt.Sprintf("Are you sure you want to delete `%s`?", sanitizeLink(t.Content))
	a.ui.PopupConfirm(message, []string{"Delete", "Cancel"}, func(text string) {
		if text == "Delete" {
			if err := a.client.DeleteTask(t.ID); err != nil {
				a.ui.ErrorMessage(err)
				return
			}

			a.tasks = append(a.tasks[:r], a.tasks[r+1:]...)
			a.ui.RemoveRow(r)
		}
	})
}

func (a *Application) GetSelection() (int, *Task) {
	r := a.ui.GetSelection()
	t := a.tasks[r]
	return r, t
}

func (a *Application) Update() error {
	return a.SetFilter(a.config.Filter)
}

func (a *Application) SetFilter(str string) error {
	isPremium, err := a.client.isPremium()
	if err != nil {
		a.ui.ErrorMessage(err)
		return err
	}

	if str == "" {
		str = "#inbox"
	}

	var params map[string]interface{}
	if isPremium {
		params = map[string]interface{}{"filter": str}
	} else {
		var projectID uint
		for k, v := range a.projects {
			if strings.EqualFold(str, v) {
				projectID = k
				break
			}
		}

		if projectID == 0 {
			a.ui.ErrorMessage(fmt.Errorf("Invalid project name: %s", str))
			return err
		}

		params = map[string]interface{}{"project_id": projectID}
	}

	if a.tasks, err = a.client.ListTasks(&params); err != nil {
		return err
	}

	a.config.Filter = str
	a.config.Save()

	a.ui.Init()
	a.ui.FilterStatus(str)
	for i, t := range a.tasks {
		a.ui.RenderRow(i, a.cells(i, t)...)
	}

	return nil
}

func (a *Application) cells(r int, t *Task) []*tview.TableCell {
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
		c.SetTextColor(tcell.ColorIndianRed)
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

	c = tview.NewTableCell(a.project(t.ProjectID)).SetMaxWidth(16)
	cells = append(cells, c)

	c = tview.NewTableCell(sanitizeLink(t.Content))
	cells = append(cells, c)

	return cells
}

func (a *Application) project(projectID uint) string {
	return a.projects[projectID]
}

func (a *Application) label(labelIDs []uint) []string {
	list := []string{}
	for _, labelID := range labelIDs {
		list = append(list, a.labels[labelID])
	}
	return list
}
