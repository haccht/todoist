package todoist

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

const (
	popupWidth = 80
)

type UI struct {
	*tview.Application

	table  *tview.Table
	footer *tview.TextView
	pages  *tview.Pages
}

func NewUI() *UI {
	var u UI
	u.Application = tview.NewApplication()

	u.table = tview.NewTable()
	u.table.SetFixed(1, 5).SetSelectable(true, false)

	u.footer = tview.NewTextView()
	u.footer.SetBackgroundColor(tcell.Color239)
	u.footer.SetDynamicColors(true)

	mainPage := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(u.table, 0, 1, true).
		AddItem(u.footer, 1, 1, false)

	u.pages = tview.NewPages()
	u.pages.AddPage("mainPage", mainPage, true, true)

	u.Init()
	return &u
}

func newModal(primitive tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(primitive, height, 1, false).
			AddItem(nil, 0, 1, false), width, 1, false).
		AddItem(nil, 0, 1, false)
}

func (u *UI) Init() {
	u.table.Clear()
	u.table.ScrollToBeginning().Select(1, 0)
	headers := []string{"ID", "DueDate", "Pri", "Project", "Content"}
	for i, header := range headers {
		c := tview.NewTableCell(header)
		c.SetTextColor(tcell.ColorYellow).SetSelectable(false)
		u.table.SetCell(0, i, c)
	}
}

func (u *UI) SetInputCapture(f func(*tcell.EventKey) *tcell.EventKey) {
	u.table.SetInputCapture(f)
}

func (u *UI) GetSelection() int {
	r, _ := u.table.GetSelection()
	return r - 1
}

func (u *UI) RenderTableRow(r int, cells ...*tview.TableCell) {
	for i, c := range cells {
		u.table.SetCell(r+1, i, c)
	}
}

func (u *UI) RemoveTableRow(r int) {
	u.table.RemoveRow(r + 1)
}

func (u *UI) Popup(title, text string) {
	textView := tview.NewTextView()
	textView.SetText(text)
	textView.SetTitle(fmt.Sprintf(" %s ", title)).SetTitleAlign(tview.AlignLeft)
	textView.SetBorder(true).SetBorderPadding(0, 0, 1, 1)
	textView.SetDynamicColors(true)
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			u.pages.SwitchToPage("mainPage")
			u.pages.RemovePage("popupPage")
			u.SetFocus(u.table)
		}
		return event
	})

	u.pages.AddPage("popupPage", newModal(textView, popupWidth+2, 0), true, true)
	u.SetFocus(textView)
}

func (u *UI) PopupConfirm(message string, buttonLabels []string, callback func(string)) {
	modal := tview.NewModal()
	modal.SetText(message).SetTextColor(tcell.ColorRed)
	modal.AddButtons(buttonLabels)
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		callback(buttonLabel)

		u.pages.SwitchToPage("mainPage")
		u.pages.RemovePage("popupModal")
		u.SetFocus(u.table)
	})

	u.pages.AddPage("popupModal", modal, true, true)
	u.SetFocus(modal)
}

func (u *UI) FormInput(title, text string, callback func(string)) {
	input := tview.NewInputField()
	input.SetTitle(fmt.Sprintf(" %s ", title)).SetTitleAlign(tview.AlignLeft)
	input.SetBorder(true).SetBorderPadding(0, 0, 1, 1)
	input.SetFieldBackgroundColor(tcell.ColorBlack)
	input.SetFieldWidth(popupWidth).SetText(text)
	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			callback(input.GetText())
		}

		u.pages.SwitchToPage("mainPage")
		u.pages.RemovePage("popupPage")
		u.SetFocus(u.table)
	})

	u.pages.AddPage("popupPage", newModal(input, popupWidth+2, 3), true, true)
	u.SetFocus(input)
}

func (u *UI) StatusMessage(message string, duration time.Duration) {
	original := u.footer.GetText(false)
	u.footer.SetText(message)

	if duration > 0 {
		go func() {
			time.Sleep(duration)
			u.QueueUpdateDraw(func() {
				u.footer.SetText(original)
			})
		}()
	}
}

func (u *UI) Run() error {
	return u.SetRoot(u.pages, true).Run()
}
