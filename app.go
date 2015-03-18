package main

import (
	"errors"
	"fmt"
	"github.com/russross/blackfriday"
	"github.com/yuin/gopher-lua"
	htemplate "html/template"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"
)

type stats struct {
	m       sync.Mutex
	counter map[string]int
}

func newStats() *stats {
	return &stats{
		counter: map[string]int{},
	}
}

func (sts *stats) Inc(name string) {
	sts.m.Lock()
	defer sts.m.Unlock()
	sts.counter[name] = sts.counter[name] + 1
}

func (sts *stats) Get(name string) int {
	sts.m.Lock()
	defer sts.m.Unlock()
	return sts.counter[name]
}

type logger func(app *application, format string, args ...interface{})

var _app *application

func appInstance() *application {
	return _app
}

type application struct {
	Config   *config
	Stats    *stats
	Articles articles
	Tags     articleMap
	Years    articleMap
	Months   articleMap

	Logger func(*application, string, ...interface{})

	m        sync.Mutex
	tplcahe  map[string]*template.Template
	htplcahe map[string]*htemplate.Template
}

func newApp() *application {
	return &application{
		Stats:    newStats(),
		Articles: []*article{},
		Tags:     make(map[string][]*article),
		Years:    make(map[string][]*article),
		Months:   make(map[string][]*article),

		Logger: func(app *application, format string, args ...interface{}) {
			nowstr := time.Now().Format(time.RFC822)
			if len(args) > 0 {
				fmt.Printf(nowstr+"\t"+format, args...)
				fmt.Print("\n")
			} else {
				fmt.Println(nowstr + "\t" + format)
			}
		},

		tplcahe:  make(map[string]*template.Template),
		htplcahe: make(map[string]*htemplate.Template),
	}
}

func (app *application) Log(format string, args ...interface{}) {
	app.m.Lock()
	defer app.m.Unlock()
	app.Logger(app, format, args...)
}

func (app *application) Debug(format string, args ...interface{}) {
	app.m.Lock()
	defer app.m.Unlock()
	if app.Config.Debug {
		app.Logger(app, format, args...)
	}
}

func (app *application) ConvertArticleText(art *article) error {
	if len(art.BodyHtml) > 0 {
		return nil
	}
	art.m.Lock()
	defer art.m.Unlock()
	L := luaPool.Get()
	defer luaPool.Put(L)
	html, err := app.convertArticleText(L, art.BodyText, art.Format)
	if err != nil {
		return err
	}
	art.BodyHtml = html
	return nil
}

func (app *application) convertArticleText(L *lua.LState, markup, format string) (string, error) {
	processor := L.GetField(L.GetField(L.GetGlobal("CONFIG"), "markup_processors"), format)
	if processor == lua.LNil {
		return "", errors.New("unknown markup format: " + format)
	}
	if fn, ok := processor.(*lua.LFunction); ok {
		L.Push(fn)
		L.Push(lua.LString(markup))
		if err := L.PCall(1, 1, nil); err != nil {
			return "", err
		}
		return luaPop(L).String(), nil
	}
	_opts, ok := processor.(*lua.LTable)
	if !ok {
		return "", errors.New("markup_processors must be a function or table")
	}
    opts := luaToGo(_opts).(map[interface{}]interface{})

	switch format {
	case ".md":
		lhtmlopts, _ := opts["htmlopts"]
		lexts, _ := opts["exts"]
		htmlopts, err1 := strListToOption("htmlopts", lhtmlopts, 0, strToBfHtmlOpts)
		if err1 != nil {
			return "", err1
		}
		exts, err2 := strListToOption("exts", lexts, 0, strToBfExts)
		if err2 != nil {
			return "", err2
		}
		renderer := blackfriday.HtmlRenderer(htmlopts, "", "")
		return string(blackfriday.Markdown(([]byte)(markup), renderer, exts)), nil
	}
	return "", errors.New("no builtin processors found for '" + format + "'")
}

func (app *application) TitleTemplate(name string) *htemplate.Template {
	app.m.Lock()
	defer app.m.Unlock()
	tpl, ok := app.htplcahe[name]
	if !ok {
		exitApplication(name+" is invalid title", 1)
	}
	return tpl
}

func (app *application) PathTemplate(name string) *template.Template {
	app.m.Lock()
	defer app.m.Unlock()
	tpl, ok := app.tplcahe[name]
	if !ok {
		exitApplication(name+" is invalid url_path", 1)
	}
	return tpl
}

func (app *application) Title(name string, data interface{}) string {
	title, err := execHtemplate(app.TitleTemplate(name), data)
	if err != nil {
		exitApplication(err.Error(), 1)
	}
	return title
}

func (app *application) Path(name string, data interface{}) string {
	path, err := execTemplate(app.PathTemplate(name), data)
	if err != nil {
		exitApplication(err.Error(), 1)
	}
	return path
}

func (app *application) relUrl(name string, data interface{}) string {
	url := strings.TrimSuffix(app.Path(name, data), "index.html")
	if app.Config.TrimHtml {
		return strings.TrimSuffix(url, ".html")
	}
	return urlEncode(url)
}

func (app *application) Url(name string, data interface{}) string {
	return "/" + app.relUrl(name, data)
}

func (app *application) FullUrl(name string, data interface{}) string {
	return app.Config.SiteUrl + app.relUrl(name, data)
}

func (app *application) CompileTemplates() (err error) {
	defer func() {
		v := recover()
		if v != nil {
			err = v.(error)
		}
	}()
	rv := reflect.ValueOf(app.Config).Elem()
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		name := rt.Field(i).Name
		if strings.HasSuffix(name, "UrlPath") {
			app.tplcahe[strings.TrimSuffix(name, "UrlPath")] = template.Must(template.New("").Parse(rv.Field(i).String()))
		} else if strings.HasSuffix(name, "Title") {
			app.htplcahe[strings.TrimSuffix(name, "Title")] = htemplate.Must(htemplate.New("").Parse(rv.Field(i).String()))
		}
	}
	return
}

func (app *application) LoadArticles() error {
	c := app.Config
	basedir := filepath.Join(c.ContentDir, "articles")
	lastpath := ""
	err := filepath.Walk(basedir, func(path string, info os.FileInfo, err error) error {
		lastpath = path
		basename := filepath.Base(path)
		if info.IsDir() {
			if strings.HasPrefix(basename, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(basename, ".") {
			return nil
		}
		art, err := loadArticle(app, path)
		if err != nil {
			return err
		}
		art.PermlinkPath = app.Url("Article", art)
		art.PermlinkUrl = app.Config.SiteUrl + strings.TrimLeft(app.Url("Article", art), "/")
		app.Articles = append(app.Articles, art)
		return nil

	})
	if err != nil {
		return errors.New(fmt.Sprintf("error in %v:\n  %v", lastpath, err.Error()))
	}
	sort.Sort(app.Articles)

	for _, art := range app.Articles {
		for _, tag := range art.Tags {
			app.Tags.Add(tag, art)
		}
		iyear, imonth := art.PostedAt.Year(), art.PostedAt.Month()
		syear, smonth := fmt.Sprintf("%04d", iyear), fmt.Sprintf("%04d%02d", iyear, imonth)
		app.Years.Add(syear, art)
		app.Months.Add(smonth, art)
	}
	return nil
}

func (app *application) openEditor(path string) error {
	edcopy := make([]string, len(app.Config.Editor)+1)
	copy(edcopy, app.Config.Editor)
	edcopy[len(edcopy)-1] = path
	cmd := exec.Command(edcopy[0], edcopy[1:len(edcopy)]...)
	return cmd.Start()
}

var strToBfHtmlOpts = map[string]int{
	"HTML_SKIP_HTML":                 blackfriday.HTML_SKIP_HTML,
	"HTML_SKIP_STYLE":                blackfriday.HTML_SKIP_STYLE,
	"HTML_SKIP_IMAGES":               blackfriday.HTML_SKIP_IMAGES,
	"HTML_SKIP_LINKS":                blackfriday.HTML_SKIP_LINKS,
	"HTML_SAFELINK":                  blackfriday.HTML_SAFELINK,
	"HTML_NOFOLLOW_LINKS":            blackfriday.HTML_NOFOLLOW_LINKS,
	"HTML_HREF_TARGET_BLANK":         blackfriday.HTML_HREF_TARGET_BLANK,
	"HTML_TOC":                       blackfriday.HTML_TOC,
	"HTML_OMIT_CONTENTS":             blackfriday.HTML_OMIT_CONTENTS,
	"HTML_COMPLETE_PAGE":             blackfriday.HTML_COMPLETE_PAGE,
	"HTML_USE_XHTML":                 blackfriday.HTML_USE_XHTML,
	"HTML_USE_SMARTYPANTS":           blackfriday.HTML_USE_SMARTYPANTS,
	"HTML_SMARTYPANTS_FRACTIONS":     blackfriday.HTML_SMARTYPANTS_FRACTIONS,
	"HTML_SMARTYPANTS_LATEX_DASHES":  blackfriday.HTML_SMARTYPANTS_LATEX_DASHES,
	"HTML_SMARTYPANTS_ANGLED_QUOTES": blackfriday.HTML_SMARTYPANTS_ANGLED_QUOTES,
	"HTML_FOOTNOTE_RETURN_LINKS":     blackfriday.HTML_FOOTNOTE_RETURN_LINKS,
}

var strToBfExts = map[string]int{
	"EXTENSION_NO_INTRA_EMPHASIS":          blackfriday.EXTENSION_NO_INTRA_EMPHASIS,
	"EXTENSION_TABLES":                     blackfriday.EXTENSION_TABLES,
	"EXTENSION_FENCED_CODE":                blackfriday.EXTENSION_FENCED_CODE,
	"EXTENSION_AUTOLINK":                   blackfriday.EXTENSION_AUTOLINK,
	"EXTENSION_STRIKETHROUGH":              blackfriday.EXTENSION_STRIKETHROUGH,
	"EXTENSION_LAX_HTML_BLOCKS":            blackfriday.EXTENSION_LAX_HTML_BLOCKS,
	"EXTENSION_SPACE_HEADERS":              blackfriday.EXTENSION_SPACE_HEADERS,
	"EXTENSION_HARD_LINE_BREAK":            blackfriday.EXTENSION_HARD_LINE_BREAK,
	"EXTENSION_TAB_SIZE_EIGHT":             blackfriday.EXTENSION_TAB_SIZE_EIGHT,
	"EXTENSION_FOOTNOTES":                  blackfriday.EXTENSION_FOOTNOTES,
	"EXTENSION_NO_EMPTY_LINE_BEFORE_BLOCK": blackfriday.EXTENSION_NO_EMPTY_LINE_BEFORE_BLOCK,
	"EXTENSION_HEADER_IDS":                 blackfriday.EXTENSION_HEADER_IDS,
	"EXTENSION_TITLEBLOCK":                 blackfriday.EXTENSION_TITLEBLOCK,
	"EXTENSION_AUTO_HEADER_IDS":            blackfriday.EXTENSION_AUTO_HEADER_IDS,
}
