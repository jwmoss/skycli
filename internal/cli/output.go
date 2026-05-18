package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type printer struct {
	w     io.Writer
	json  bool
	plain bool
}

func newPrinter(w io.Writer, asJSON, plain bool) *printer {
	return &printer{w: w, json: asJSON, plain: plain}
}

func (p *printer) JSON(v any) error {
	enc := json.NewEncoder(p.w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (p *printer) Table(headers []string, rows [][]string) {
	if p.plain {
		fmt.Fprintln(p.w, strings.Join(headers, "\t"))
		for _, r := range rows {
			fmt.Fprintln(p.w, strings.Join(r, "\t"))
		}
		return
	}
	tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	tw.Flush()
}

func (p *printer) Line(format string, args ...any) {
	fmt.Fprintf(p.w, format+"\n", args...)
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func ptrStr(s *string) string {
	if s == nil {
		return "-"
	}
	return *s
}

func ptrIntStr(i *int) string {
	if i == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *i)
}

func boolYN(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
