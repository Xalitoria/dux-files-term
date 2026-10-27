{
  description = "Home Manager with Dux Files and its docked terminal";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    home-manager = {
      url = "github:nix-community/home-manager";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    dux-files = {
      url = "github:Xalitoria/dux-files-term";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    { nixpkgs, home-manager, dux-files, ... }:
    {
      homeConfigurations.example = home-manager.lib.homeManagerConfiguration {
        pkgs = nixpkgs.legacyPackages.x86_64-linux;
        modules = [
          dux-files.homeModules.default
          {
            home.username = "user";
            home.homeDirectory = "/home/user";
            home.stateVersion = "25.11";

            programs.dux-files = {
              enable = true;
              systemd.enable = true;
              portal.enable = true;
              settings.themeMode = "dark";
            };
          }
        ];
      };
    };
}
