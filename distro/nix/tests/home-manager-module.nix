{
  self,
  pkgs,
  ...
}:
let
  homeManagerNixosModule =
    (fetchTarball {
      url = "https://github.com/nix-community/home-manager/archive/c03e4752899e55705dfa63979abd885c582a5c48.tar.gz";
      sha256 = "0c3jm12hgqns8wlgid5pfyg5qn39lsf7adv6wv5swjh54mhi3qj0";
    })
    + "/nixos";
in
pkgs.testers.runNixOSTest {
  name = "dankfiles-home-manager-module";

  nodes.machine =
    { ... }:
    {
      imports = [ homeManagerNixosModule ];

      users.users.danklinux = {
        isNormalUser = true;
        createHome = true;
        home = "/home/danklinux";
        extraGroups = [ "wheel" ];
      };

      home-manager.useGlobalPkgs = true;
      home-manager.useUserPackages = true;

      home-manager.users.danklinux =
        { pkgs, ... }:
        {
          imports = [ self.homeModules.dank-files ];

          home.username = "danklinux";
          home.homeDirectory = "/home/danklinux";
          home.stateVersion = "25.11";

          programs.dank-files = {
            enable = true;
            systemd = {
              enable = true;
              target = "default.target";
            };

            settings = {
              themeMode = "dark";
              radiusStrength = 50;
            };
          };
        };

      system.stateVersion = "25.11";
    };

  testScript = ''
    import json

    machine.wait_for_unit("multi-user.target")

    machine.succeed("su -- danklinux -c 'command -v dfiles'")
    machine.succeed("su -- danklinux -c 'test -f ~/.config/systemd/user/dfiles.service'")
    machine.succeed("su -- danklinux -c 'test -L ~/.config/systemd/user/default.target.wants/dfiles.service'")

    machine.succeed("su -- danklinux -c 'test -f ~/.config/dfiles/ui-settings.json'")
    settings = json.loads(machine.succeed("su -- danklinux -c 'cat ~/.config/dfiles/ui-settings.json'"))
    t.assertEqual(settings["themeMode"], "dark")
    t.assertEqual(settings["radiusStrength"], 50)
  '';
}
