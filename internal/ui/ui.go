package ui

import (
	"fmt"
	"io"
	"strings"

	glamour "charm.land/glamour/v2"
	lipgloss "charm.land/lipgloss/v2"
)

type Renderer struct {
	stdout   io.Writer
	stderr   io.Writer
	title    lipgloss.Style
	subtitle lipgloss.Style
	key      lipgloss.Style
	value    lipgloss.Style
	error    lipgloss.Style
	warn     lipgloss.Style
	success  lipgloss.Style
	md       *glamour.TermRenderer
}

func New(stdout, stderr io.Writer) *Renderer {
	md, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		md = nil
	}

	return &Renderer{
		stdout:   stdout,
		stderr:   stderr,
		title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")),
		subtitle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		key:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("44")),
		value:    lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		error:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")),
		warn:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")),
		success:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")),
		md:       md,
	}
}

func (r *Renderer) Stdout() io.Writer { return r.stdout }
func (r *Renderer) Stderr() io.Writer { return r.stderr }

func (r *Renderer) Title(text string) {
	fmt.Fprintln(r.stdout, r.title.Render(text))
}

func (r *Renderer) Subtitle(text string) {
	fmt.Fprintln(r.stdout, r.subtitle.Render(text))
	fmt.Fprintln(r.stdout)
}

func (r *Renderer) Section(text string) {
	fmt.Fprintln(r.stdout, r.title.Render(text))
}

func (r *Renderer) KeyValue(key, value string) {
	fmt.Fprintf(r.stdout, "%s %s\n", r.key.Render(key+":"), r.value.Render(value))
}

func (r *Renderer) Line(text string) {
	fmt.Fprintln(r.stdout, text)
}

func (r *Renderer) Success(text string) {
	fmt.Fprintln(r.stdout, r.success.Render(text))
}

func (r *Renderer) Warning(text string) {
	fmt.Fprintln(r.stdout, r.warn.Render(text))
}

func (r *Renderer) Error(text string) {
	fmt.Fprintln(r.stderr, r.error.Render(text))
}

func (r *Renderer) Markdown(text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	if r.md == nil {
		fmt.Fprintln(r.stdout, trimmed)
		fmt.Fprintln(r.stdout)
		return
	}
	out, err := r.md.Render(trimmed)
	if err != nil {
		fmt.Fprintln(r.stdout, trimmed)
		fmt.Fprintln(r.stdout)
		return
	}
	fmt.Fprint(r.stdout, out)
	if !strings.HasSuffix(out, "\n") {
		fmt.Fprintln(r.stdout)
	}
	fmt.Fprintln(r.stdout)
}
