package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Thelost77/beech/internal/app"
	"github.com/Thelost77/beech/internal/buildinfo"
	"github.com/Thelost77/beech/internal/document"
	"github.com/Thelost77/beech/internal/outline"
	"github.com/Thelost77/beech/internal/storage"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage: beech [--version] [FILE]")
		flag.PrintDefaults()
	}
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("beech %s\n", buildinfo.Current())
		return
	}
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}

	var path string
	if flag.NArg() == 1 {
		path = flag.Arg(0)
	}
	initial, err := openInitial(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "beech: %v\n", err)
		os.Exit(1)
	}

	model := app.New(initial)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "beech: %v\n", err)
		os.Exit(1)
	}
}

func openInitial(path string) (app.InitialDocument, error) {
	style := outline.DefaultStyle()
	if path == "" {
		doc := document.New("New map")
		data, err := outline.Serialize(doc, nil, style)
		if err != nil {
			return app.InitialDocument{}, err
		}
		return app.InitialDocument{Document: doc, Style: style, SavedData: data}, nil
	}

	loaded, err := storage.Load(path)
	if err == nil {
		extension := strings.ToLower(filepath.Ext(loaded.Path))
		switch extension {
		case ".md":
			parsed, parseErr := outline.Parse(loaded.Data)
			if parseErr != nil {
				return app.InitialDocument{}, fmt.Errorf("parse %s: %w", path, parseErr)
			}
			return app.InitialDocument{
				Document:    parsed.Document,
				Collapsed:   parsed.Collapsed,
				Path:        loaded.Path,
				Style:       parsed.Style,
				Fingerprint: loaded.Fingerprint,
				FileMode:    loaded.Mode,
				SavedData:   loaded.Data,
			}, nil
		case ".hmm":
			parsed, importErr := outline.ImportHMM(loaded.Data)
			if importErr != nil {
				return app.InitialDocument{}, fmt.Errorf("import %s: %w", path, importErr)
			}
			suggestedPath := strings.TrimSuffix(loaded.Path, filepath.Ext(loaded.Path)) + ".md"
			return app.InitialDocument{
				Document:      parsed.Document,
				Collapsed:     parsed.Collapsed,
				SuggestedPath: suggestedPath,
				Imported:      true,
				Style:         parsed.Style,
				FileMode:      loaded.Mode,
				Dirty:         true,
			}, nil
		default:
			return app.InitialDocument{}, fmt.Errorf("unsupported file %s: Beech opens .md files and imports existing .hmm files", path)
		}
	}
	if !errors.Is(err, os.ErrNotExist) {
		return app.InitialDocument{}, fmt.Errorf("open %s: %w", path, err)
	}

	if extension := strings.ToLower(filepath.Ext(path)); extension == "" {
		path += ".md"
	} else if extension != ".md" {
		return app.InitialDocument{}, fmt.Errorf("new documents must use the .md extension")
	}
	absolute, err := storage.NewPath(path)
	if err != nil {
		return app.InitialDocument{}, err
	}
	title := strings.TrimSpace(strings.TrimSuffix(filepath.Base(absolute), filepath.Ext(absolute)))
	if title == "" || title == "." {
		title = "New map"
	}
	return app.InitialDocument{
		Document: document.New(title),
		Path:     absolute,
		Style:    style,
		FileMode: fs.FileMode(0o644),
		Dirty:    true,
	}, nil
}
