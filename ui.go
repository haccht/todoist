package todoist

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type UI struct {
	*tview.Application

	pages  *tview.Pages
	table  *tview.Table
	status *tview.TextView
	footer *tview.Flex
}

func NewUI() *UI {
	var u UI
	u.Application = tview.NewApplication()

	u.table = tview.NewTable()
	u.table.SetFixed(1, 5).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.ColorDefault, tcell.Color235, tcell.AttrUnderline|tcell.AttrBold)

	u.status = tview.NewTextView()
	u.status.SetDynamicColors(true)

	help := tview.NewTextView()
	help.SetText(" [q]Quit [?]Help [Enter]Detail ").
		SetTextAlign(tview.AlignRight).
		SetBackgroundColor(tcell.Color237)

	u.footer = tview.NewFlex().
		AddItem(u.status, 0, 0, false).
		AddItem(help, 0, 1, false)

	main := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(u.table, 0, 1, true).
		AddItem(u.footer, 1, 1, false)

	u.pages = tview.NewPages()
	u.pages.AddPage("main", main, true, true)

	u.Init()
	return &u
}

func modal(primitive tview.Primitive, width, height int) tview.Primitive {
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
		c := tview.NewTableCell(header).SetSelectable(false)
		c.SetAttributes(tcell.AttrBold).
			SetTextColor(tcell.ColorBlack).
			SetBackgroundColor(tcell.ColorWhiteSmoke)

		if i == len(headers)-1 {
			c.SetExpansion(1)
		}

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

func (u *UI) RenderRow(r int, cells ...*tview.TableCell) {
	for i, c := range cells {
		u.table.SetCell(r+1, i, c)
	}
}

func (u *UI) RemoveRow(r int) {
	u.table.RemoveRow(r + 1)
}

func (u *UI) PopupConfirm(message string, buttonLabels []string, callbackFunc func(string)) {
	confirm := tview.NewModal().
		SetText(message).SetTextColor(tcell.ColorRed).
		AddButtons(buttonLabels).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			callbackFunc(buttonLabel)

			u.SetFocus(u.table)
			u.pages.HidePage("modal").RemovePage("modal")
		})

	u.pages.AddPage("modal", confirm, true, true)
	u.SetFocus(confirm)
}

func (u *UI) PopupInput(title, text string, callbackFunc func(string)) {
	_, _, width, _ := u.pages.GetRect()
	innterWidth := int(float32(width) * 0.8)

	input := tview.NewInputField()
	input.SetFieldWidth(innterWidth).SetText(text).
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetTitle(fmt.Sprintf(" %s ", title)).SetTitleAlign(tview.AlignLeft).
		SetBorder(true).SetBorderPadding(0, 0, 1, 1)

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlD:
			return tcell.NewEventKey(tcell.KeyDelete, event.Rune(), event.Modifiers())
		case tcell.KeyCtrlF:
			return tcell.NewEventKey(tcell.KeyRight, event.Rune(), event.Modifiers())
		case tcell.KeyCtrlB:
			return tcell.NewEventKey(tcell.KeyLeft, event.Rune(), event.Modifiers())
		}
		return event
	})

	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			callbackFunc(strings.TrimSpace(input.GetText()))
		}

		u.SetFocus(u.table)
		u.pages.HidePage("modal").RemovePage("modal")
	})

	u.pages.AddPage("modal", modal(input, innterWidth+2, 3), true, true)
	u.SetFocus(input)
}

func (u *UI) Popup(title, content string) {
	_, _, width, _ := u.pages.GetRect()
	innterWidth := int(float32(width) * 0.8)

	text := tview.NewTextView()
	text.SetText(content).
		SetDynamicColors(true).
		SetTitle(fmt.Sprintf(" %s ", title)).SetTitleAlign(tview.AlignLeft).
		SetBorder(true).SetBorderPadding(0, 0, 1, 1)

	text.SetDoneFunc(func(key tcell.Key) {
		u.SetFocus(u.table)
		u.pages.HidePage("modal").RemovePage("modal")
	})

	u.pages.AddPage("modal", modal(text, innterWidth+2, 0), true, true)
	u.SetFocus(text)
}

func (u *UI) StatusLine(message string, duration time.Duration) {
	original := u.status.GetText(false)

	u.status.SetText(message)
	u.footer.ResizeItem(u.status, tview.TaggedStringWidth(message), 0)

	if duration > 0 {
		go func() {
			time.Sleep(duration)
			u.QueueUpdateDraw(func() {
				u.status.SetText(original)
				u.footer.ResizeItem(u.status, tview.TaggedStringWidth(original), 0)
			})
		}()
	}
}

func (u *UI) FilterStatus(filter string) {
	u.StatusLine(fmt.Sprintf("[black:white:b] %s ", tview.Escape(filter)), 0*time.Second)
}

func (u *UI) ErrorMessage(err error) {
	u.StatusLine(fmt.Sprintf("[white:red:b] ERROR - %s ", tview.Escape(err.Error())), 3*time.Second)
}

func (u *UI) Run() error {
	return u.SetRoot(u.pages, true).Run()
}
