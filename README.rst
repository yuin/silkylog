===============================================================================
silkylog: an extensible static site generator
===============================================================================

silkylog is a simple and extensible static site generator written in Go and `GopherLua <https://github.com/yuin/gopher-lua>`_ .

----------------------------------------------------------------
Usage
----------------------------------------------------------------
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Create an environment
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: bash
    
    git clone https://github.com/yuin/silkylog.git myblog
    cd myblog
    go get -u github.com/russross/blackfriday
    go get -u github.com/codegangsta/cli
    go get -u github.com/mitchellh/mapstructure
    go build
    mkdir public_html
    silkylog new
    silkylog build
    silkylog serve
    # open your browser with an URL http://localhost:7000

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Directory structure
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

::

    .
    +-- src
    |   +-- articles
    |       +-- 2015
    |           +-- 01
    |               +-- 24_article-slug.md
    |               +-- 25_article-slug.rst
    |   +-- extras
    |       +-- favicon.ico
    |       +-- 404.html
    +-- public_html
    +-- themes
    |   +-- default
    |       +-- extras
    |       +-- feeds
    |       +-- include
    |       +-- layouts
    |       +-- pages
    |       +-- theme.lua
    +-- config.lua

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Article format
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
An article consists of two parts: a HEADER, followed by the BODY. 
The header contains information about the article.

The body markup type is decided based on the extension of the filename.

- .md : markdown
- .rst : reStructuredText

::

    :title: Article title
    :tags: golang,lua,gopherlua
    :status: published
    :posted_at: 2015-02-15 22:43:19
    :updated_at: 2015-02-15 22:43:19
    
    Article Body


~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Commands
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

::

   build        build my site
   clean        clean all data
   serve        serve contents
   preview      preview contents
   help, h      Shows a list of commands or help for one command

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Configuration
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
TODO

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Lua API
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

:silkylog.runprocessor(cmd string, [cmdopts string, cmdopts string...], text string) -> string:
    run an external markup processor. A markup processor reads the text from stdin, converts it into a html and prints it to stdout.

:silkylog.htmlescape(text string) -> string:
    replace special characters with the correct HTML entities.

:silkylog.htmlunescape(text string) -> string:
    turn HTML character references into their plain text.
    
:silkylog.urlescape(text string) -> string:
    replace special characters with single byte characters.

:silkylog.formatmarkup(text string, format string) -> string:
    convert the ``text`` written in ``format`` into HTMLs.

:silkylog.title(data table) -> string:
    format the title defined in the ``config.lua`` with the ``data``.

:silkylog.path(data table) -> string:
    format the path defined in the ``config.lua`` with the ``data``.

:silkylog.url(data table) -> string:
    format the url defined in the ``config.lua`` with the ``data``.

:silkylog.fullurl(data table) -> string:
    format the url with domains defined in the ``config.lua`` with the ``data``.

:silkylog.copyfile(src, dst string) -> true or (nil, message string): 
    copy the file ``src`` to the ``dst``. return true if no errors were occurred, nil and an error message otherwise.

:silkylog.copytree(src, dst string) -> true or (nil, message string): 
    copy the directory ``src`` to the ``dst``. return true if no errors were occurred, nil and an error message otherwise.

:silkylog.isdir(path string) -> bool:
    return true if the ``path`` is a directory, false otherwise.

:silkylog.isfile(path string) -> bool:
    return true if the ``path`` is a regular file, false otherwise.

:silkylog.pathexists(path string) -> bool:
    return true if the ``path`` refers to an existing path, false otherwise.

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Your own markup processors
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
TODO

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Create a new theme
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
TODO

----------------------------------------------------------------
Real world examples
----------------------------------------------------------------
- `inforno <http://inforno.net>`_ : My website.

----------------------------------------------------------------
License
----------------------------------------------------------------
MIT

----------------------------------------------------------------
Todo
----------------------------------------------------------------

- [ ] Writing tests
- [ ] Writing documents
- [ ] A nice default site template
- [ ] More Lua APIs

----------------------------------------------------------------
Author
----------------------------------------------------------------
Yusuke Inuzuka
