{
  self,
  pkgs,
  ...
}:
pkgs.testers.runNixOSTest {
  name = "dankfiles-nixos-service-start-module";

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
      systemd = {
        enable = true;
        target = "default.target";
      };
    };

    system.stateVersion = "25.11";
  };

  testScript = ''
    machine.wait_for_unit("multi-user.target")

    unit = "/run/current-system/sw/share/systemd/user/dfiles.service"
    machine.succeed(f"test -f {unit}")
    machine.succeed(f"grep -q 'run --session --hidden --keep-running' {unit}")
    machine.succeed("test -e /etc/systemd/user/default.target.wants/dfiles.service")
  '';
}
