## Newscanoe

Newscanoe is a minimalistic and no-frills reimplementation of the glorious [newsboat](https://newsboat.org/): 
- only for Linux terminal emulators
- written in Go but rigorously nonglamorous
- meant to be lighter and easier to build from source or to distribute (in the future) in a distribution-independent package format such as Snap, Flatpak, or AppImage.


The only prerequisite is a file containing only urls of RSS/Atom feeds line-by-line (see the [urls sample file](./assets/urls)) located in the directory `$XDG_CONFIG_HOME/newscanoe` or `$HOME/.config/newscanoe`.

Once loaded, feeds are cached in the directory `$XDG_CACHE_HOME/newscanoe` or `$HOME/.cache/newscanoe`.

> Warn: the project is still a WIP 🚧

[![asciicast](https://asciinema.org/a/238FVtsUqBAgusEY76RYEiWAQ.svg)](https://asciinema.org/a/238FVtsUqBAgusEY76RYEiWAQ)

---

### Credits

Much of the wizardry used to control the terminal was made possible only thanks to the  well-written tutorial [Build Your Own Text Editor](https://viewsourcecode.org/snaptoken/kilo/) by Paige Ruten, which explains in depth the source code of [kilo](https://github.com/antirez/kilo), the infamous small text editor crafted by antirez.
