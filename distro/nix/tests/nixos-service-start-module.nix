{
  self,
  pkgs,
  ...
}:
pkgs.testers.runNixOSTest {
  name = "dux-files-nixos-service-start-module";

  nodes.machine = {
    imports = [
      self.nixosModules.dux-files
    ];

    users.users.duxlabs = {
      isNormalUser = true;
      extraGroups = [ "wheel" ];
    };

    programs.dux-files = {
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
