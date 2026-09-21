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
	err   error
}

func newPrinter(w io.Writer, asJSON, plain bool) *printer {
	return &printer{w: w, json: asJSON, plain: plain}
}

func (p *printer) JSON(v any) error {
	if p.err != nil {
		return p.err
	}
	enc := json.NewEncoder(p.w)
	enc.SetIndent("", "  ")
	p.err = enc.Encode(v)
	return p.err
}

func (p *printer) Table(headers []string, rows [][]string) {
	if p.plain {
		p.Line("%s", strings.Join(headers, "\t"))
		for _, r := range rows {
			p.Line("%s", strings.Join(r, "\t"))
		}
		return
	}
	tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)
	table := newPrinter(tw, false, false)
	table.Line("%s", strings.Join(headers, "\t"))
	for _, r := range rows {
		table.Line("%s", strings.Join(r, "\t"))
	}
	p.err = table.err
	if p.err == nil {
		p.err = tw.Flush()
	}
}

func (p *printer) Line(format string, args ...any) {
	if p.err == nil {
		_, p.err = fmt.Fprintf(p.w, format+"\n", args...)
	}
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
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
