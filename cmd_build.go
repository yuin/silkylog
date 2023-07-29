package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"text/template"
	"time"
)

func buildTemplate(app *application, renderer *renderer, tpl *template.Template, dir, counter string) error {
	basedir := filepath.Join(app.Config.ThemeDir, app.Config.Theme, dir)
	lst, err := os.ReadDir(basedir)
	if err != nil {
		return err
	}
	viewmodel := newViewModel(app, "", nil)
	for _, item := range lst {
		app.Debug("file(%v): %v", dir, item.Name())
		app.Stats.Inc(counter)
		txt, err := renderer.RenderType(app, item.Name(), dir, viewmodel)
		if err != nil {
			return fmt.Errorf("%v/%v : %w", dir, item.Name(), err)
		}
		path, err2 := execTemplate(tpl, map[string]string{"Name": item.Name()})
		if err2 != nil {
			return fmt.Errorf("%v/%v : %w", dir, item.Name(), err2)
		}
		if err := writeFile(txt, filepath.Join(app.Config.OutputDir, path)); err != nil {
			return fmt.Errorf("%v/%v : %w", dir, item.Name(), err)
		}
	}
	return nil
}

func buildList(app *application, renderer *renderer, lst, name string, arts []*article,
	dc func() map[any]any, vu func(*viewModel)) error {
	brek := false
	for page := 1; !brek; page++ {
		app.Stats.Inc(name)
		data := dc()
		data["Page"] = page
		path := app.Path(name, data)
		app.Debug("article list: %v", path)
		title := app.Title(name, data)
		vm := newViewModel(app, title, nil)
		vu(vm)
		step := app.Config.Pagination2
		if lst == "list1" {
			step = app.Config.Pagination1
		}
		brek = vm.SetPage(page, step, arts, data, name)
		html, err := renderer.RenderPage(app, lst, vm)
		if err != nil {
			return fmt.Errorf("%v: %w", path, err)
		}
		if err := writeFile(html, filepath.Join(app.Config.OutputDir, path)); err != nil {
			return fmt.Errorf("%v: %w", path, err)
		}

		if page == 1 {
			data["Page"] = 0
			indexpath := app.Path(name, data)
			if err := copyFile(filepath.Join(app.Config.OutputDir, path),
				filepath.Join(app.Config.OutputDir, indexpath)); err != nil {
				return fmt.Errorf("%v: %w", path, err)
			}
		}
	}
	return nil
}

func buildArticle(app *application, renderer *renderer, art *article, errch chan error) {
	app.Stats.Inc("Article")
	app.Debug("article: %v", art.FilePath)
	if err := app.ConvertArticleText(art); err != nil {
		errch <- errors.New(art.FilePath + ": " + err.Error())
		return
	}
	title := app.Title("Article", H("App", app, "Article", art))
	html, err2 := renderer.RenderPage(app, "article", newViewModel(app, title, art))
	if err2 != nil {
		errch <- errors.New(art.FilePath + ": " + err2.Error())
		return
	}
	if err := writeFile(html, filepath.Join(app.Config.OutputDir, app.Path("Article", art))); err != nil {
		errch <- errors.New(art.FilePath + ": " + err.Error())
		return
	}

}

func build(app *application) error {
	started := time.Now()
	app.Log("build start")
	var err error
	err = app.CompileTemplates()
	if err != nil {
		return err
	}
	err = app.LoadArticles("published")
	if err != nil {
		return err
	}
	renderer := newRenderer()
	sem := make(chan int, app.Config.NumThreads)
	var wg sync.WaitGroup

	// articles
	{
		errch := make(chan error)
		quit := make(chan int)
		go func() {
			for {
				select {
				case err := <-errch:
					exitApplication(err.Error(), 1)
				case <-quit:
					return
				}
			}
		}()
		for _, art := range app.Articles {
			wg.Add(1)
			go func(art *article) {
				defer func() {
					wg.Done()
					<-sem
				}()
				sem <- 1
				buildArticle(app, renderer, art, errch)
			}(art)
		}
		wg.Wait()
		quit <- 1
		close(errch)
		app.Log("%d articles", app.Stats.Get("Article"))
	}

	// index
	if err := buildList(app, renderer, "list1", "Index", app.Articles,
		func() map[any]any {
			return H("App", app)
		},
		func(vm *viewModel) {
		}); err != nil {
		return err
	}
	app.Log("%d index pages", app.Stats.Get("Index"))

	//tag
	for tag, arts := range app.Tags {
		if err := buildList(app, renderer, "list2", "Tag", arts,
			func() map[any]any {
				return H("App", app, "Tag", tag)
			},
			func(vm *viewModel) {
				vm.Tag = tag
			}); err != nil {
			return err
		}
	}
	app.Log("%d tag pages", app.Stats.Get("Tag"))

	//annual
	for syear, arts := range app.Years {
		year := parseIntMust(syear)

		if err := buildList(app, renderer, "list2", "Annual", arts,
			func() map[any]any {
				return H("App", app, "Year", year)
			},
			func(vm *viewModel) {
				vm.Year = year
			}); err != nil {
			return err
		}
	}
	app.Log("%d annual archive pages", app.Stats.Get("Annual"))

	//monthly
	for smonth, arts := range app.Months {
		year := parseIntMust(smonth[0:4])
		month := parseIntMust(smonth[4:6])

		if err := buildList(app, renderer, "list2", "Monthly", arts,
			func() map[any]any {
				return H("App", app, "Year", year, "Month", month)
			},
			func(vm *viewModel) {
				vm.Year = year
				vm.Month = month
			}); err != nil {
			return err
		}
	}
	app.Log("%d monthly archive pages", app.Stats.Get("Monthly"))

	// include
	if err := buildTemplate(app, renderer, app.PathTemplate("Include"), "include", "Include"); err != nil {
		return err
	}
	app.Log("%d include pages", app.Stats.Get("Include"))

	// feeds
	if err := buildTemplate(app, renderer, app.PathTemplate("Feed"), "feeds", "Feed"); err != nil {
		return err
	}
	app.Log("%d feeds", app.Stats.Get("Feed"))

	// extras
	if err := copyExtras(app, renderer, app.Config.ExtraFiles,
		filepath.Join(app.Config.ContentDir, "extras")); err != nil {
		return err
	}
	if err := copyExtras(app, renderer, app.Config.ThemeConfig.ExtraFiles,
		filepath.Join(app.Config.ThemeDir, app.Config.Theme, "extras")); err != nil {
		return err
	}
	app.Log("%d extra files", app.Stats.Get("Extra"))

	app.Log("-----------------------------")
	app.Log("build: OK(%v)", time.Since(started))
	app.Log("-----------------------------")
	return nil
}

func copyExtras(app *application, renderer *renderer, extras []extraFile, sdir string) error {
	done := make(map[string]int)
	odir := app.Config.OutputDir
	for _, f := range extras {
		path := filepath.Join(sdir, f.Src)
		app.Debug("copy extras start: %v", path)
		matches, err := filepath.Glob(path)
		if err != nil {
			return fmt.Errorf("%v: %w", path, err)
		}
		for _, m := range matches {
			app.Stats.Inc("Extra")
			if _, ok := done[m]; !ok {
				done[m] = 1
			} else {
				continue
			}
			dst := filepath.Join(odir, f.Dst, filepath.Base(m))
			app.Debug("copy extras: %v -> %v", m, dst)
			if f.Template && isFile(m) {
				txt, err := renderer.RenderPage(app, m, newViewModel(app, "", nil))
				if err != nil {
					return fmt.Errorf("%v: %w", m, err)
				}
				if err := writeFile(txt, dst); err != nil {
					return fmt.Errorf("%v: %w", m, err)
				}
			} else {
				if isDir(m) {
					if err := copyTree(m, dst); err != nil {
						return fmt.Errorf("%v: %w", m, err)
					}
				} else {
					if err := copyFile(m, dst); err != nil {
						return fmt.Errorf("%v: %w", m, err)
					}
				}
			}
		}
	}
	return nil
}
