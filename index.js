var binwrap = require("binwrap");
var path = require("path");

var packageInfo = require(path.join(__dirname, "package.json"));
var version = packageInfo.version;
var root = `https://github.com/grafana/plugin-validator/releases/download/v${version}`;

module.exports = binwrap({
  dirname: __dirname,
  binaries: ["plugincheck2"],
  urls: {
    // plugin-validator_0.7.3_darwin_amd64.tar.gz
    "darwin-x64":
      root + "/plugin-validator_" + version + "_darwin_amd64.tar.gz",
    "darwin-arm64":
      root + "/plugin-validator_" + version + "_darwin_arm64.tar.gz",
    "linux-x64": root + "/plugin-validator_" + version + "_linux_amd64.tar.gz",
    "linux-arm64":
      root + "/plugin-validator_" + version + "_linux_arm64.tar.gz",
    "win32-x64": root + "/plugin-validator_" + version + "_windows_amd64.zip",
    "win32-ia32": root + "/plugin-validator_" + version + "_windows_386.zip",
    "win32-arm64": root + "/plugin-validator_" + version + "_windows_arm64.zip",
  },
});
