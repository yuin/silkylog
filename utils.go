package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/yuin/gopher-lua"
	htemplate "html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

func parseIntMust(s string) int {
	i64, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return int(i64)
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func readInput(prompt, defaults string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	s, err := reader.ReadString('\n')
	if err != nil || len(strings.TrimSpace(s)) == 0 {
		return defaults
	}
	return strings.TrimSpace(s)
}

func toCapCase(s string) string {
	return strings.ToUpper(string(s[0])) + regexp.MustCompile(`_([a-z])`).ReplaceAllStringFunc(s[1:len(s)], func(s string) string { return strings.ToUpper(s[1:len(s)]) })
}

func urlEncode(s string) string {
	u := &url.URL{Path: s}
	return u.String()
}

func makeSlug(s string) string {
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, "-")
	s = regexp.MustCompile(`[\\/:\*\?"<>\|]`).ReplaceAllString(s, "")
	return strings.ToLower(s)
}

func H(args ...interface{}) map[interface{}]interface{} {
	ret := make(map[interface{}]interface{})
	for i := 0; i < len(args); i += 2 {
		ret[args[i]] = args[i+1]
	}
	return ret
}

type fileType int

const (
	ftFile fileType = iota
	ftDir
	ftLink
	ftNotExists
	ftOther
)

func pathFilePath(path string) fileType {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ftNotExists
		}
		return ftOther
	}

	if (fi.Mode() & os.ModeSymlink) == os.ModeSymlink {
		return ftLink
	}
	if fi.IsDir() {
		return ftDir
	}
	return ftFile
}

func isDir(path string) bool { return pathFilePath(path) == ftDir }

func isFile(path string) bool { return pathFilePath(path) == ftFile }

func pathExists(path string) bool { return pathFilePath(path) != ftNotExists }

func ensureDirExists(path string) error {
	dir := filepath.Dir(path)
	if !pathExists(dir) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeFile(data string, path string) error {
	if err := ensureDirExists(path); err != nil {
		return err
	}
	if err := ioutil.WriteFile(path, ([]byte)(data), 0755); err != nil {
		return err
	}
	return nil
}

func copyFile(source, dest string) error {
	if err := ensureDirExists(dest); err != nil {
		return err
	}
	data, err := ioutil.ReadFile(source)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dest, data, 0644)
}

// source = src/dir, dest = dest/hoge
// copy src/dir/{a,b,c} -> dst/hoge/{a,b,c}
func copyTree(source string, dest string) error {
	if !pathExists(dest) {
		err := os.MkdirAll(dest, 0755)
		if err != nil {
			return err
		}
	}
	if !isDir(source) {
		return errors.New("copyTree: " + source + " is not a directory")
	}
	if !isDir(dest) {
		return errors.New("copyTree: " + dest + " is not a directory")
	}

	lst, err := ioutil.ReadDir(source)
	if err != nil {
		return err
	}

	for _, item := range lst {
		srcfile := filepath.Join(source, item.Name())
		destfile := filepath.Join(dest, item.Name())
		var err error
		if isFile(srcfile) {
			err = copyFile(srcfile, destfile)
		} else if isDir(srcfile) {
			err = copyTree(srcfile, destfile)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func download(url, path string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return errors.New("failed to download '" + url + "'")
	}
	body, err1 := ioutil.ReadAll(response.Body)
	if err1 != nil {
		return err1
	}
	if err := writeFile(string(body), path); err != nil {
		return err
	}
	return nil
}

func unzip(zipfile, dest string) error {
	if !pathExists(dest) {
		err := os.MkdirAll(dest, 0755)
		if err != nil {
			return err
		}
	}

	r, err := zip.OpenReader(zipfile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			f, err := os.OpenFile(
				path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func execTemplate(t *template.Template, any interface{}) (string, error) {
	var b bytes.Buffer
	if err := t.Execute(&b, any); err != nil {
		return "", err
	}
	return b.String(), nil
}

func execHtemplate(t *htemplate.Template, any interface{}) (string, error) {
	var b bytes.Buffer
	if err := t.Execute(&b, any); err != nil {
		return "", err
	}
	return b.String(), nil
}

func exitApplication(msg string, code int) {
	out := os.Stdout
	if code != 0 {
		out = os.Stderr
	}
	fmt.Fprint(out, msg)
	fmt.Fprint(out, "\n")
	os.Exit(code)
}

func luaToGoStruct(tbl *lua.LTable, st interface{}) error {
	mp, ok := luaToGo(tbl).(map[interface{}]interface{})
	if !ok {
		return errors.New("arguments #1 must be a table")
	}
	return mapstructure.Decode(mp, st)
}

func luaToGo(lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LString:
		return string(v)
	case lua.LNumber:
		return float64(v)
	case *lua.LTable:
		if v.MaxN() == 0 { // table
			ret := make(map[interface{}]interface{})
			v.ForEach(func(key, value lua.LValue) {
				ret[luaToGo(key)] = luaToGo(value)
			})
			return ret
		} else { // array
			ret := make([]interface{}, 0, v.MaxN())
			v.ForEach(func(key, value lua.LValue) {
				ret = append(ret, luaToGo(value))
			})
			return ret
		}
	default:
		return v
	}
}

func goToLua(L *lua.LState, v interface{}) lua.LValue {
	var art article
	at := reflect.TypeOf(art)
	rv := reflect.ValueOf(v)
	kind := rv.Kind()
	switch {
	case kind == 0:
		return lua.LNil
	case reflect.Int >= kind && kind <= reflect.Int64:
		return lua.LNumber(rv.Int())
	case reflect.Uint >= kind && kind <= reflect.Uintptr:
		return lua.LNumber(rv.Uint())
	case reflect.Float32 >= kind && kind <= reflect.Float64:
		return lua.LNumber(rv.Float())
	case kind == reflect.String:
		return lua.LString(rv.String())
	case kind == reflect.Bool:
		if rv.Bool() {
			return lua.LTrue
		}
		return lua.LFalse
	case kind == reflect.Ptr && rv.Elem().Type() == at:
		return rv.Interface().(*article).ToLua(L)
	case kind == reflect.Slice:
		tb := L.NewTable()
		for i := 0; i < rv.Len(); i++ {
			tb.Append(goToLua(L, rv.Index(i).Interface()))
		}
		return tb
	case kind == reflect.Map:
		tb := L.NewTable()
		for _, key := range rv.MapKeys() {
			tb.RawSet(goToLua(L, key.Interface()), goToLua(L, rv.MapIndex(key).Interface()))
		}
		return tb
	default:
		return lua.LNil
	}
}

func luaPop(L *lua.LState) lua.LValue {
	lv := L.Get(-1)
	L.Pop(1)
	return lv
}

func timeToLuaTable(L *lua.LState, t time.Time) *lua.LTable {
	tb := L.NewTable()
	tb.RawSetH(lua.LString("year"), lua.LNumber(t.Year()))
	tb.RawSetH(lua.LString("month"), lua.LNumber(t.Month()))
	tb.RawSetH(lua.LString("day"), lua.LNumber(t.Day()))
	tb.RawSetH(lua.LString("hour"), lua.LNumber(t.Hour()))
	tb.RawSetH(lua.LString("minute"), lua.LNumber(t.Minute()))
	tb.RawSetH(lua.LString("second"), lua.LNumber(t.Second()))
	tzname, tzoffset := t.Zone()
	tb.RawSetH(lua.LString("tzname"), lua.LString(tzname))
	tb.RawSetH(lua.LString("tzoffset"), lua.LNumber(tzoffset))
	return tb
}

func strListToOption(name string, v interface{}, init int, options map[string]int) (int, error) {
	if fmt.Sprint(v) == "<nil>" {
		return init, nil
	}

	tbl, ok := v.([]interface{})
	if !ok {
		return init, errors.New(name + " must be a list of string")
	}
	for _, opt := range tbl {
		s, sok := opt.(string)
		if !sok {
			return init, errors.New(name + " must be a list of string")
		}
		val, ok := options[s]
		if !ok {
			return init, errors.New("invalid option '" + s + "' for " + name)
		}
		init |= val
	}
	return init, nil
}

//
