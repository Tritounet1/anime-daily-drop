package scrape

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func PrintTable(entries []Entry, todayStr string) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.Style().Options.SeparateRows = true
	t.AppendHeader(table.Row{"Title", "Day", "Time"})
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMax: 70, WidthMaxEnforcer: text.WrapSoft},
	})

	for _, e := range entries {
		day := text.Colors{text.FgGreen}.Sprint(e.Day)
		if e.Day == todayStr {
			day = text.Colors{text.FgYellow, text.Bold}.Sprint(e.Day)
		}
		t.AppendRow(table.Row{e.Title, day, e.Time})
	}

	t.Render()
}
