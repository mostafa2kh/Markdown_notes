// notes.go
// Markdown Notes CLI (single-file)
// Build: go build -o notes notes.go
//
// Commands:
//   notes add <title>            - opens $EDITOR (or vim) to write markdown body
//   notes list                   - list saved notes (id, title, tags)
//   notes view <id>              - print note (title + body)
//   notes search <query>         - search title/body/tags (case-insensitive)
//   notes tag <id> <tag> [tag2]  - add one or more tags to a note
//   notes export <id> <file>     - export note to a simple HTML file
//   notes help                   - show usage
//
// Data: one JSON file per note in ./notes_db/ (note files named 0001.json, 0002.json, ...)

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const dbDir = "notes_db"

type Note struct {
	ID      int       `json:"id"`
	Title   string    `json:"title"`
	Body    string    `json:"body"`
	Tags    []string  `json:"tags"`
	Created time.Time `json:"created"`
}

func ensureDir() error {
	return os.MkdirAll(dbDir, 0o755)
}

func notePath(id int) string {
	return filepath.Join(dbDir, fmt.Sprintf("%04d.json", id))
}

func loadAll() ([]Note, error) {
	if err := ensureDir(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dbDir)
	if err != nil {
		return nil, err
	}
	notes := make([]Note, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dbDir, e.Name()))
		if err != nil {
			continue
		}
		var n Note
		if err := json.Unmarshal(b, &n); err == nil {
			notes = append(notes, n)
		}
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].ID < notes[j].ID })
	return notes, nil
}

func nextID(notes []Note) int {
	max := 0
	for _, n := range notes {
		if n.ID > max {
			max = n.ID
		}
	}
	return max + 1
}

func saveNote(n Note) error {
	if err := ensureDir(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(n, "", "  ")
	if err != nil {
		return err
	}
	tmp := notePath(n.ID) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, notePath(n.ID))
}

func openEditor(initial string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	tmp, err := os.CreateTemp("", "note-*.md")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	_ = tmp.Close()
	// write initial content
	if err := os.WriteFile(name, []byte(initial), 0o644); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	cmd := exec.Command(editor, name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	b, err := os.ReadFile(name)
	_ = os.Remove(name)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func cmdAdd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: add <title>")
	}
	title := strings.Join(args, " ")
	notes, err := loadAll()
	if err != nil {
		return err
	}
	id := nextID(notes)
	body, err := openEditor("")
	if err != nil {
		return fmt.Errorf("editor error: %w", err)
	}
	n := Note{
		ID:      id,
		Title:   title,
		Body:    body,
		Tags:    []string{},
		Created: time.Now().UTC(),
	}
	if err := saveNote(n); err != nil {
		return err
	}
	fmt.Printf("Saved note #%d\n", id)
	return nil
}

func cmdList(args []string) error {
	notes, err := loadAll()
	if err != nil {
		return err
	}
	if len(notes) == 0 {
		fmt.Println("No notes.")
		return nil
	}
	fmt.Printf("ID  Title - tags\n")
	for _, n := range notes {
		fmt.Printf("%3d  %s - %s\n", n.ID, n.Title, strings.Join(n.Tags, ","))
	}
	return nil
}

func loadNote(id int) (*Note, error) {
	b, err := os.ReadFile(notePath(id))
	if err != nil {
		return nil, err
	}
	var n Note
	if err := json.Unmarshal(b, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

func cmdView(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: view <id>")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	n, err := loadNote(id)
	if err != nil {
		return err
	}
	fmt.Printf("# %s\n\n%s\n", n.Title, n.Body)
	return nil
}

func cmdSearch(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: search <query>")
	}
	q := strings.ToLower(strings.Join(args, " "))
	notes, err := loadAll()
	if err != nil {
		return err
	}
	found := 0
	for _, n := range notes {
		if strings.Contains(strings.ToLower(n.Title), q) ||
			strings.Contains(strings.ToLower(n.Body), q) ||
			strings.Contains(strings.ToLower(strings.Join(n.Tags, ",")), q) {
			fmt.Printf("%3d  %s\n", n.ID, n.Title)
			found++
		}
	}
	if found == 0 {
		fmt.Println("No matches.")
	}
	return nil
}

func cmdTag(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: tag <id> tag1 [tag2 ...]")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	n, err := loadNote(id)
	if err != nil {
		return err
	}
	newTags := args[1:]
	for _, t := range newTags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		n.Tags = append(n.Tags, t)
	}
	// dedupe while preserving order
	seen := map[string]bool{}
	res := make([]string, 0, len(n.Tags))
	for _, t := range n.Tags {
		if !seen[t] {
			seen[t] = true
			res = append(res, t)
		}
	}
	n.Tags = res
	if err := saveNote(*n); err != nil {
		return err
	}
	fmt.Printf("Updated tags for #%d\n", id)
	return nil
}

func htmlEscape(s string) string {
	repl := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return repl.Replace(s)
}

func cmdExport(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: export <id> <file.html>")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	out := args[1]
	n, err := loadNote(id)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>" + htmlEscape(n.Title) + "</title>\n</head>\n<body>\n")
	b.WriteString("<h1>" + htmlEscape(n.Title) + "</h1>\n")
	// Very small markdown-ish -> HTML: handle lines starting with "# " as header, otherwise paragraphs.
	lines := strings.Split(n.Body, "\n")
	for _, L := range lines {
		if strings.HasPrefix(L, "# ") {
			b.WriteString("<h2>" + htmlEscape(strings.TrimSpace(strings.TrimPrefix(L, "# "))) + "</h2>\n")
			continue
		}
		if strings.TrimSpace(L) == "" {
			continue
		}
		b.WriteString("<p>" + htmlEscape(L) + "</p>\n")
	}
	b.WriteString("</body>\n</html>\n")
	if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
		return err
	}
	fmt.Printf("Exported note #%d to %s\n", id, out)
	return nil
}

func cmdHelp() {
	prog := filepath.Base(os.Args[0])
	fmt.Printf("%s - markdown notes CLI\n\n", prog)
	fmt.Println("Commands:")
	fmt.Println("  add <title>           Add a note (opens $EDITOR or vim)")
	fmt.Println("  list                  List notes")
	fmt.Println("  view <id>             View note")
	fmt.Println("  search <query>        Search title/body/tags")
	fmt.Println("  tag <id> tag1 ...     Add tags to a note")
	fmt.Println("  export <id> file.html Export to simple HTML")
	fmt.Println("  help                  Show this help")
}

func main() {
	if len(os.Args) < 2 {
		cmdHelp()
		return
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	var err error
	switch cmd {
	case "add":
		err = cmdAdd(args)
	case "list":
		err = cmdList(args)
	case "view":
		err = cmdView(args)
	case "search":
		err = cmdSearch(args)
	case "tag":
		err = cmdTag(args)
	case "export":
		err = cmdExport(args)
	case "help", "-h", "--help":
		cmdHelp()
		return
	default:
		cmdHelp()
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
