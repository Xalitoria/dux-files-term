# dfiles
File Manager

## Desktop integration

`dfiles open` takes paths and URIs, one tab each in the last used window. A file opens its folder with the file selected. `dfiles open --new-window` opens them in a new window, and `dfiles open` with no argument always opens a new window, so launching the app again gives you another window.

An archive given to `dfiles open`, or double clicked in a window, is extracted into a folder beside it, the way Nautilus does it. The setting "Extract archives on open" turns that off. Supported for extraction: zip, tar and tar compressed with gzip, bzip2, xz, zstd or lzma, 7z, rar and cpio, plus lone gz, bz2, xz, zst and lzma files. Compression writes zip, tar.gz, tar.xz and tar.zst. All of it is pure Go, no external tools.

| location | opens |
| --- | --- |
| `/path`, `relative/path`, `file:///path` | that path |
| `trash:///` | the home trash's `files` folder |
| `computer:///` | `/` |
| `network:///` | the Connect to Server dialog |
| `admin:///path` | the path; denied operations offer Retry as administrator |
| `sftp:// ssh:// smb:// ftp:// ftps:// dav:// davs:// nfs:// afp://` | mounted through gvfs, then opened in a new tab at the path inside the share |

`dfiles select <path...>` opens the folders holding the paths with them selected, `dfiles properties <path...>` opens the properties dialog.

While it runs, dfiles owns `org.freedesktop.FileManager1` on the session bus, so `ShowFolders`, `ShowItems` and `ShowItemProperties` calls land here. Browsers built on Chromium ask the desktop portal's `OpenDirectory` when a portal runs, and the portal forwards that to `ShowItems`. dfiles never takes the name from another file manager; it waits in the queue until that one quits. Settings, General, "Register as the session file manager" stops a running dfiles from asking for the name.

`make install` also installs `dev.duxlabs.files.FileManager1.service`, so a call can start dfiles when nothing owns the name. An instance started that way serves the call whatever the setting says, and quits if it never opens a window (for example when the call named a missing file). `make uninstall-dbus` removes the file. Nautilus, Dolphin, Thunar and Nemo ship activation files for the same name. dbus-daemon(1) gives the first service directory precedence (the default `PREFIX=/usr/local` puts ours ahead of `/usr/share`) and calls the choice arbitrary inside one directory.

The systemd user unit starts dfiles with `--keep-running`, so the close button hides the window there. Started any other way, closing the window quits unless Close behavior says to keep running.

### Open and save dialogs

dfiles can serve `org.freedesktop.impl.portal.FileChooser`, the backend side of the desktop portal's file chooser. Apps that ask the portal for files (Flatpak apps, Chromium browsers, Firefox with `widget.use-xdg-desktop-portal.file-picker` set to 1, GTK 4 apps under `GDK_DEBUG=portals`, Qt apps under `QT_QPA_PLATFORMTHEME=xdgdesktopportal`) then get a Dux Files window with the same sidebar, bookmarks, search and keys as the file manager. The dialog is a standalone window: Qt 6.12 and Quickshell do not import xdg-foreign handles, so it is not attached to the calling window and the `modal` option is ignored. Apps that draw their own dialogs are unchanged.

Nothing is switched on by install. `make install` ships `dfiles.portal` and the activation file `org.freedesktop.impl.portal.desktop.dfiles.service`; the choice is per user, in `~/.config/xdg-desktop-portal/<desktop>-portals.conf`. Settings, General, "Use Dux Files for open and save dialogs" or `dfiles portal install` write the per-user files when the packaged ones are missing, put `dfiles` first in that file's `org.freedesktop.impl.portal.FileChooser` key (keeping the previous backend as the fallback and every other line as it was) and record what changed, so Restore or `dfiles portal uninstall` puts it back. `dfiles portal status` shows each part. xdg-desktop-portal reads its configuration once at startup, so the change takes effect after `systemctl --user restart xdg-desktop-portal.service`; the UI and the CLI ask before doing that, because the restart cancels file dialogs other apps have open.

The portal activates the backend on demand through the activation file. Without the systemd user unit that means one hidden start of dfiles whenever xdg-desktop-portal starts (it quits again after 30 s with nothing to show) and a cold start for the first dialog; with the unit the resident instance answers at once. A niri rule that floats the chooser, which carries the app's own id and the title the app asked for:

```kdl
window-rule {
    match app-id=r#"^dev\.duxlabs\.files$"# title=r#"^(Open|Save|Select|Upload|Export|Import)"#
    open-floating true
}
```

Packagers: install `assets/portal/dfiles.portal` under `share/xdg-desktop-portal/portals/` and the service file under `share/dbus-1/services/` with the binary path substituted, and do not ship a `portals.conf` that prefers it. On NixOS and home-manager the module's `programs.dux-files.portal.enable` registers the backend and `portal.preferred` writes the preference into `xdg.portal.config.<portal.desktop>`. The Flatpak build cannot register the backend: Flatpak exports no `share/xdg-desktop-portal` files.

To open folders here from other apps:

```sh
xdg-mime default dev.duxlabs.files.desktop inode/directory
```

The installer does not run this.

## Runtime dependencies

Required: `quickshell` for the UI.

Optional, picked up from PATH when the daemon starts:

| package | binary | enables |
| --- | --- | --- |
| `bubblewrap` | `bwrap` | runs every external thumbnailer in a sandbox, as gnome-desktop does |
| `ffmpegthumbnailer` | `ffmpegthumbnailer` | video thumbnails |
| `poppler` (`poppler-utils` on debian) | `pdftoppm` | pdf thumbnails |
| `gvfs` | | network shares and `trash://` on removable media |
| `udisks2` | | the devices sidebar |

Images in png, jpeg, gif, bmp, tiff and webp are decoded in-process. Every other format goes through the freedesktop `thumbnailers/*.thumbnailer` entries under the XDG data dirs, so packages that ship one (glycin for heif, avif, jxl and ico, gsf for office documents, evince or totem on gnome) work without configuration. `dfiles ipc system.capabilities` lists what was found under `thumbnailers`.

## CLI

These commands talk to the running dfiles when there is one and run the same code in-process otherwise; `dfiles ipc` needs the running daemon. `--json` prints machine-readable output.

```sh
dfiles trash put|list|count|restore|empty
dfiles mime query|default|set
dfiles devices list|mount|unmount|eject
dfiles search <root> <query>
dfiles ipc <method> key=value...
```

Jobs started from the CLI answer their own prompts: a restore that finds its original path taken keeps both (`--conflict skip|replace` to change that), and an item that fails is skipped, reported, and makes the command exit 1.
