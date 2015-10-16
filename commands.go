package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func clean(app *application) error {
	app.Log("clean start")
	outputdir := app.Config.OutputDir
	for _, target := range app.Config.Clean {
		path := filepath.Join(outputdir, target)
		app.Log("remove: %v", path)
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	app.Log("-----------------------------")
	app.Log("clean: OK")
	app.Log("-----------------------------")
	return nil
}

func newsite(app *application, path string) error {
	if len(path) == 0 {
		return errors.New("empty path")
	}
	zipfile, err := ioutil.TempFile("", "silkylog")
	if err != nil {
		return err
	}
	zipfile.Close()
	zippath := zipfile.Name()
	if err := download("https://github.com/yuin/silkylog/archive/master.zip", zippath); err != nil {
		return err
	}
	defer os.Remove(zippath)
	if err := unzip(zippath, path); err != nil {
		return err
	}
	silkylogpath := filepath.Join(path, "silkylog-master")

	// remove *.go files
	if gofiles, err := filepath.Glob(filepath.Join(silkylogpath, "*.go")); err != nil {
		return err
	} else {
		for _, gofile := range gofiles {
			if err := os.Remove(gofile); err != nil {
				return err
			}
		}
	}
	// remove files
	for _, rfile := range []string{".gitignore", "LICENSE", "README.rst"} {
		rpath := filepath.Join(silkylogpath, rfile)
		if err := os.Remove(rpath); err != nil {
			return err
		}
	}
	// create dirs
	for _, ndir := range []string{"public_html"} {
		ndirpath := filepath.Join(silkylogpath, ndir)
		if err := os.MkdirAll(ndirpath, 0755); err != nil {
			return err
		}
	}

	if files, err := filepath.Glob(filepath.Join(silkylogpath, "*")); err != nil {
		return err
	} else {
		for _, file := range files {
			if err := os.Rename(file, filepath.Join(path, filepath.Base(file))); err != nil {
				return err
			}
		}
	}
	if err := os.Remove(silkylogpath); err != nil {
		return err
	}

	app.Log("new site was created under %s, enjoy!", path)
	return nil
}

func newarticle(app *application) error {
	const timeformat = "2006-01-02 15:04:05"
	dates := ""
	var date time.Time

	for {
		dates = readInput("Datetime(YYYY-MM-DD HH:MM:SS) (default: now): ", time.Now().Format(timeformat))
		if t, err := time.Parse(timeformat, dates); err != nil {
			fmt.Println("Invalid datetime format")
		} else {
			date = t
			break
		}
	}
	title := readInput("Title: ", "title")
	slug := readInput("Slug: ", makeSlug(title))
	tags := readInput("Tags: ", "")
	status := readInput("Status(published or draft) (default: draft): ", "draft")
	markup := readInput("Markup (default: .md): ", ".md")
	path := filepath.Join(app.Config.ContentDir, "articles", date.Format("2006"), date.Format("01"), fmt.Sprintf("%02d_%s%s", date.Day(), slug, markup))
	data := fmt.Sprintf(":title: %s\n:tags: %s\n:status: %s\n:posted_at: %s\n:updated_at: %s\n\nhave fun!\n", title, tags, status, dates, dates)
	if err := writeFile(data, path); err != nil {
		return err
	}
	if err := app.openEditor(path); err != nil {
		return errors.New("Failed to start an editor process.")
	}
	return nil
}

func preview(app *application, port int, path string) error {
	if len(path) == 0 {
		return errors.New("empty path")
	}
	addr := fmt.Sprintf(":%v", port)
	app.CompileTemplates()
	fileserver := fileServer(app)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		urlpath := r.URL.Path
		if urlpath == "/preview" {
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			art, err := loadArticle(app, path)
			if err != nil {
				w.Write(([]byte)(err.Error()))
				return
			}
			renderer := newRenderer()
			if err := app.ConvertArticleText(art); err != nil {
				w.Write(([]byte)(err.Error()))
				return
			}
			title := app.Title("Article", H("App", app, "Article", art))
			html, err2 := renderer.RenderPage(app, "article", newViewModel(app, title, art))
			if err2 != nil {
				w.Write(([]byte)(err2.Error()))
				return
			}
			w.Write(([]byte)(html))
		} else {
			fileserver(w, r)
			return
		}
	})
	http.ListenAndServe(addr, nil)
	return nil
}

func fileServer(app *application) func(w http.ResponseWriter, r *http.Request) {
	fileserver := http.StripPrefix("/", http.FileServer(http.Dir(app.Config.OutputDir)))
	return func(w http.ResponseWriter, r *http.Request) {
		w2 := newResponseWriter(w)
		fileserver.ServeHTTP(w2, r)
		if w2.Status == 404 && app.Config.TrimHtml {
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			r.URL.Path = r.URL.Path + ".html"
			fileserver.ServeHTTP(w, r)
		} else {
			w.WriteHeader(w2.Status)
			w.Write(w2.Buf.Bytes())
		}
	}
}

func serve(app *application, port int) error {
	addr := fmt.Sprintf(":%v", port)
	http.HandleFunc("/", fileServer(app))
	http.ListenAndServe(addr, nil)
	return nil
}

type responseWriter struct {
	http.ResponseWriter
	Status int
	Buf    bytes.Buffer
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	var buf bytes.Buffer
	self := &responseWriter{w, 200, buf}
	return self
}

func (res *responseWriter) WriteHeader(status int) {
	res.Status = status
}

func (res *responseWriter) Write(b []byte) (int, error) {
	return res.Buf.Write(b)
}
