{
  lib,
  dfilesPkgs,
  pkgs,
  ...
}:
let
  inherit (lib) types;
in
{
  options.programs.dank-files = {
    enable = lib.mkEnableOption "DankFiles";

    package = lib.mkPackageOption dfilesPkgs "dankfiles" {
      extraDescription = "The DankFiles package to use (defaults to be built from source)";
    };

    quickshell = {
      package = lib.mkPackageOption pkgs "quickshell" {
        extraDescription = "The Quickshell package used to launch the file manager UI";
      };
    };

    portal = {
      enable = lib.mkEnableOption "serving open and save dialogs through xdg-desktop-portal (registers the dfiles backend)";

      preferred = lib.mkOption {
        type = types.bool;
        default = false;
        description = "Prefer dfiles for org.freedesktop.impl.portal.FileChooser in the portal configuration named by portal.desktop.";
      };

      desktop = lib.mkOption {
        type = types.str;
        default = "common";
        description = "Which xdg.portal.config entry receives the FileChooser preference (a lowercased XDG_CURRENT_DESKTOP name, or common).";
      };
    };

    systemd = {
      enable = lib.mkEnableOption "DankFiles systemd startup";

      target = lib.mkOption {
        type = types.str;
        default = "graphical-session.target";
        description = "Systemd target that will automatically start dfiles.";
      };

      restartIfChanged = lib.mkOption {
        type = types.bool;
        default = true;
        description = "Auto-restart dfiles.service when dank-files changes";
      };
    };
  };
}
