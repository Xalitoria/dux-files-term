{
  self,
  pkgs,
  ...
}:
pkgs.testers.runNixOSTest {
  name = "dankfiles-nixos-module";

  nodes.machine = {
    imports = [
      self.nixosModules.dank-files
    ];

    users.users.danklinux = {
      isNormalUser = true;
      extraGroups = [ "wheel" ];
    };

    programs.dank-files = {
      enable = true;
      systemd.enable = true;
    };

    system.stateVersion = "25.11";
  };

  testScript = ''
    machine.wait_for_unit("multi-user.target")

    machine.succeed("command -v dfiles")
    machine.succeed("command -v qs")
    machine.succeed("su -- danklinux -c 'dfiles --help >/dev/null'")
    machine.succeed("test -f /run/current-system/sw/share/systemd/user/dfiles.service")
    machine.succeed("test -f /run/current-system/sw/share/dbus-1/services/com.danklinux.dankfiles.FileManager1.service")
  '';
}
