## Newscanoe

Newscanoe aims to be a minimal reimplementation of the glorious [newsboat](https://newsboat.org/): 
- only for Linux (at the moment, at least) terminal emulators supporting [VT100](https://en.wikipedia.org/wiki/VT100) terminal escape sequences 
- written in Go but rigorously nonglamorous (i.e. vim-like style)
- meant to be lighter and easier to build from source or to distribute (in the future) as a traditional distribution-dependent (e.g. rpm/deb) or independent (e.g. Snap, Flatpak, or AppImage) package.

### Configuration

The only config file consists of urls of RSS/Atom feeds listed line-by-line (see the [urls sample file](./assets/urls)) and located in the directory `$XDG_CONFIG_HOME/newscanoe` or `$HOME/.config/newscanoe`.

If such file does not already exist, it will be created at the first execution of the app and you will be able to manually insert a url by typing `a`, Otherwise create such file with any text editor.

Once loaded, feeds are cached in the directory `$XDG_CACHE_HOME/newscanoe` or `$HOME/.cache/newscanoe`.

Currently, the app uses just the default foreground colour (+ red/green as feedbacks to user actions) of your terminal theme to highlight the different UI components.

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
    - `ENTER`, append it to the config file
    - `<`, `>`, move the cursor to the previous/next char
    - `BACKSPACE`, cancel last char
    - `CANC`, cancel currently highlighted char

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

[![asciicast](https://asciinema.org/a/GmD6rN1s4vcQVT0xmlYOrlacq.svg)](https://asciinema.org/a/GmD6rN1s4vcQVT0xmlYOrlacq)

### Development

To work on the source code, start the app enabling debug mode with the `-d` flag and redirect stderr to a log file:
```bash
:~$ go run cmd/newscanoe/main.go -d  2> log 
```

To debug with Visual Studio Code, set the following `console` field in the `launch.json` file as follows:
```
{
    "version": "0.2.0",
    "configurations": [
        {
            [...]
            "console": "integratedTerminal",
            [...]
        }
    ]
}
```

---

### Credits

Much of the wizardry used to control the terminal was inspired by the well-written tutorial [Build Your Own Text Editor](https://viewsourcecode.org/snaptoken/kilo/) by [Paige Ruten](https://viewsourcecode.org/), which explains in depth the source code of [kilo](https://github.com/antirez/kilo), the infamous small text editor conceived by [Salvatore Sanfilippo aka antirez](http://invece.org/).

### References

- [Text Editor Data Structures](https://cdacamar.github.io/data%20structures/algorithms/benchmarking/text%20editors/c++/editor-data-structures/?utm_source=programmingdigest&utm_medium&utm_campaign=1663)
- [Text Rendering Hates You](https://faultlore.com/blah/text-hates-you/)
