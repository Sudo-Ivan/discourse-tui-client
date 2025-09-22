# Discourse TUI Client

A simple and fast discourse tui client that doesn't require the user API.

## Install

1. Download binaries from [releases](https://github.com/Sudo-Ivan/discourse-tui-client/releases).

## Supported Platforms

- Linux
- FreeBSD
- MacOS
- Windows
- Arm

## To Do

- [ ] Multiple instances support
- [ ] Reply and Post
- [ ] Edit and Delete
- [ ] Fix URLs (not showing)
- [ ] Improve keyboard navigation
- [ ] Navigate categories and tags
- [ ] DM support
- [ ] Fix codeblocks and ASCII art rendering
- [ ] Cookies.txt file encryption using a pin or password. 

## Usage

### Running the client

```bash
discourse-tui-client
```

Which will then prompt you with login screen if no cookies.txt found.

Arguments:

```
  -c string
        Path to cookies file (shorthand).
  -cookies string
        Path to cookies file (optional).
  -d    Enable debug logging (shorthand).
  -debug
        Enable debug logging.
  -l    Logout and delete cookies (shorthand).
  -logout
        Logout and delete cookies.
  -o string
        Output posts to file (shorthand)
  -output string
        Output posts to file (txt, json, or html)
  -r    Reset cache and force fresh fetch (shorthand).
  -reset-cache
        Reset cache and force fresh fetch.
  -u string
        Discourse instance URL (shorthand).
  -url string
        Discourse instance URL (e.g. https://forum.example.com).
```

### Extracting topics to a file

```bash
discourse-tui-client --output topics.html # or .txt, .json
```

## How it works

This client interacts with Discourse forums by:

1. **Cookie-based Authentication**: Instead of using API tokens, it manages authentication through browser-style cookies stored as `cookies.txt` in users `$HOME/.config/discourse-tui-client/cookies.txt`. 

2. **Direct HTTP Requests**: Communicates with Discourse instances through standard HTTP requests to endpoints like `/latest.json` and `/t/{id}.json`.

3. **Local Cache**: Stores responses in the user's cache directory to improve performance and enable offline access to previously viewed content.

4. **Terminal UI**: Features a responsive terminal interface built with Bubble Tea that displays:
   - A navigable list of latest topics
   - Content viewer for reading posts
   - Search capabilities
   - Fullscreen mode

5. **Customizable Colors**: Allows theme customization through a simple configuration file `colors.txt` in users `$HOME/.config/discourse-tui-client/colors.txt`.

Go external dependencies:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [lipgloss](https://github.com/charmbracelet/lipgloss)
- [Bubbles](https://github.com/charmbracelet/bubbles)
- [bluemonday](https://github.com/microcosm-cc/bluemonday)
- [gjson](https://github.com/tidwall/gjson)

## Customizing Colors

The colors are customizable through a simple configuration file `colors.txt` in $HOME/.config/discourse-tui-client/colors.txt.

The file contains a list of colors in the format `key=value`.

```
title=#FAFAFA
item=#FFFFFF
selected=170
status=#626262
error=#FF0000 
```

## License

MIT License