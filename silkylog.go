package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
	lua "github.com/yuin/gopher-lua"
)

func createRootLState(app *application) *lua.LState {
	L := lua.NewState()
	app.Config = loadConfig(L)
	luaPool.Put(L)
	return L
}

func main() {
	cliapp := cli.NewApp()
	app := newApp()
	_app = app
	defer luaPool.Shutdown()

	cliapp.Name = "silkylog"
	cliapp.Usage = "simple static site generator"
	cliapp.Author = "Yusuke Inuzuka"
	cliapp.Email = ""
	cliapp.Version = "0.1"
	cliapp.Commands = []cli.Command{
		{
			Name:  "site",
			Usage: "create a new site",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path",
					Usage: "path to create a site(default: current directory)",
				},
			},
			Action: func(c *cli.Context) error {
				createRootLState(app)
				err := newsite(app, c.String("path"))
				if err != nil {
					return cli.NewExitError(err.Error(), 1)
				}
				return nil
			},
		},
		{
			Name:  "build",
			Usage: "build my site",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "clean",
					Usage: "clean all data before building my site",
				},
			},
			Action: func(c *cli.Context) error {
				createRootLState(app)
				var err error
				if c.Bool("clean") {
					err = clean(app)
					if err != nil {
						return cli.NewExitError(err.Error(), 1)
					}
				}
				err = build(app)
				if err != nil {
					return cli.NewExitError(err.Error(), 1)
				}
				return nil
			},
		},
		{
			Name:  "clean",
			Usage: "clean all data",
			Action: func(c *cli.Context) error {
				createRootLState(app)
				err := clean(app)
				if err != nil {
					return cli.NewExitError(err.Error(), 1)
				}
				return nil
			},
		},
		{
			Name:  "serve",
			Usage: "serve contents",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "port",
					Usage: "server port(default 7000)",
				},
			},
			Action: func(c *cli.Context) error {
				createRootLState(app)
				port := c.Int("port")
				if port == 0 {
					port = 7000
				}
				err := serve(app, port)
				if err != nil {
					return cli.NewExitError(err.Error(), 1)
				}
				return nil
			},
		},
		{
			Name:  "preview",
			Usage: "preview contents",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path",
					Usage: "source file path(required)",
				},
				cli.IntFlag{
					Name:  "port",
					Usage: "server port(default 7000)",
				},
			},
			Action: func(c *cli.Context) error {
				createRootLState(app)
				var err error
				port := c.Int("port")
				if port == 0 {
					port = 7000
				}
				err = preview(app, port, c.String("path"))
				if err != nil {
					return cli.NewExitError(err.Error(), 1)
				}
				return nil
			},
		},
		{
			Name:  "new",
			Usage: "create new article",
			Action: func(c *cli.Context) error {
				createRootLState(app)
				err := newarticle(app)
				if err != nil {
					return cli.NewExitError(err.Error(), 1)
				}
				return nil
			},
		},
	}
	err := cliapp.Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
