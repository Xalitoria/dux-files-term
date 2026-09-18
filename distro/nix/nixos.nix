{
  config,
  pkgs,
  lib,
  ...
}@args:
let
  cfg = config.programs.dank-files;
in
{
  imports = [ (import ./options.nix args) ];

  config = lib.mkIf cfg.enable {
    environment.systemPackages = [
      cfg.package
      cfg.quickshell.package
    ];

    systemd.packages = lib.mkIf cfg.systemd.enable [ cfg.package ];

    systemd.user.services.dfiles = lib.mkIf cfg.systemd.enable {
      wantedBy = [ cfg.systemd.target ];
      restartIfChanged = cfg.systemd.restartIfChanged;
      path = [ cfg.quickshell.package ];
    };
  };
}
