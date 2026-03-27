package ui

import (
	"fmt"
	"io"
	"strings"

	glamour "charm.land/glamour/v2"
	lipgloss "charm.land/lipgloss/v2"
)

type Cell struct {
	Text  string
	Style string
}

type Renderer struct {
	stdout io.Writer
	stderr io.Writer

	title      lipgloss.Style
	subtitle   lipgloss.Style
	panelTitle lipgloss.Style
	panelBody  lipgloss.Style
	header     lipgloss.Style
	key        lipgloss.Style
	value      lipgloss.Style
	muted      lipgloss.Style
	path       lipgloss.Style
	command    lipgloss.Style
	preset     lipgloss.Style
	gitCmd     lipgloss.Style
	buildCmd   lipgloss.Style
	deployCmd  lipgloss.Style
	infoCmd    lipgloss.Style
	error      lipgloss.Style
	warn       lipgloss.Style
	success    lipgloss.Style
	ok         lipgloss.Style
	fail       lipgloss.Style
	md         *glamour.TermRenderer
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
		stdout:     stdout,
		stderr:     stderr,
		title:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#81A1C1")),
		subtitle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#A3BE8C")),
		panelTitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#81A1C1")),
		panelBody:  lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")),
		header:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#81A1C1")),
		key:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A3BE8C")),
		value:      lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")),
		muted:      lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEE9")),
		path:       lipgloss.NewStyle().Foreground(lipgloss.Color("#88C0D0")),
		command:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#88C0D0")),
		preset:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EBCB8B")),
		gitCmd:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#88C0D0")),
		buildCmd:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EBCB8B")),
		deployCmd:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#D08770")),
		infoCmd:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#B48EAD")),
		error:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#BF616A")),
		warn:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EBCB8B")),
		success:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A3BE8C")),
		ok:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A3BE8C")),
		fail:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#BF616A")),
		md:         md,
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
	fmt.Fprintln(r.stdout, r.header.Render(text))
}

func (r *Renderer) Panel(title string, lines ...string) {
	body := strings.Join(lines, "\n")
	if body == "" {
		body = title
		title = ""
	}
	content := r.panelBody.Render(body)
	if title != "" {
		content = r.panelTitle.Render(title) + "\n" + content
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#81A1C1")).
		Padding(0, 1).
		Render(content)
	fmt.Fprintln(r.stdout)
	fmt.Fprintln(r.stdout, box)
	fmt.Fprintln(r.stdout)
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

func (r *Renderer) Status(status string) string {
	switch status {
	case "OK":
		return r.ok.Render(status)
	case "WARN":
		return r.warn.Render(status)
	case "FAIL":
		return r.fail.Render(status)
	default:
		return status
	}
}

func (r *Renderer) Styled(styleName, text string) string {
	switch styleName {
	case "header":
		return r.header.Render(text)
	case "key":
		return r.key.Render(text)
	case "val":
		return r.value.Render(text)
	case "muted":
		return r.muted.Render(text)
	case "path":
		return r.path.Render(text)
	case "cmd":
		return r.command.Render(text)
	case "preset":
		return r.preset.Render(text)
	case "git-cmd":
		return r.gitCmd.Render(text)
	case "build-cmd":
		return r.buildCmd.Render(text)
	case "deploy-cmd":
		return r.deployCmd.Render(text)
	case "info-cmd":
		return r.infoCmd.Render(text)
	case "success":
		return r.success.Render(text)
	case "warning":
		return r.warn.Render(text)
	case "error":
		return r.error.Render(text)
	default:
		return text
	}
}

func (r *Renderer) Table(title string, headers []Cell, rows [][]Cell) {
	if title != "" {
		fmt.Fprintln(r.stdout, r.header.Render(title))
	}
	colCount := len(headers)
	for _, row := range rows {
		if len(row) > colCount {
			colCount = len(row)
		}
	}
	if colCount == 0 {
		fmt.Fprintln(r.stdout)
		return
	}
	widths := make([]int, colCount)
	for i, h := range headers {
		if w := lipgloss.Width(h.Text); w > widths[i] {
			widths[i] = w
		}
	}
	for _, row := range rows {
		for i, cell := range row {
			if w := lipgloss.Width(cell.Text); w > widths[i] {
				widths[i] = w
			}
		}
	}
	printRow := func(row []Cell, isHeader bool) {
		parts := make([]string, colCount)
		for i := 0; i < colCount; i++ {
			cell := Cell{}
			if i < len(row) {
				cell = row[i]
			}
			text := cell.Text
			if pad := widths[i] - lipgloss.Width(text); pad > 0 {
				text += strings.Repeat(" ", pad)
			}
			style := cell.Style
			if isHeader && style == "" {
				style = "key"
			}
			parts[i] = r.Styled(style, text)
		}
		fmt.Fprintln(r.stdout, strings.TrimRight(strings.Join(parts, "  "), " "))
	}
	if len(headers) > 0 {
		printRow(headers, true)
		underline := make([]Cell, colCount)
		for i := range underline {
			underline[i] = Cell{Text: strings.Repeat("-", max(widths[i], 3)), Style: "muted"}
		}
		printRow(underline, false)
	}
	for _, row := range rows {
		printRow(row, false)
	}
	fmt.Fprintln(r.stdout)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
