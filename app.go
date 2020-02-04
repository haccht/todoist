package todoist

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type Application struct {
	ui     *UI
	config *Config
	client *Client

	tasks    []*Task
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
	a.ui.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			a.ShowDetail()
		} else {
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
				a.Stop()
			}
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

func (a *Application) Stop() {
	a.ui.Stop()
}

func (a *Application) Init() error {
	a.labels = map[uint]string{}
	if labels, err := a.client.ListLabels(); err != nil {
		return err
	} else {
		for _, label := range labels {
			a.labels[label.ID] = "@" + label.Name
		}
	}

	a.projects = map[uint]string{}
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
	var help = `
       q [yellow]quit[-]
       ? [yellow]help[-]

       f [yellow]filter list[-]
       r [yellow]refresh lisk[-]

       a [yellow]quick add[-]
       v [yellow]task detail[-]
   enter [yellow]task detail[-]

 shift+c [yellow]close task[-]
 shift+d [yellow]delete task[-]

       e [yellow]edit text[-]
       p [yellow]move project[-]
       d [yellow]set due date[-]
     1-4 [yellow]set priority P1 to P4[-]`

	a.ui.Popup("Help", help)
}

func (a *Application) ShowDetail() {
	_, t := a.GetSelection()

	var b strings.Builder
	fmt.Fprintf(&b, "[::b]Project:[::-]  %s\n", a.Project(t.ProjectID))
	fmt.Fprintf(&b, "[::b]DueDate:[::-]  %s\n", t.DueString())
	fmt.Fprintf(&b, "[::b]Labels:[::-]   %s\n", strings.Join(a.Labels(t.LabelIDs), ","))
	fmt.Fprintf(&b, "[::b]Priority:[::-] P%d\n", 5-t.Priority)
	fmt.Fprintf(&b, "[::b]URL:[::-] %s\n", t.URL)

	rep := regexp.MustCompile("\\((https?://[^\\)]+)\\)")
	fmt.Fprintf(&b, "\n\n%s[-]", rep.ReplaceAllString(t.Content, "( $1 )"))

	comments, err := a.client.ListComments(&map[string]interface{}{"task_id": t.ID})
	if err != nil {
		a.ui.ErrorMessage(err)
	} else if len(comments) > 0 {
		fmt.Fprintf(&b, "\n\n--\n")
		for _, comment := range comments {
			fmt.Fprintf(&b, "%s\n%s\n", comment.Posted, rep.ReplaceAllString(comment.Content, "( $1 )"))
		}
	}

	a.ui.Popup("Detail", b.String())
}

func (a *Application) QuickFilter() {
	a.ui.FormInput("Quick filter", a.config.Filter, func(text string) {
		if err := a.SetFilter(text); err != nil {
			a.ui.ErrorMessage(err)
		}
	})
}

func (a *Application) QuickAdd() {
	a.ui.FormInput("Quick add", "", func(text string) {
		var err error
		if err = a.client.QuickAddTask(text, nil); err != nil {
			a.ui.ErrorMessage(err)
			return
		}

		if err = a.Update(); err != nil {
			a.ui.ErrorMessage(err)
		}
	})
}

func (a *Application) EditContent() {
	r, t := a.GetSelection()
	a.ui.FormInput("Edit text", t.Content, func(text string) {
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
	a.ui.FormInput("Edit due date", t.Due.String, func(text string) {
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

func (a *Application) EditProject() {
	r, t := a.GetSelection()
	a.ui.FormInput("Move project", a.Project(t.ProjectID), func(text string) {
		var projectID uint
		for k, v := range a.projects {
			if strings.EqualFold(text, v) {
				projectID = k
				break
			}
		}

		if projectID == 0 {
			a.ui.ErrorMessage(fmt.Errorf("Invalid project Name: %s", text))
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

func (a *Application) Complete() {
	r, t := a.GetSelection()
	if err := a.client.CloseTask(t.ID); err != nil {
		a.ui.ErrorMessage(err)
		return
	}

	a.tasks = append(a.tasks[:r], a.tasks[r+1:]...)
	a.ui.RemoveRow(r)
}

func (a *Application) Delete() {
	r, t := a.GetSelection()
	message := fmt.Sprintf("Are you sure you want to delete `%s`?", t.Content)
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

func (a *Application) SetFilter(filter string) error {
	if filter == "" {
		filter = "#inbox"
	}

	tasks, err := a.client.ListTasks(&map[string]interface{}{"filter": filter})
	if err != nil {
		return err
	}

	a.tasks = tasks
	a.config.Filter = filter
	a.config.Save()

	a.ui.Init()
	a.ui.FilterStatus(filter)
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
		c.SetTextColor(tcell.ColorFuchsia)
	}

	c = tview.NewTableCell(fmt.Sprintf("P%d", 5-t.Priority))
	cells = append(cells, c)
	switch t.Priority {
	case 4:
		c.SetTextColor(tcell.ColorRed)
	case 3:
		c.SetTextColor(tcell.ColorIndianRed)
	case 2:
		c.SetTextColor(tcell.ColorDarkRed)
	}

	c = tview.NewTableCell(a.Project(t.ProjectID)).SetMaxWidth(16)
	cells = append(cells, c)

	c = tview.NewTableCell(t.Content)
	cells = append(cells, c)

	return cells
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
