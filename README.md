## Newscanoe

Newscanoe aims to be a minimal reimplementation of the glorious [newsboat](https://newsboat.org/): 
- only for Linux terminal emulators (at the moment, at least)
- written in Go but rigorously nonglamorous
- meant to be lighter and easier to build from source or to distribute (in the future) as a traditional distribution-dependent (e.g. rpm/deb) or independent (e.g. Snap, Flatpak, or AppImage) package.

### Configuration

The only prerequisite is a file containing only urls of RSS/Atom feeds line-by-line (see the [urls sample file](./assets/urls)) located in the directory `$XDG_CONFIG_HOME/newscanoe` or `$HOME/.config/newscanoe`.

Once loaded, feeds are cached in the directory `$XDG_CACHE_HOME/newscanoe` or `$HOME/.cache/newscanoe`.

### Keybindings

Supported key bindings:
- `r`, load/reload the currently selected feed
- `R`, load/reload all the feeds
- `q`, quit the app
- `BACKSPACE`, go back to previous section
- `ENTER`, go to the list of articles of a feed or display a single article in the list (poor quality at the moment, read below for some alternatives)
- `l`, opens an article with `lynx` if installed in the system 
- `o`, open an article with the default browser for the user's desktop environment
- `^`, `v`, `<`, `>`, move the cursor across the text

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

[![asciicast](https://asciinema.org/a/YyzLpxYUswPqeaeoRYVnAP9y1.svg)](https://asciinema.org/a/YyzLpxYUswPqeaeoRYVnAP9y1)

### Development

To work on the source code, start the app enabling debug mode with the `-d` flag and redirect stderr to a file:
```bash
:~/newscanoe$ go run cmd/newscanoe/main.go -d  2> log 
```

---

### Credits

Much of the wizardry used to control the terminal was inspired by the well-written tutorial [Build Your Own Text Editor](https://viewsourcecode.org/snaptoken/kilo/) by [Paige Ruten](https://viewsourcecode.org/), which explains in depth the source code of [kilo](https://github.com/antirez/kilo), the infamous small text editor conceived by [Salvatore Sanfilippo aka antirez](http://invece.org/).
