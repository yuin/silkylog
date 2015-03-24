package main

import (
	"fmt"
	"github.com/yuin/gluamapper"
	"github.com/yuin/gopher-lua"
	"path/filepath"
	"time"
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
	TrimHtml    bool

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
		loc, err := time.LoadLocation(cfg.Timezone)
		if err != nil {
			exitApplication("invalid timezone: "+cfg.Timezone, 1)
		}
		cfg.location = loc
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
