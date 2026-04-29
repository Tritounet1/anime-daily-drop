package scrape

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"os"
)

const (
	Reset  = "\033[0m"
	Cyan   = "\033[36m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
)

func PrintTable(entries []Entry, todayStr string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%sTITLE%s\t%sDAY%s\t%sTIME%s\n", Cyan, Reset, Cyan, Reset, Cyan, Reset)
	fmt.Fprintln(w, strings.Repeat("─", 10)+"\t"+strings.Repeat("─", 10)+"\t"+strings.Repeat("─", 4))
	for _, e := range entries {
		color := Green
		if e.Day == todayStr {
			color = Yellow
		}
		fmt.Fprintf(w, "%s%s%s\t%s%s%s\t%s\n",
			Cyan, e.Title, Reset,
			color, e.Day, Reset,
			e.Time)
	}
	w.Flush()
}

func Truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
