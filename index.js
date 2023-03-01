var binwrap = require("binwrap");
var path = require("path");

var packageInfo = require(path.join(__dirname, "package.json"));
var version = packageInfo.version;
var root = `https://github.com/grafana/plugin-validator/releases/download/v${version}`;
var root = "https://dl.bintray.com/me/myApp/" + version;

module.exports = binwrap({
  dirname: __dirname,
  binaries: ["myapp-cli"],
  urls: {
    "darwin-x64": root + "/mac-x64.tgz",
    "linux-x64": root + "/linux-x64.tgz",
    "win32-x64": root + "/win-i386.zip",
    "win32-ia32": root + "/win-i386.zip",
  },
});
