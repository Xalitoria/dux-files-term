{
  config,
  pkgs,
  lib,
  ...
}@args:
let
  cfg = config.programs.dank-files;
  jsonFormat = pkgs.formats.json { };
in
{
  imports = [ (import ./options.nix args) ];

  options.programs.dank-files.settings = lib.mkOption {
    type = jsonFormat.type;
    default = { };
    description = "Dank Files UI settings written to ~/.config/dfiles/ui-settings.json.";
    example = lib.literalExpression ''
      {
        themeMode = "dark";
        radiusStrength = 50;
      }
    '';
  };

  config = lib.mkIf cfg.enable {
    programs.dank-files.systemd.target = lib.mkDefault config.wayland.systemd.target;

    home.packages = [
      cfg.package
      cfg.quickshell.package
    ];

    dbus.packages = [ cfg.package ];

    xdg.portal.extraPortals = lib.mkIf cfg.portal.enable [ cfg.package ];
    xdg.portal.config.${cfg.portal.desktop}."org.freedesktop.impl.portal.FileChooser" = lib.mkIf (cfg.portal.enable && cfg.portal.preferred) [ "dfiles" ];

    systemd.user.services.dfiles = lib.mkIf cfg.systemd.enable {
      Unit = {
        Description = "DankFiles";
        PartOf = [ cfg.systemd.target ];
        After = [ cfg.systemd.target ];
      };

      Service = {
        ExecStart = lib.getExe cfg.package + " run --session --hidden --keep-running";
        Restart = "on-failure";
        RestartSec = "2";
        Slice = "app.slice";
      };

      Install.WantedBy = [ cfg.systemd.target ];
    };

    xdg.configFile."dfiles/ui-settings.json" = lib.mkIf (cfg.settings != { }) {
      source = jsonFormat.generate "ui-settings.json" cfg.settings;
    };
  };
}
