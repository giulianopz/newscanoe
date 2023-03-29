## Newscanoe

Newscanoe is a minimal reimplementation of the glorious [newsboat](https://newsboat.org/): 
- only for Linux terminal emulators (at the moment, at least)
- written in Go but rigorously nonglamorous
- meant to be lighter and easier to build from source or to distribute (in the future) as a traditional distribution-dependent (e.g. rpm/deb) or independent (e.g. Snap, Flatpak, or AppImage) package.

## Configuration

The only prerequisite is a file containing only urls of RSS/Atom feeds line-by-line (see the [urls sample file](./assets/urls)) located in the directory `$XDG_CONFIG_HOME/newscanoe` or `$HOME/.config/newscanoe`.

Once loaded, feeds are cached in the directory `$XDG_CACHE_HOME/newscanoe` or `$HOME/.cache/newscanoe`.

## Usage

Supported keybindings:
- `r`, load/reload the currently selected feed
- `R`, load/reload all the feeds
- `q`, quit the app
- `BACKSPACE`, go back to previous section
- `ENTER`, open a feed or an article
- `o`, open an article with the default browser for the user's desktop environment
- `^`, `v`, `<`, `>`, move the cursor across the text

### Install

> Warning: the project is still a WIP 🚧

Build from source:
```
git clone https://github.com/giulianopz/newscanoe.git
go build -o newscanoe cmd/newscanoe/main.go && cp ./newscanoe /usr/local/bin
# or with task
task build && task install
newscanoe
```

Or download the latest pre-compiled binary from [GitHub](https://github.com/giulianopz/newscanoe/releases) and then install it in your PATH.

[![asciicast](https://asciinema.org/a/HHpxc4qJBpuYpPQ7y4wkp4LK5.svg)](https://asciinema.org/a/HHpxc4qJBpuYpPQ7y4wkp4LK5)

---

### Credits

Much of the wizardry used to control the terminal was made possible only thanks to the  well-written tutorial [Build Your Own Text Editor](https://viewsourcecode.org/snaptoken/kilo/) by Paige Ruten, which explains in depth the source code of [kilo](https://github.com/antirez/kilo), the infamous small text editor crafted by antirez.
