{
  description = "Fintech Wallet API - Go REST API with PostgreSQL HA & Redis";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];

      forEachSupportedSystem = f: nixpkgs.lib.genAttrs supportedSystems (system: f {
        pkgs = import nixpkgs { inherit system; };
      });
    in
    {
      # Development Environment (nix develop)
      devShells = forEachSupportedSystem ({ pkgs }: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go             # Go compiler
            gopls          # Go Language Server
            gotools        # goimports, etc.
            golangci-lint  # Linter

            postgresql     # psql for debugging
            redis          # redis-cli for debugging
            docker-compose # Orchestrate containers
          ];

          shellHook = ''
            echo "============================================================"
            echo "💰 Fintech Wallet API - Dev Environment"
            echo "⚙️  Go version: $(go version)"
            echo "🐘 psql:        $(psql --version)"
            echo "============================================================"
          '';
        };
      });

      # Reproducible Build (nix build)
      packages = forEachSupportedSystem ({ pkgs }: {
        default = pkgs.buildGoModule {
          pname = "fintech-wallet-api";
          version = "0.1.0";
          src = ./.;

          # Set to pkgs.lib.fakeHash, run `nix build`, then paste the real hash.
          vendorHash = null;
        };
      });
    };
}
