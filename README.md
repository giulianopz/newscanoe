## Newscanoe

Newscanoe aims to be a minimal reimplementation of the glorious [newsboat](https://newsboat.org/): 
- only for Linux terminal emulators (at the moment, at least)
- written in Go but rigorously nonglamorous
- meant to be lighter and easier to build from source or to distribute (in the future) as a traditional distribution-dependent (e.g. rpm/deb) or independent (e.g. Snap, Flatpak, or AppImage) package.

### Configuration

The only config file consists of urls of RSS/Atom feeds listed line-by-line (see the [urls sample file](./assets/urls)) and located in the directory `$XDG_CONFIG_HOME/newscanoe` or `$HOME/.config/newscanoe`.

If such file does not already exist, it will be created at the first execution of the app and the user will be able to manually insert a url by typing `a` or by creating/modifying such file with any text editor.

Once loaded, feeds are cached in the directory `$XDG_CACHE_HOME/newscanoe` or `$HOME/.cache/newscanoe`.

### Keybindings

Supported key bindings:
- `r`, load/reload the currently selected feed
- `R`, load/reload all the feeds
- `q`, quit the app
- `BACKSPACE`, go back to previous section
- `ENTER`, go into the currently highlighted element
- `l`, open an article with `lynx` if installed in the system 
- `o`, open an article with the default browser for the user's desktop environment
- `^`, `v`, move the cursor to the previous/next row
- `Page Up`, `Page Down`, move the cursor to the previous/next page
- `a`, insert manually a new feed url, then:
    - `<`, `>`, move the cursor to the previous/next char
    - `BACKSPACE`, cancel last char
    - `CANC`, cancel currently highlighted char
    - `ENTER`, append the typed in url in the config file

### Installation

Build from source:
```bash
:~$ git clone https://github.com/giulianopz/newscanoe.git
:~$ go build -o newscanoe cmd/newscanoe/main.go && cp ./newscanoe /usr/local/bin
# or with task
:~$ task build && task install
# then, start the app
:~$ newscanoe
```

Or download the latest pre-compiled binary from [GitHub](https://github.com/giulianopz/newscanoe/releases) and then install it in your PATH.

[![asciicast](https://asciinema.org/a/6KaRRd6u85jQPQP664Zr7WvEA.svg)](https://asciinema.org/a/6KaRRd6u85jQPQP664Zr7WvEA)

### Development

To work on the source code, start the app enabling debug mode with the `-d` flag and redirect stderr to a file:
```bash
:~$ go run cmd/newscanoe/main.go -d  2> log 
```

---

### Credits

Much of the wizardry used to control the terminal was inspired by the well-written tutorial [Build Your Own Text Editor](https://viewsourcecode.org/snaptoken/kilo/) by [Paige Ruten](https://viewsourcecode.org/), which explains in depth the source code of [kilo](https://github.com/antirez/kilo), the infamous small text editor conceived by [Salvatore Sanfilippo aka antirez](http://invece.org/).
