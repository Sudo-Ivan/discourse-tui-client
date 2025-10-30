# Discourse TUI Client

A simple and fast discourse tui client that doesn't require the user API.

## Features

- **Authentication**: Cookie-based authentication with optional AES-GCM encryption for the cookie file.
- **Multi-instance support**: Connect to different Discourse forums
- **Offline access**: Local caching of topics and posts for offline reading
- **Terminal UI**: Interactive TUI for browsing topics and reading posts
- **Search functionality**: Full-text search across posts and topics
- **Topic creation**: Create new topics directly from the TUI
- **Export options**: Save topics to text, JSON, or HTML files
- **Unauthenticated mode**: Browse public forums without login
- **Customizable colors**: Theme customization via configuration file
- **Keyboard navigation**: Efficient keyboard shortcuts for all operations
- **Create Posts**: Create new posts directly from the TUI

## Supported Platforms

- Linux
- FreeBSD
- OpenBSD
- MacOS
- Windows
- ARM64 and v6,v7 (Raspberry Pi, etc.)

## To Do

- [ ] Multiple instances support
- [ ] Reply
- [ ] Edit and Delete
- [ ] Fix URLs (not showing)
- [ ] Improve keyboard navigation
- [ ] Navigate categories and tags
- [ ] DM support
- [ ] Fix codeblocks and ASCII art rendering 

## Installation

### Script

Run the installation script directly from the repository:

```bash
# Using curl
curl -fsSL https://git.quad4.io/Ivan/discourse-tui-client/raw/branch/main/install.sh | sh

# Using wget
wget -qO- https://git.quad4.io/Ivan/discourse-tui-client/raw/branch/main/install.sh | sh

# Or download and run manually
curl -fsSL https://git.quad4.io/Ivan/discourse-tui-client/raw/branch/main/install.sh -o install.sh
chmod +x install.sh
./install.sh
```

### Binary Releases

1. Download the binary from the [releases](https://github.com/Sudo-Ivan/discourse-tui-client/releases) page.
2. Make the binary executable:
```bash
chmod +x discourse-tui-client
```
3. Move the binary to a directory in your PATH:
```bash
sudo mv discourse-tui-client /usr/local/bin/
```
4. Run the client:
```bash
discourse-tui-client
```

### From source

```bash
go install git.quad4.io/Ivan/discourse-tui-client@latest
```

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
  -e    Encrypt cookies file with a password (shorthand).
  -encrypt-cookies
        Encrypt cookies file with a password.
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

1. **Cookie-based Authentication**: Instead of using API tokens, it manages authentication through browser-style cookies stored as `cookies.txt` in users `$HOME/.config/discourse-tui-client/cookies.txt`. Optionally, cookies can be encrypted using AES-GCM encryption with a user-provided password. 

2. **Direct HTTP Requests**: Communicates with Discourse instances through standard HTTP requests to endpoints like `/latest.json` and `/t/{id}.json`.

3. **Local Cache**: Stores responses in the user's cache directory to improve performance and enable offline access to previously viewed content.

4. **Terminal UI**: Features a responsive terminal interface built with Bubble Tea that displays:
   - A navigable list of latest topics
   - Content viewer for reading posts
   - Full-text search across all posts and topics
   - Fullscreen mode

5. **Customizable Colors**: Allows theme customization through a simple configuration file `colors.txt` in users `$HOME/.config/discourse-tui-client/colors.txt`.

Go external dependencies:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [lipgloss](https://github.com/charmbracelet/lipgloss)
- [Bubbles](https://github.com/charmbracelet/bubbles)
- [bluemonday](https://github.com/microcosm-cc/bluemonday)
- [gjson](https://github.com/tidwall/gjson)

## Cookie Encryption

For enhanced security, you can encrypt your cookies file using AES-GCM encryption with a password. Use the `--encrypt-cookies` or `-e` flag when running the application.

When encryption is enabled:
- You'll be prompted to enter a password when logging in and when the application starts
- The cookies file will be stored in encrypted form on disk
- The same password must be used to decrypt the cookies later

Example:
```bash
discourse-tui-client --encrypt-cookies
```

**Note**: If you forget your encryption password, you'll need to log in again to recreate the cookies file.

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