package main

import (
	"bytes"
	"fmt"
	"github.com/yuin/gopher-lua"
	"html"
	"html/template"
	"io/ioutil"
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type renderer struct {
	m        sync.Mutex
	m2       sync.Mutex
	tplcache map[string]*template.Template
	layouts  map[string]string
}

type viewModel struct {
	L         *lua.LState
	App       *application
	PageTitle string
	Article   *article
	Articles  []*article
	Tag       string
	Year      int
	Month     int
	Start     int
	End       int
	Step      int
	Page      int
	LastPage  int
	IsFirst   bool
	IsLast    bool
	PathData  interface{}
	ListName  string
}

func newViewModel(app *application, title string, art *article) *viewModel {
	return &viewModel{
		App:       app,
		PageTitle: title,
		Article:   art,
	}
}

func (vm *viewModel) SetPage(page, step int, arts []*article, pd interface{}, ln string) bool {
	vm.Start = intMin((page-1)*step, len(arts)-1)
	vm.End = intMin(vm.Start+step, len(arts))
	vm.Page = page
	vm.Articles = arts[vm.Start:vm.End]
	vm.IsLast = vm.End == len(arts)
	vm.IsFirst = page == 1
	vm.LastPage = int(math.Ceil(float64(len(arts)) / float64(step)))
	vm.Step = step
	vm.PathData = pd
	vm.ListName = ln
	return vm.IsLast
}

func (vm *viewModel) Lua(name string, args ...interface{}) template.HTML {
	L := vm.L
	fn := L.Get(lua.GlobalsIndex)
	for _, name := range strings.Split(name, ".") {
		fn = L.GetField(fn, name)
	}
	if fn.Type() == lua.LTFunction {
		L.Push(fn)
		for _, arg := range args {
			lv, ok := arg.(lua.LValue)
			if !ok {
				lv = GoToLua(L, arg)
			}
			L.Push(lv)
		}
		L.Call(len(args), 1)
		return template.HTML(L.Get(-1).String())
	} else {
		return template.HTML(fn.String())
	}
}

func (vm *viewModel) LValue(v interface{}) lua.LValue {
	return GoToLua(vm.L, v)
}

func newRenderer() *renderer {
	return &renderer{
		tplcache: make(map[string]*template.Template),
		layouts:  make(map[string]string),
	}
}

var funcMap = template.FuncMap{
	"raw":      func(h string) template.HTML { return template.HTML(h) },
	"yield":    func() template.HTML { return template.HTML("") },
	"paginate": defaultPagenator,
	"toint":    parseIntMust,
	"htmlescape": func(s string) template.HTML {
		return template.HTML(html.EscapeString(s))
	},
	"htmlunescape": func(s string) string {
		return html.UnescapeString(s)
	},
	"urlencode": func(s string) template.HTML {
		return template.HTML(urlEncode(s))
	},
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
	"mul": func(a, b int) int { return a * b },
	"div": func(a, b int) int { return a / b },
	"mod": func(a, b int) int { return a % b },
	"substr": func(s string, i, j int) string {
		if i < 0 {
			i = len(s) + i
		}
		if j < 0 {
			j = len(s) + j + 1
		}
		i = intMax(intMin(i, len(s)-1), 0)
		j = intMax(intMin(j, len(s)), 0)
		return s[i:j]
	},
	"H": H,
}

func (rd *renderer) loadTemplate(path string) error {
	if _, ok := rd.tplcache[path]; ok {
		return nil
	}
	bts, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	tpl, err2 := template.New("").Funcs(funcMap).Parse(string(bts))
	if err2 != nil {
		return err2
	}
	rd.tplcache[path] = tpl
	return nil
}

func (rd *renderer) Render(app *application, path string, data *viewModel) (string, error) {
	L := luaPool.Get()
	defer luaPool.Put(L)
	rd.m.Lock()
	defer rd.m.Unlock()
	data.L = L
	tpl, cok := rd.tplcache[path]
	if !cok {
		if err := rd.loadTemplate(path); err != nil {
			return "", err
		}
		tpl = rd.tplcache[path]
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (rd *renderer) RenderType(app *application, name string, typ string, data *viewModel) (string, error) {
	L := luaPool.Get()
	defer luaPool.Put(L)
	data.L = L
	themebase := filepath.Join(app.Config.ThemeDir, app.Config.Theme)
	path := filepath.Join(themebase, typ, name)
	return rd.Render(app, path, data)
}

func (rd *renderer) RenderPage(app *application, name string, data *viewModel) (string, error) {
	L := luaPool.Get()
	defer luaPool.Put(L)
	rd.m.Lock()
	defer rd.m.Unlock()
	data.L = L
	themebase := filepath.Join(app.Config.ThemeDir, app.Config.Theme)
	path := name
	if !pathExists(path) {
		path = filepath.Join(themebase, "pages", name+".html")
	}
	tpl, cok := rd.tplcache[path]
	layout := rd.layouts[path]
	if !cok {
		laypat := regexp.MustCompile(`{{/\*\s*layout:\s*([^\s]+)\s*\*/}}`)
		bts, err := ioutil.ReadFile(path)
		if err != nil {
			return "", err
		}
		matches := laypat.FindAllSubmatch(bts, -1)
		rd.layouts[path] = ""
		if len(matches) > 0 {
			rd.layouts[path] = string(matches[0][1])
			path := filepath.Join(themebase, "layouts", rd.layouts[path]+".html")
			if err := rd.loadTemplate(path); err != nil {
				return "", err
			}
		}
		if err := rd.loadTemplate(path); err != nil {
			return "", err
		}
		tpl = rd.tplcache[path]
		layout = rd.layouts[path]
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}

	if len(layout) > 0 {
		layoutpath := filepath.Join(themebase, "layouts", rd.layouts[path]+".html")
		laytoutpl, _ := rd.tplcache[layoutpath].Clone()
		laytoutpl.Funcs(template.FuncMap{
			"yield": func() template.HTML {
				return template.HTML(buf.String())
			},
		})
		var w bytes.Buffer
		if err := laytoutpl.Execute(&w, data); err != nil {
			return "", err
		}
		return w.String(), nil
	} else {
		return buf.String(), nil
	}
}

func defaultPagenator(vm *viewModel, anchor string) template.HTML {
	pd := vm.PathData.(map[interface{}]interface{})
	pagelink := func(page int) string {
		pd["Page"] = page
		return vm.App.Url(vm.ListName, pd)
	}
	page := vm.Page
	maxpage := vm.LastPage
	if page > maxpage {
		page = 1
	}
	start, end := intMax(page-4, 1), intMin(page+4, maxpage)
	tpl := []string{"<nav class=\"paging\"><ul>"}
	if (page - 1) < 1 {
		tpl = append(tpl, "<li class=\"previous-off\">&laquo;Previous</li>")
	} else {
		tpl = append(tpl, fmt.Sprintf("<li class=\"previous\"><a href=\"%s\" rel=\"prev\" class=\"%s\">&laquo;Previous</a></li>", pagelink(page-1), anchor))
	}
	if start != 1 {
		tpl = append(tpl, fmt.Sprintf("<li><a href=\"%s\" class=\"%s\">1</a></li>", pagelink(1), anchor))
	}
	if start > 2 {
		tpl = append(tpl, "<li>&nbsp;&nbsp;.......&nbsp;&nbsp;</li>")
	}
	for i := start; i <= end; i++ {
		if i == page {
			tpl = append(tpl, fmt.Sprintf("<li class=\"active\">%d</li>", i))
		} else {
			tpl = append(tpl, fmt.Sprintf("<li><a href=\"%s\" class=\"%s\">%d</a></li>", pagelink(i), anchor, i))
		}
	}
	if end < (maxpage - 1) {
		tpl = append(tpl, "<li>&nbsp;&nbsp;......&nbsp;&nbsp;</li>")
	}
	if end != maxpage {
		tpl = append(tpl, fmt.Sprintf("<li><a href=\"%s\" class=\"%s\">%d</a></li>", pagelink(maxpage), anchor, maxpage))
	}
	if (page + 1) > maxpage {
		tpl = append(tpl, "<li class=\"next-off\">Next&raquo;</li>")
	} else {
		tpl = append(tpl, fmt.Sprintf("<li class=\"next\"><a href=\"%s\" rel=\"next\" class=\"%s\">Next&raquo;</a></li>", pagelink(page+1), anchor))
	}
	tpl = append(tpl, "</ul></nav>")
	return template.HTML(strings.Join(tpl, ""))
}
