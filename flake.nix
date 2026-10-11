{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      treefmt-nix,
      ...
    }:
    let
      systems = [
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
        "x86_64-linux"
      ];
      forEachSystem = nixpkgs.lib.genAttrs systems;
      perSystem = forEachSystem (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
          treefmtEval = treefmt-nix.lib.evalModule pkgs ./treefmt.nix;
          package = pkgs.buildGoModule {
            pname = "agevault";
            version = "1.2.0";
            src = ./.;
            vendorHash = "sha256-yt4K+EcoZMJ36E5qolZdbDoeWp1WhVzppVx6nDrFq2s=";
            nativeBuildInputs = [ pkgs.installShellFiles ];
            preInstall = ''
              installManPage doc/*.1
            '';
          };
        in
        {
          inherit package;
          devShell = pkgs.mkShell {
            inputsFrom = [ package ];
            packages = with pkgs; [
              gotools
              ronn
            ];
          };
          fmt = treefmtEval.config.build.check self;
          formatter = treefmtEval.config.build.wrapper;
        }
      );
    in
    {
      packages = forEachSystem (system: {
        default = perSystem.${system}.package;
      });
      devShells = forEachSystem (system: {
        default = perSystem.${system}.devShell;
      });
      checks = forEachSystem (system: {
        inherit (perSystem.${system}) package fmt;
      });
      formatter = forEachSystem (system: perSystem.${system}.formatter);
    };
}
