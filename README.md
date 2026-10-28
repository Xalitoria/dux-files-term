# Dux Files

<div align="center">
  <img src="assets/dev.duxlabs.files.svg" alt="Dux Files" width="160">

### A file manager for the dux desktop

Built with [Quickshell](https://quickshell.org/) and [Go](https://go.dev/)

</div>

Dux Files is a standalone file manager for Wayland. A small Go daemon does the file work and a Quickshell UI sits on top of it. It looks and feels like the rest of the dux desktop and picks up Dux Shell's colors when DMS is running, but it works fine on its own.

This repository is a fork of [duxlabs-dev/dux-files](https://github.com/duxlabs-dev/dux-files) that adds a [docked terminal](#docked-terminal). Everything else works like upstream.

## Features

**Tabs, panes and views**
Tabs, a split view with two panes, and list, grid and column views. Sort and view can be remembered per folder.

**Fast navigation**
Type to jump to a name or filter the folder, jump to any folder you have visited before, and reopen closed tabs.

**Search**
Recursive search by name with filters. Uses [danksearch](https://github.com/duxlabs-dev/dux-search) when it is running and walks the tree when it isn't.

**File operations**
Copy, move, rename, trash and delete run as jobs with progress and conflict handling, and most of them can be undone. Denied operations offer Retry as administrator through polkit.

**Archives**
Opening an archive extracts it next to itself, like Nautilus. Compress to zip, tar.gz, tar.xz and tar.zst. All pure Go, no external tools.

**Devices and network**
Mount, unmount and eject drives through udisks2. Connect to sftp, smb, ftp, WebDAV, nfs and afp shares through gvfs.

**Trash, recent files and bookmarks**
Browse and restore the trash, see recently used files, and keep bookmarks in the sidebar.

**Docked terminal**
A shell built into the window, docked under the files or beside them, that follows the folder you are in and uses the colors and transparency of your own terminal. See [Docked terminal](#docked-terminal).

**Open and save dialogs**
Can serve the desktop portal's file chooser, so other apps get the same window, sidebar and keys when they open or save a file.

**Desktop integration**
Owns `org.freedesktop.FileManager1`, so "show in folder" from browsers and other apps lands here.

## Installation

### From source

You need Go and `quickshell`.

```bash
git clone --recurse-submodules https://github.com/Xalitoria/dux-files-term.git
cd dux-files-term
make
sudo make install
```

`make install` puts the `dfiles` binary (UI included), the desktop entry, icon, metainfo, polkit policy and the D-Bus activation files under `/usr/local`. `sudo make uninstall` removes them.

To keep it running in the background so windows open instantly:

```bash
make install-systemd
systemctl --user enable --now dfiles
```

### NixOS / home-manager

```nix
{
  inputs.dux-files.url = "github:Xalitoria/dux-files-term";

  # home-manager or NixOS
  imports = [ inputs.dux-files.homeModules.dux-files ];
  programs.dux-files.enable = true;
}
```

Use `nixosModules.dux-files` for a NixOS configuration. `programs.dux-files.systemd.enable` starts it with the session, and `programs.dux-files.portal.enable` registers the open and save dialogs. `programs.dux-files.settings` is written to `~/.config/dfiles/ui-settings.json`.

[distro/nix/example/flake.nix](distro/nix/example/flake.nix) is a complete standalone home-manager flake. Try it with:

```bash
nix build ./distro/nix/example#homeConfigurations.example.activationPackage --override-input dux-files "git+file://$PWD"
```

### Gentoo

There is a live ebuild in [distro/gentoo](distro/gentoo). Copy it into a local overlay and install it:

```bash
sudo cp -r distro/gentoo/x11-misc/dux-files-term /var/db/repos/localrepo/x11-misc/
echo "x11-misc/dux-files-term **" | sudo tee /etc/portage/package.accept_keywords/dux-files-term
sudo emerge -av x11-misc/dux-files-term
```

It builds from this repository with `dev-lang/go` 1.27 or newer, vendoring the Go modules at build time, and it needs `gui-apps/quickshell[wayland]`. The `systemd` USE flag installs the user unit. It installs the same files as `x11-misc/dux-files`, so it blocks that package.

### Flatpak

See [distro/flatpak/README.md](distro/flatpak/README.md).

### Optional dependencies

Picked up from `PATH` when the daemon starts:

| package | enables |
| --- | --- |
| `bubblewrap` | runs external thumbnailers in a sandbox, like gnome-desktop does |
| `ffmpegthumbnailer` | video thumbnails |
| `poppler` (`poppler-utils` on Debian) | pdf thumbnails |
| `gvfs` | network shares and the trash on removable drives |
| `udisks2` | the devices sidebar |
| `dconf` | following the colors of GNOME Terminal in the docked terminal |

Images in png, jpeg, gif, bmp, tiff and webp are decoded in-process. Anything else uses the freedesktop thumbnailers installed on the system, so packages that ship one (glycin, gsf, evince, totem) just work. `dfiles ipc system.capabilities` shows what was found.

## Usage

```bash
dfiles show                     # Show the window, starting dfiles if needed
dfiles open ~/Downloads         # Open folders, one tab each
dfiles open --new-window ~/src  # Same, in a new window
dfiles select ~/notes.txt       # Open the folder with the file selected
dfiles properties ~/notes.txt   # Properties dialog
```

`dfiles open` accepts plain paths and these locations:

| location | opens |
| --- | --- |
| `/path`, `relative/path`, `file:///path` | that path |
| `trash:///` | the trash |
| `computer:///` | `/` |
| `network:///` | the Connect to Server dialog |
| `admin:///path` | the path, with Retry as administrator on denied operations |
| `sftp:// ssh:// smb:// ftp:// ftps:// dav:// davs:// nfs:// afp://` | mounted through gvfs, then opened in a new tab |

Started by the systemd unit, closing the window just hides it. Started any other way, closing the last window quits unless Settings, Close behavior says otherwise.

### Command line

The CLI talks to the running dfiles when there is one and does the work itself when there isn't. `--json` gives machine-readable output.

```bash
dfiles trash put|list|count|restore|empty
dfiles mime query|default|set
dfiles devices list|mount|unmount|eject
dfiles search <root> <query>
dfiles ipc list                 # Every IPC method the daemon exposes
dfiles ipc <method> key=value...
```

Jobs started from the CLI answer their own prompts. A restore that finds its original path taken keeps both (`--conflict skip|replace` changes that), and a failed item is skipped, reported, and makes the command exit 1.

## Docked terminal

Press `F4` to cycle the terminal through its places, or pick one from the terminal button in the toolbar:

| mode | where |
| --- | --- |
| hidden | no terminal, the shell ends |
| under the files | below the panes, next to the sidebar |
| along the bottom | the full width of the window, below the sidebar as well |
| on the right | the right half of the window, below the toolbar |

`F4` also works while the terminal has focus. Drag the splitter to resize it, the height is remembered. The terminal starts your `$SHELL` in the folder you are looking at. When the shell is idle at a prompt it changes folder with you, and a program that is running is never typed into. Exiting the shell hides the terminal.

| key | does |
| --- | --- |
| `Ctrl+Shift+C` | copy the selection |
| `Ctrl+Shift+V` | paste |
| `Ctrl+Shift+Plus` / `Ctrl+Shift+Minus` | zoom in and out |
| `Ctrl+Shift+Backspace` | reset the zoom |

These always work. Your terminal's own bindings for copy, paste and zoom work as well, see [Behavior](#behavior). The zoom is remembered.

### Mouse

Drag to select, double click for a word and triple click for a line. By default a word is made of letters, digits and `@-./_~?&=%+#`, so a path or a link is picked whole. Your terminal's word setting replaces that. The wheel scrolls the history. Programs that ask for the mouse, such as editors and file browsers in the terminal, get clicks, drags, motion and the wheel, and hold `Shift` to select instead. Programs that ask for pixel coordinates, such as kitty's own `kitten` tools, get the middle of the cell.

Selecting works on what is on screen. Typing, scrolling and resizing clear the selection. Wrapped lines copy with a line break at each row edge. There is no primary selection, so middle click doesn't paste.

### Colors

The terminal looks like your terminal. When it opens, dfiles reads the config of your default terminal and takes its foreground, background, cursor and 16 ANSI colors, and its transparency, so a prompt looks the same in both places. Colors the config leaves out use the Dux theme. With no readable config, the whole terminal uses the Dux theme.

The default terminal is the first entry of `~/.config/xdg-terminals.list` (a `<desktop>-xdg-terminals.list` comes first), then `$TERMINAL`. Generic names such as `x-terminal-emulator` don't count. Without a default, the first supported terminal that has a config is used. A default that isn't supported follows nothing.

| terminal | read from |
| --- | --- |
| kitty | `kitty.conf` and its `include` lines |
| Ghostty | `config`, `config-file` and `theme` (a `light:X,dark:Y` theme uses the dark one) |
| Alacritty | `alacritty.toml` and its imports |
| foot | `foot.ini` and its `include` |
| WezTerm | the literal settings of the `colors` table in `wezterm.lua`, and scheme files in its `colors/` folder |
| Konsole | the default profile's color scheme |
| Rio | `config.toml` and its theme |
| Xfce Terminal | `terminalrc` |
| GNOME Terminal | the default profile in dconf, through the `dconf` tool |
| Terminator | the `default` profile in `config` |
| xterm, urxvt | `~/.Xresources` or `~/.Xdefaults`, without transparency |

WezTerm is configured in Lua, so only literal settings are read, and built-in scheme names are not known. The config is read when the terminal opens, so close and reopen it after changing yours. Configs are looked up in the XDG config folders and in the real `~/.config`, so the Flatpak build, which has host filesystem access, finds them too.

Transparency needs a compositor that shows what is behind the window. The main window is transparent and everything but the terminal is painted opaque. A see-through terminal has square corners. Blur is up to the compositor.

### Behavior

Selection, keys and font follow the same config as the colors, for the terminals in this table. Settings a config leaves out keep the defaults above.

| setting | kitty | Ghostty | Alacritty | foot | WezTerm |
| --- | --- | --- | --- | --- | --- |
| selection colors | `selection_foreground`, `selection_background` | `selection-foreground`, `selection-background` | `colors.selection` | `selection-foreground`, `selection-background` | `colors.selection_fg`, `selection_bg` |
| word characters | `select_by_word_characters` | `selection-word-chars` | `selection.semantic_escape_chars` | `word-delimiters` | `selection_word_boundary` |
| copy on select | `copy_on_select clipboard` | `copy-on-select = clipboard` | `selection.save_to_clipboard` | `selection-target` is `clipboard` or `both` | no |
| copy, paste, zoom keys | `map` | `keybind` | `keyboard.bindings` | `[key-bindings]` | `keys` entries written on one table each |
| font | `font_family`, `font_size` | `font-family`, `font-size` | `font.normal.family`, `font.size` | `font` | `font`, `font_size` |
| cursor shape and blink | `cursor_shape`, `cursor_blink_interval` | `cursor-style`, `cursor-style-blink` | `cursor.style` | `[cursor]` `style`, `blink` | `default_cursor_style` |

- A bound key is added to the defaults, it doesn't remove them. Only bindings for copy, paste and zoom are used. Key sequences, bindings that need a mode, kitty's conditional `map --when-focus-on` and everything else are ignored.
- Terminals other than kitty and xterm zoom with `Ctrl+Plus`, `Ctrl+Minus` and `Ctrl+0` by default, so once their config is found those work in the dock too.
- A copy key that isn't `Ctrl+Shift+C`, such as `Ctrl+C`, only copies while something is selected, and otherwise goes to the shell.
- Copy on select copies to the clipboard. There is no primary selection, so `true` in Ghostty and `primary` in kitty copy nothing.
- The font is used when it is installed and monospaced, then the Dux theme font. Sizes are points. A Nerd Font in your terminal config makes prompt icons show up here too.
- The cursor is steady unless the config asks for blinking. A program that sets its own cursor (a vi mode in the shell, an editor) wins over the config.
- Other terminals only give their colors.

### What programs can do

The terminal is `xterm-256color` with true color, and understands these on top of the usual:

- **Images**: the kitty graphics protocol, so `kitten icat`, `kitten choose-fonts`, and file managers and system info tools that draw images work. Images can be sent directly, from a file, a temporary file or shared memory, as RGB, RGBA or PNG, compressed or not, cropped, scaled to a number of cells, layered above or below the text, and deleted again. They scroll with the text and go away when their cells are cleared. A terminal keeps up to 64 MiB of images, the oldest are dropped first. Not supported: animation frames and Unicode placeholders.
- **Hyperlinks**: `OSC 8` links and plain `http(s)://`, `ftp://` and `file://` text open with `Ctrl+click`, in the browser or the default app. The pointer becomes a hand over a link while `Ctrl` is held. Programs that track the mouse get the click unless `Shift` is held.
- **Underlines**: single, double, curly, dotted and dashed, with their own colors.
- **Cursor**: block, bar and underline, steady or blinking, set by the config or by the program.
- **Clipboard**: a program can put text on the clipboard with `OSC 52`, which is how editors and `tmux` copy over SSH. Reading the clipboard from a program isn't allowed.
- **Focus**: programs that ask for focus events are told when the terminal gains or loses focus.
- **Synchronized output**: a program can hold the drawing back while it paints a whole screen (mode 2026), up to 150 ms.
- **Sizes**: the cell and window size in pixels are reported to `ioctl` and `CSI 14 t`, `16 t` and `18 t`, which image tools use to scale things.

Not supported yet: the kitty keyboard protocol (programs get the usual xterm keys), Sixel, shell integration marks and desktop notifications.

Inside a Flatpak the terminal needs host access (`org.freedesktop.Flatpak`) and doesn't follow folders. Without that access, opening it shows an error.

## Default file manager

To open folders here from other apps:

```bash
xdg-mime default dev.duxlabs.files.desktop inode/directory
```

The installer doesn't do this for you.

While it runs, dfiles owns `org.freedesktop.FileManager1`, so `ShowFolders`, `ShowItems` and `ShowItemProperties` calls come here. Chromium based browsers go through the portal's `OpenDirectory`, which forwards to `ShowItems`. dfiles never takes the name from another file manager, it waits in line until that one quits. Turn it off with Settings, General, "Register as the session file manager".

`make install` also ships an activation file for the name, so a call can start dfiles when nothing owns it. Nautilus, Dolphin, Thunar and Nemo ship one too. dbus-daemon prefers the first service directory, and the default `PREFIX=/usr/local` comes before `/usr/share`.

## Open and save dialogs

Nothing changes until you turn it on. Use Settings, General, "Use Dux Files for open and save dialogs", or:

```bash
dfiles portal install
dfiles portal status
dfiles portal uninstall   # put things back
```

This puts dfiles first for the FileChooser in your `~/.config/xdg-desktop-portal/<desktop>-portals.conf`, keeps the old backend as a fallback, and remembers what it changed. xdg-desktop-portal only reads its config at startup, so it has to restart. Both the settings page and the CLI ask first, since a restart cancels dialogs other apps have open.

Apps that use the portal pick it up: Flatpak apps, Chromium browsers, Firefox with `widget.use-xdg-desktop-portal.file-picker` set to 1, GTK 4 apps with `GDK_DEBUG=portals` and Qt apps with `QT_QPA_PLATFORMTHEME=xdgdesktopportal`. Apps that draw their own dialogs are unchanged.

The dialog is its own window. Qt 6.12 and Quickshell can't attach it to the calling app's window, so `modal` is ignored. Without the systemd unit the first dialog is a cold start. A niri rule to float it:

```kdl
window-rule {
    match app-id=r#"^dev\.duxlabs\.files$"# title=r#"^(Open|Save|Select|Upload|Export|Import)"#
    open-floating true
}
```

Packagers: install `assets/portal/dfiles.portal` under `share/xdg-desktop-portal/portals/` and the service file under `share/dbus-1/services/` with the binary path substituted. Don't ship a `portals.conf` that prefers it. The Flatpak build can't register the backend because Flatpak exports no portal files.

## Repository Structure

```
dux-files-term/
├── core/               # Go daemon and the dfiles CLI
│   ├── cmd/dfiles/     # CLI commands and the daemon entrypoint
│   └── internal/       # File ops, IPC, devices, search, portal, D-Bus, terminal and its colors
├── quickshell/         # The UI
│   ├── Common/         # Theme, settings, i18n
│   ├── Services/       # Bridges to the daemon
│   ├── Modules/        # Windows, panes, dialogs, settings
│   └── translations/   # POEditor catalogs
├── dux-qml-common/    # Shared widget library (submodule, linked as quickshell/DCommon)
├── distro/             # Nix, Gentoo and Flatpak packaging
└── assets/             # Icon, desktop entry, D-Bus, portal and systemd files
```

## Development

```bash
make run                                                  # Build and run against the repo's QML
DFILES_ENABLE_HOTRELOAD=1 core/bin/dfiles run -c quickshell  # Hot reload the UI
make test                                                 # Go tests
make test-qml                                             # QML tests
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, the shared widget library, translations and our rules on AI generated code.

## License

MIT License - See [LICENSE](LICENSE) for details.
