package scrape

import (
	"os"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func PrintTable(entries []Entry, now time.Time) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.Style().Options.SeparateRows = true
	t.AppendHeader(table.Row{"Titre", "Ép.", "Jour", "Heure"})
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMax: 70, WidthMaxEnforcer: text.WrapSoft},
	})

	for _, e := range entries {
		day := text.Colors{text.FgGreen}.Sprint(e.Day())
		if e.IsSameDay(now) {
			day = text.Colors{text.FgYellow, text.Bold}.Sprint(e.Day())
		}
		t.AppendRow(table.Row{e.Title, e.Episode, day, e.Time()})
	}

	t.Render()
}
