package main

import (
	"fmt"
	"github.com/yuin/gopher-lua"
	"path/filepath"
	"time"
)

type config struct {
	Debug       bool     `mapstructure:"debug"`
	SiteUrl     string   `mapstructure:"site_url"`
	Editor      []string `mapstructure:"editor"`
	NumThreads  int      `mapstructure:"numthreads"`
	Timezone    string   `mapstructure:"timezone"`
	Theme       string   `mapstructure:"theme"`
	Pagination1 int      `mapstructure:"pagination1"`
	Pagination2 int      `mapstructure:"pagination2"`
	TrimHtml    bool     `mapstructure:"trim_html"`

	Params map[string]interface{} `mapstructure:"params"`

	TopUrlPath string `mapstructure:"top_url_path"`

	ArticleUrlPath string `mapstructure:"article_url_path"`
	ArticleTitle   string `mapstructure:"article_title"`

	IndexUrlPath string `mapstructure:"index_url_path"`
	IndexTitle   string `mapstructure:"index_title"`

	TagUrlPath string `mapstructure:"tag_url_path"`
	TagTitle   string `mapstructure:"tag_title"`

	AnnualUrlPath string `mapstructure:"annual_url_path"`
	AnnualTitle   string `mapstructure:"annual_title"`

	MonthlyUrlPath string `mapstructure:"monthly_url_path"`
	MonthlyTitle   string `mapstructure:"monthly_title"`

	IncludeUrlPath string `mapstructure:"include_url_path"`
	FeedUrlPath    string `mapstructure:"feed_url_path"`
	FileUrlPath    string `mapstructure:"file_url_path"`

	ContentDir string      `mapstructure:"content_dir"`
	ThemeDir   string      `mapstructure:"theme_dir"`
	OutputDir  string      `mapstructure:"output_dir"`
	ExtraFiles []extraFile `mapstructure:"extra_files"`
	Clean      []string    `mapstructure:"clean"`

	MarkupProcessors map[string]interface{} `mapstructure:"markup_processors"`

	ThemeConfig *config

	location *time.Location
}

type extraFile struct {
	Src      string `mapstructure:"src"`
	Dst      string `mapstructure:"dst"`
	Template bool   `mapstructure:"template"`
}

func (cfg *config) convertParamsCase() {
	for key, value := range cfg.Params {
		cfg.Params[toCapCase(key)] = value
	}
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
		if err := luaToGoStruct(tbl, cfg); err != nil {
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
		if err := luaToGoStruct(tbl, themecfg); err != nil {
			exitApplication(err.Error(), 1)
		}
		L.SetGlobal("THEME_CONFIG", tbl)
		return 0
	}))
	if err := L.DoFile(filepath.Join(cfg.ThemeDir, cfg.Theme, "theme.lua")); err != nil {
		exitApplication(fmt.Sprintf("Failed to load theme.lua:\n\n%v", err.Error()), 1)
	}
	cfg.ThemeConfig = themecfg
	cfg.convertParamsCase()

	return cfg
}
