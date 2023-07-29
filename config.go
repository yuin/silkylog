package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

type config struct {
	Debug       bool
	SiteUrl     string
	Editor      []string
	NumThreads  int
	Timezone    string
	Theme       string
	Pagination1 int
	Pagination2 int
	TrimHTML    bool

	Params map[string]interface{}

	TopUrlPath string

	ArticleUrlPath string
	ArticleTitle   string

	IndexUrlPath string
	IndexTitle   string

	TagUrlPath string
	TagTitle   string

	AnnualUrlPath string
	AnnualTitle   string

	MonthlyUrlPath string
	MonthlyTitle   string

	IncludeUrlPath string
	FeedUrlPath    string
	FileUrlPath    string

	ContentDir string
	ThemeDir   string
	OutputDir  string
	ExtraFiles []extraFile
	Clean      []string

	MarkupProcessors map[string]interface{}

	ThemeConfig *config

	location *time.Location
}

type extraFile struct {
	Src      string `mapstructure:"src"`
	Dst      string `mapstructure:"dst"`
	Template bool   `mapstructure:"template"`
}

func (cfg *config) Location() *time.Location {
	if cfg.location == nil {
		re := regexp.MustCompile(`([^\s]+) ([\+\-])(\d+):(\d+)`)
		groups := re.FindStringSubmatch(cfg.Timezone)
		if len(groups) == 0 {
			loc, err := time.LoadLocation(cfg.Timezone)
			if err != nil {
				exitApplication("invalid timezone: "+cfg.Timezone, 1)
			}
			cfg.location = loc
		} else {
			hour, _ := strconv.ParseInt(groups[3], 10, 32)
			min, _ := strconv.ParseInt(groups[4], 10, 32)
			sec := hour*60*60 + min*60
			if groups[2] == "-" {
				sec = sec * -1
			}
			cfg.location = time.FixedZone(groups[1], int(sec))
		}
	}
	return cfg.location
}

func loadConfig(L *lua.LState) *config {
	L.PreloadModule("silkylog", LuaModuleLoader)
	cfg := &config{}
	L.SetGlobal("config", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		if err := gluamapper.Map(tbl, cfg); err != nil {
			exitApplication(err.Error(), 1)
		}
		L.SetGlobal("CONFIG", tbl)
		return 0
	}))
	if err := L.DoFile("config.lua"); err != nil {
		exitApplication(fmt.Sprintf("Failed to load config.lua:\n\n%v", err.Error()), 1)
	}
	themecfg := &config{}
	L.SetGlobal("config", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		if err := gluamapper.Map(tbl, themecfg); err != nil {
			exitApplication(err.Error(), 1)
		}
		L.SetGlobal("THEME_CONFIG", tbl)
		return 0
	}))
	if err := L.DoFile(filepath.Join(cfg.ThemeDir, cfg.Theme, "theme.lua")); err != nil {
		exitApplication(fmt.Sprintf("Failed to load theme.lua:\n\n%v", err.Error()), 1)
	}
	cfg.ThemeConfig = themecfg

	return cfg
}
