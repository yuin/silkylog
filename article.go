package main

import (
	"errors"
	"github.com/yuin/gopher-lua"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type article struct {
	m         sync.Mutex
	FilePath  string
	Format    string
	Title     string
	Slug      string
	BodyText  string
	BodyHtml  string
	Status    string
	Tags      []string
	PostedAt  time.Time
	UpdatedAt time.Time

	PermlinkPath string
	PermlinkUrl  string
}

type articles []*article

func (a articles) Len() int { return len(a) }

func (a articles) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a articles) Less(i, j int) bool { return a[i].PostedAt.Unix() > a[j].PostedAt.Unix() }

func (a articles) SubList(i, j int) articles {
	if i < 0 {
		i = len(a) + i
	}
	if j < 0 {
		j = len(a) + j + 1
	}
	i = intMax(intMin(i, len(a)-1), 0)
	j = intMax(intMin(j, len(a)), 0)
	return articles(a[i:j])
}

type articleMap map[string][]*article

func (am articleMap) Add(key string, art *article) {
	_, ok := am[key]
	if !ok {
		am[key] = make([]*article, 0, 10)
	}
	am[key] = append(am[key], art)
}

func (am articleMap) SortedMapKeys(reverse bool) []string {
	keys := []string{}
	for key, _ := range am {
		keys = append(keys, key)
	}
	if !reverse {
		sort.Strings(keys)
	} else {
		sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	}
	return keys
}

func loadArticle(app *application, path string) (*article, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	art := &article{}
	art.FilePath = path
	art.Format = filepath.Ext(path)
	art.Tags = []string{}
	buf := []string{}
	btext, err := ioutil.ReadAll(fp)
	if err != nil {
		return nil, err
	}
	text := string(btext)
	body := false

	for _, line := range strings.Split(text, "\n") {
		var err error
		if len(line) == 0 && !body {
			body = true
			continue
		}
		if body {
			buf = append(buf, string(line))
		} else {
			err = parseArticleHeader(app, art, string(line))
		}
		if err != nil {
			return nil, err
		}
	}
	art.BodyText = strings.Join(buf, "\n")
	art.PermlinkPath = app.Url("Article", art)
	art.PermlinkUrl = app.Config.SiteUrl + strings.TrimLeft(app.Url("Article", art), "/")
	return art, nil
}

func parseArticleHeader(app *application, art *article, line string) error {
	const timeformat = "2006-01-02 15:04:05"
	names := strings.Split(line, ":")
	if len(names) < 3 {
		return errors.New("invalid header: " + line)
	}
	name := names[1]
	value := strings.TrimSpace(strings.Join(names[2:len(names)], ":"))
	switch name {
	case "title":
		art.Title = value
	case "slug":
		art.Slug = value
	case "status":
		if value != "draft" && value != "published" {
			return errors.New("invalid status: " + value)
		}
		art.Status = value
	case "tags":
		for _, tag := range strings.Split(value, ",") {
			art.Tags = append(art.Tags, strings.TrimSpace(tag))
		}
	case "posted_at":
		t, err := time.ParseInLocation(timeformat, value, app.Config.Location())
		if err != nil {
			return errors.New("invalid posted_at: " + err.Error())
		}
		art.PostedAt = t
	case "updated_at":
		t, err := time.ParseInLocation(timeformat, value, app.Config.Location())
		if err != nil {
			return errors.New("invalid updated_at:" + err.Error())
		}
		art.UpdatedAt = t
	}

	if len(art.Slug) == 0 {
		basename := filepath.Base(art.FilePath)
		match := regexp.MustCompile(`(\d+_)(.*)\.(\w+)`).FindStringSubmatch(basename)
		if len(match) > 0 {
			art.Slug = match[2]
		} else {
			art.Slug = art.Title
		}
	}

	if art.UpdatedAt.Year() == 1 {
		art.UpdatedAt = art.PostedAt
	}

	return nil
}

func (art *article) ToLua(L *lua.LState) *lua.LTable {
	tb := L.NewTable()
	tb.RawSetString("file_path", lua.LString(art.FilePath))
	tb.RawSetString("format", lua.LString(art.Format))
	tb.RawSetString("title", lua.LString(art.Title))
	tb.RawSetString("slug", lua.LString(art.Slug))
	tb.RawSetString("body_text", lua.LString(art.BodyText))
	tb.RawSetString("body_html", lua.LString(art.BodyHtml))
	tb.RawSetString("status", lua.LString(art.Status))
	tags := L.NewTable()
	for _, tag := range art.Tags {
		tags.Append(lua.LString(tag))
	}
	tb.RawSetString("tags", tags)
	tb.RawSetString("posted_at", timeToLuaTable(L, art.PostedAt))
	tb.RawSetString("updated_at", timeToLuaTable(L, art.UpdatedAt))
	tb.RawSetString("permlink_path", lua.LString(art.PermlinkPath))
	tb.RawSetString("permlink_url", lua.LString(art.PermlinkUrl))
	return tb
}
