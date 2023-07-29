silkylog = require("silkylog")

config {
  debug               = false,
  site_url            = "http://example.com/",
  editor              = {"vim"},
  numthreads          = 8,
  timezone            = "JST +09:00",
  theme               = "default",
  pagination1         = 3,
  pagination2         = 50,
  trim_html           = true,

  params = {
    author              = "Your name",
    site_name           = "Your site",
    site_description    = "Your site description",
    google_analytics_id = "",
    disqus_short_name   = "",
  },

  top_url_path        = "",
  article_url_path    = [[articles/{{ .PostedAt.Year | printf "%04d" }}/{{ .PostedAt.Month | printf "%02d" }}/{{ .PostedAt.Day | printf "%02d" }}/{{ .Slug }}.html]],
  article_title       = [[{{ .App.Config.Params.SiteName }} :: {{ .Article.Title }}]],

  index_url_path      = [[{{if (eq .Page 0)}}index.html{{else}}page/{{ .Page }}/index.html{{end}}]],
  index_title         = [[{{.App.Config.Params.SiteName}}]],

  tag_url_path        = [[articles/tag/{{ .Tag }}/{{if (ne .Page 0)}}page/{{ .Page }}/{{end}}index.html]],
  tag_title           = [[{{.App.Config.Params.SiteName}} :: tag :: {{.Tag}}]],

  annual_url_path     = [[articles/{{ .Year | printf "%04d" }}/{{if (ne .Page 0)}}page/{{ .Page }}/{{end}}index.html]],
  annual_title        = [[{{.App.Config.Params.SiteName}} :: annual archive :: {{.Year}}]],

  monthly_url_path    = [[articles/{{ .Year | printf "%04d" }}/{{ .Month | printf "%02d" }}/{{if (ne .Page 0)}}page/{{ .Page }}/{{end}}index.html]],
  monthly_title       = [[{{.App.Config.Params.SiteName}} :: monthly archive :: {{.Year}}.{{.Month}}]],

  include_url_path    = [[include/{{ .Name }}]],
  feed_url_path       = [[{{ .Name }}]],
  file_url_path       = [[{{ .Path }}]],

  content_dir         = "src",
  output_dir          = "public_html",
  theme_dir           = "themes",
  extra_files         = {
    {src = "favicon.ico", dst = "", template = false},
    {src = "404.html", dst = "", template = false},
    {src = "CNAME", dst = "", template = false},
    {src = "profile.html", dst = "", template = true}
  },
  clean = {
    "articles",
    "page",
  },

  markup_processors = {
    [".md"]  = {
      name = "goldmark",
      mdopts = {
        "autoHeadingID",
        "attribute",
      },
      htmlopts = {
        "unsafe",
      },
      exts = {
        "table",
        "strikethrough",
        "linkify",
        "taskList",
        "gfm",
        "definitionList",
        "footnote",
        "typographer",
        "cjk",
        "highlighting"
      }
    },
    [".rst"] = function(text) 
      local html = assert(silkylog.runprocessor([[python]],[[rst2html.py]], text))
      return html
    end
  }
}

