package main

import (
	"bytes"
	"github.com/yuin/gopher-lua"
	"html"
	"os/exec"
	"sync"
)

type lStatePool struct {
	m     sync.Mutex
	saved []*lua.LState
}

func (pl *lStatePool) Get() *lua.LState {
	pl.m.Lock()
	defer pl.m.Unlock()
	n := len(pl.saved)
	if n == 0 {
		return pl.New()
	}
	x := pl.saved[n-1]
	pl.saved = pl.saved[0 : n-1]
	return x
}

func (pl *lStatePool) New() *lua.LState {
	L := lua.NewState()
	loadConfig(L)
	return L
}

func (pl *lStatePool) Put(L *lua.LState) {
	pl.m.Lock()
	defer pl.m.Unlock()
	pl.saved = append(pl.saved, L)
}

func (pl *lStatePool) Shutdown() {
	for _, L := range pl.saved {
		L.Close()
	}
}

var luaPool = &lStatePool{
	saved: make([]*lua.LState, 0, 4),
}

func LuaModuleLoader(L *lua.LState) int {
	mod := L.SetFuncs(L.NewTable(), exports)
	//L.SetField(mod, "name", lua.LString("value"))
	L.Push(mod)
	return 1
}

var exports = map[string]lua.LGFunction{
	"runprocessor": luaRunProcessor,
	"htmlescape":   luaHtmlEscape,
	"htmlunescape": luaHtmlUnescape,
	"urlencode":    luaUrlEncode,
	"formatmarkup": luaFormatMarkup,
	"title":        luaTitle,
	"path":         luaPath,
	"url":          luaUrl,
	"fullurl":      luaFullUrl,
	"copyfile":     luaCopyFile,
	"copytree":     luaCopyTree,
	"isdir":        luaIsDir,
	"isfile":       luaIsFile,
	"pathexists":   luaPathExists,
}

func luaRunProcessor(L *lua.LState) int {
	cmdline := []string{}
	text := L.CheckString(-1)
	for i := 1; i < L.GetTop(); i++ {
		cmdline = append(cmdline, L.Get(i).String())
	}
	cmd := exec.Command(cmdline[0], cmdline[1:len(cmdline)]...)
	stdin, err1 := cmd.StdinPipe()
	if err1 != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err1.Error()))
		return 2
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Start(); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	if _, err := stdin.Write(([]byte)(text)); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	stdin.Close()
	if err := cmd.Wait(); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LString(buf.String()))
	return 1
}

func luaHtmlEscape(L *lua.LState) int {
	str := L.CheckString(1)
	L.Push(lua.LString(html.EscapeString(str)))
	return 1
}

func luaHtmlUnescape(L *lua.LState) int {
	str := L.CheckString(1)
	L.Push(lua.LString(html.UnescapeString(str)))
	return 1
}

func luaUrlEncode(L *lua.LState) int {
	str := L.CheckString(1)
	L.Push(lua.LString(urlEncode(str)))
	return 1
}

func luaFormatMarkup(L *lua.LState) int {
	app := appInstance()
	text := L.CheckString(1)
	format := L.CheckString(2)
	html, err := app.convertArticleText(L, text, format)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LString(html))
	return 1
}

func luaMapArg(L *lua.LState, idx int) map[interface{}]interface{} {
	app := appInstance()
	data := luaToGo(L.CheckTable(idx)).(map[interface{}]interface{})
	data["App"] = app
	return data
}

func luaTitle(L *lua.LState) int {
	app := appInstance()
	name := L.CheckString(1)
	data := luaMapArg(L, 2)
	L.Push(lua.LString(app.Title(name, data)))
	return 1
}

func luaPath(L *lua.LState) int {
	app := appInstance()
	name := L.CheckString(1)
	data := luaMapArg(L, 2)
	L.Push(lua.LString(app.Path(name, data)))
	return 1
}

func luaUrl(L *lua.LState) int {
	app := appInstance()
	name := L.CheckString(1)
	data := luaMapArg(L, 2)
	L.Push(lua.LString(app.Url(name, data)))
	return 1
}

func luaFullUrl(L *lua.LState) int {
	app := appInstance()
	name := L.CheckString(1)
	data := luaMapArg(L, 2)
	L.Push(lua.LString(app.FullUrl(name, data)))
	return 1
}

func luaCopyFile(L *lua.LState) int {
	src, dst := L.CheckString(1), L.CheckString(2)
	if err := copyFile(src, dst); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LTrue)
	return 1
}

func luaCopyTree(L *lua.LState) int {
	src, dst := L.CheckString(1), L.CheckString(2)
	if err := copyTree(src, dst); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LTrue)
	return 1
}

func luaIsDir(L *lua.LState) int {
	if isDir(L.CheckString(1)) {
		L.Push(lua.LTrue)
	} else {
		L.Push(lua.LFalse)
	}
	return 1
}

func luaIsFile(L *lua.LState) int {
	if isFile(L.CheckString(1)) {
		L.Push(lua.LTrue)
	} else {
		L.Push(lua.LFalse)
	}
	return 1
}

func luaPathExists(L *lua.LState) int {
	if pathExists(L.CheckString(1)) {
		L.Push(lua.LTrue)
	} else {
		L.Push(lua.LFalse)
	}
	return 1
}
