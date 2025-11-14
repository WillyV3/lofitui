# lofitui

Terminal UI for playing lofi YouTube streams. Built with Go.

I liked [bprendie/lofigirl](https://github.com/bprendie/lofigirl) and wanted to build it better. Go is nice here.

Built by [Willy](https://willyv3.com) | [buildbywilly.com](https://buildbywilly.com)

## Dependencies

Sorry, you need these:

- `mpv` - plays the video
- `yt-dlp` - extracts stream URLs

```bash
# Debian/Ubuntu
sudo apt install mpv
pip install yt-dlp

# Arch
sudo pacman -S mpv yt-dlp

# macOS
brew install mpv yt-dlp
```

## Installation

### Homebrew
```bash
brew install willyv3/tap/lofitui
```

### Go Install
```bash
go install github.com/willyv3/lofitui@latest
```

### From Source
```bash
git clone https://github.com/willyv3/lofitui.git
cd lofitui
go build
./lofitui
```

## Usage

Run `lofitui` and use arrow keys to navigate.

- `Enter` - play stream
- `m` - manage presets
- `c` - custom URL
- `q` - quit

Config stored in `~/.config/lofitui/config.json`

## Default Streams

- [Lofi Girl - Study](https://www.youtube.com/watch?v=jfKfPfyJRdk)
- [Lofi Girl - Sleep](https://www.youtube.com/watch?v=DWcJFNfaw9c)
- [Lofi Girl - Jazz](https://www.youtube.com/watch?v=HuFYqnbVbzY)
- [Synthwave Radio](https://www.youtube.com/watch?v=4xDzrJKXOOY)
- [Chillhop Music](https://www.youtube.com/watch?v=5yx6BWlEVcY)
- [The Bootleg Boy](https://www.youtube.com/watch?v=FWjZ0x2M8og)
- [Dreamhop Music](https://www.youtube.com/live/D5bqo8lcny4)
- [Lofi Geek](https://www.youtube.com/watch?v=1tJ8sc8I4z0)
- [STEEZYASFUCK](https://www.youtube.com/watch?v=S_MOd40zlYU)
- [Homework Radio](https://www.youtube.com/watch?v=lTRiuFIWV54)

## License

MIT
