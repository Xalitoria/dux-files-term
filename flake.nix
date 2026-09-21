{
  description = "DankFiles";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    dank-qml-common = {
      url = "github:AvengeMedia/dank-qml-common";
      flake = false;
    };
  };

  outputs =
    { self, nixpkgs, dank-qml-common, ... }:
    let
      goModVersion =
        let
          content = builtins.readFile ./core/go.mod;
          lines = builtins.filter builtins.isString (builtins.split "\n" content);
          goLines = builtins.filter (l: builtins.match "go [0-9]+\\..*" l != null) lines;
          matched =
            if goLines != [ ] then builtins.match "go ([0-9]+)\\.([0-9]+).*" (builtins.head goLines) else null;
        in
        if matched != null then
          {
            major = builtins.elemAt matched 0;
            minor = builtins.elemAt matched 1;
          }
        else
          {
            major = "1";
            minor = "26";
          };
      goForPkgs = pkgs: pkgs.${"go_${goModVersion.major}_${goModVersion.minor}"};

      forEachSystem =
        fn:
        nixpkgs.lib.genAttrs [ "aarch64-linux" "x86_64-linux" ] (
          system: fn system nixpkgs.legacyPackages.${system}
        );

      mkModuleWithDfilesPkgs =
        modulePath:
        args@{ pkgs, ... }:
        {
          imports = [
            (import modulePath (args // { dfilesPkgs = buildDfilesPkgs pkgs; }))
          ];
        };

      mkDfiles =
        pkgs:
        (
          let
            version = "0.1.0";
          in
          (pkgs.buildGoModule.override { go = goForPkgs pkgs; }) {
            inherit version;
            pname = "dankfiles";
            src = ./.;
            modRoot = "core";
            vendorHash = pkgs.lib.fakeHash;

            subPackages = [ "cmd/dfiles" ];

            tags = [ "withshell" ];

            # Mirror `make -C core sync-shell`. The flake src excludes submodule
            # content, so the DankCommon symlink is replaced with the pinned input.
            postPatch = ''
              rm -rf core/internal/shellembed/dist
              cp -r quickshell core/internal/shellembed/dist
              rm -f core/internal/shellembed/dist/DankCommon
              cp -r ${dank-qml-common}/DankCommon core/internal/shellembed/dist/DankCommon
              chmod -R u+w core/internal/shellembed/dist/DankCommon
              rm -rf core/internal/shellembed/dist/scripts \
                core/internal/shellembed/dist/tests \
                core/internal/shellembed/dist/.claude
              rm -f core/internal/shellembed/dist/.qmlls.ini \
                core/internal/shellembed/dist/translations/extract_translations.py
            '';

            ldflags = [
              "-s"
              "-w"
              "-X main.Version=${version}"
              "-X main.Commit=${self.shortRev or self.dirtyShortRev or "dirty"}"
            ];

            nativeBuildInputs = with pkgs; [
              installShellFiles
            ];

            postInstall = ''
              install -Dm644 ${./assets/com.danklinux.dankfiles.desktop} \
                $out/share/applications/com.danklinux.dankfiles.desktop
              install -Dm644 ${./assets/com.danklinux.dankfiles.svg} \
                $out/share/icons/hicolor/scalable/apps/com.danklinux.dankfiles.svg

              install -Dm644 ${./assets/systemd/dfiles.service} \
                $out/lib/systemd/user/dfiles.service
              substituteInPlace $out/lib/systemd/user/dfiles.service \
                --replace-fail /usr/bin/dfiles $out/bin/dfiles

              installShellCompletion --cmd dfiles \
                --bash <($out/bin/dfiles completion bash) \
                --fish <($out/bin/dfiles completion fish) \
                --zsh <($out/bin/dfiles completion zsh)
            '';

            meta = {
              description = "File manager for the dank desktop";
              homepage = "https://github.com/AvengeMedia/dankfiles";
              license = pkgs.lib.licenses.mit;
              mainProgram = "dfiles";
              platforms = pkgs.lib.platforms.linux;
            };
          }
        );

      buildDfilesPkgs = pkgs: {
        dankfiles = mkDfiles pkgs;
      };
    in
    {
      packages = forEachSystem (
        system: pkgs: {
          dankfiles = mkDfiles pkgs;
          default = self.packages.${system}.dankfiles;
        }
      );

      lib = { inherit mkDfiles buildDfilesPkgs; };

      homeModules.dank-files = mkModuleWithDfilesPkgs ./distro/nix/home.nix;

      homeModules.default = self.homeModules.dank-files;

      nixosModules.dank-files = mkModuleWithDfilesPkgs ./distro/nix/nixos.nix;

      nixosModules.default = self.nixosModules.dank-files;

      devShells = forEachSystem (
        system: pkgs: {
          default = pkgs.mkShell {
            GOTOOLCHAIN = "auto";

            buildInputs = with pkgs; [
              (goForPkgs pkgs)
              gopls
              delve
              go-tools
              gnumake

              quickshell

              prek
              uv
              python3

              nixd
              nil
            ];

            shellHook = ''
              touch quickshell/.qmlls.ini 2>/dev/null
              if [ ! -f .git/hooks/pre-commit ]; then prek install; fi
            '';
          };
        }
      );
    };
}
