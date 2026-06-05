{ lib, buildGoModule }:
buildGoModule {
  pname = "comview";
  version = "0.1.0";

  src = ./.;

  vendorHash = "sha256-f7Q4+Bd22xnxkOWjgv4TzPmgZTNHhYvMtoVyl9anGzc=";

  meta = {
    description = "Terminal UI for viewing GitHub pull request reviews";
    homepage = "https://github.com/rockorager/comview";
    license = lib.licenses.mit;
    mainProgram = "comview";
    maintainers = [ "AlexBN" ];
    platforms = lib.platforms.all;
  };
}
