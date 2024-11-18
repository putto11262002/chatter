package template

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var (
	exts = []string{".html", ".tmpl", ".tpl"}
)

type TemplStore struct {
	root      *template.Template
	templates map[string]*template.Template
}

func newTemplStore(fsys fs.FS) *TemplStore {

	rootTempl := template.New("root")

	sourceRootFn := func(fsys fs.FS, p string, key string) error {

		f, err := fsys.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		content, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		rootTempl, err = rootTempl.New(key).Parse(string(content))
		if err != nil {
			return err
		}

		return nil

	}

	if err := sourceTemplates(fsys, "shared", exts, sourceRootFn); err != nil {
		panic(err)
	}

	templates := make(map[string]*template.Template)

	sourceTemplsFn := func(fsys fs.FS, p string, key string) error {

		f, err := fsys.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		content, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		templ, err := rootTempl.Clone()
		if err != nil {
			return err
		}

		templates[key], err = templ.New(key).Parse(string(content))
		if err != nil {
			return err
		}
		return nil
	}

	if err := sourceTemplates(fsys, ".", exts, sourceTemplsFn); err != nil {
		panic(err)
	}

	return &TemplStore{root: rootTempl, templates: templates}
}

func NewTemplStore(path string) *TemplStore {
	fsys := os.DirFS(path)
	return newTemplStore(fsys)

}

func (s *TemplStore) Render(w io.Writer, template string, data interface{}) error {
	if templ, ok := s.templates[template]; !ok {
		return fmt.Errorf("template %s not found", template)
	} else {
		return templ.Execute(w, data)
	}
}

func sourceTemplates(fsys fs.FS, base string, exts []string, fn func(fsys fs.FS, path string, key string) error) error {

	return fs.WalkDir(fsys, base, func(p string, d fs.DirEntry, err error) error {

		// do not walk the root directory twice
		if p == "." {
			return nil
		}

		if err != nil {
			return err
		}

		if !d.Type().IsRegular() {
			return nil
		}

		ext := filepath.Ext(d.Name())

		if !slices.Contains(exts, ext) {
			return nil
		}

		key := strings.TrimSuffix(p, ext)

		return fn(fsys, p, key)

	})
}
