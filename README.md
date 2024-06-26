## Newscanoe

Newscanoe aims to be a minimal reimplementation of the glorious [newsboat](https://newsboat.org/): 
- for UNIX systems and terminal emulators supporting standard VT100 escape sequences (i.e. xterm-derived)
- written in Go but rigorously nonglamorous (i.e. vim-like)
- meant to be lighter and easier to build from source and to distribute.

A tool for all of you information junkies, as simple as you always secretly desired, to organize the internet and make sense of the web. 

### Configuration

A plain text file (named as `config`) is used to configure the app: it consists of a list of feed urls with a name preceded by a pound sign (`#`) (see the [example file](./assets/config) in this repo) and it is located in the directory `$XDG_CONFIG_HOME/newscanoe` (or `$HOME/.config/newscanoe`).

If such file does not already exist, it will be created at the first execution of the app and you will be prompted to manually insert a url by typing `a`. 
You can then edit such file with any text editor (`vi` is the default, unless `EDITOR` environment variable is set) by running: `newscanoe -e`. 

Once loaded, feeds are cached in the directory `$XDG_CACHE_HOME/newscanoe` (or `$HOME/.cache/newscanoe`). The cache can be cleaned up by running `newscanoe -c`

### Keybindings

Supported key bindings:
- `r`, load/reload the currently selected feed
- `R`, load/reload all the feeds
- `q`, quit the app
- `BACKSPACE`, go back to previous section
- `ENTER`, go into the currently highlighted element
- `l`, open an article with `lynx` (if installed in the system)
- `o`, open an article with the default (according to xdg-settings) browser for the user's desktop environment
- `^`, `v`, move the cursor to the previous/next row
- `Page Up`, `Page Down`, move the cursor to the previous/next page
- `a`, insert a new feed url by typing it letter-by-letter or pasting it with CTRL+SHIFT+v

### Installation

Build from source:
```bash
:~$ git clone https://github.com/giulianopz/newscanoe.git
:~$ cd newscanoe 
:~/newscanoe$ go generate ./... && go build
:~/newscanoe$ cp ./newscanoe /usr/local/bin
```

Or download the latest pre-compiled binary from [GitHub](https://github.com/giulianopz/newscanoe/releases) and then install it into your PATH.

[![asciicast](https://asciinema.org/a/ITMxRztPY65ijVedNrjHCKeWz.svg)](https://asciinema.org/a/ITMxRztPY65ijVedNrjHCKeWz)

---

### Credits

Much of the wizardry used to control the terminal was inspired by the well-written tutorial [Build Your Own Text Editor](https://viewsourcecode.org/snaptoken/kilo/) by [Paige Ruten](https://viewsourcecode.org/), which explains in depth the source code of [kilo](https://github.com/antirez/kilo), the infamous small text editor conceived by [Salvatore Sanfilippo aka antirez](http://invece.org/).
