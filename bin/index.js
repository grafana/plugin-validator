const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");

const packageJson = require("../package.json");
const binaryName =
  process.platform === "win32"
    ? packageJson.binWrapper.name + ".exe"
    : packageJson.binWrapper.name;

const binaryPath = path.join(__dirname, "..", ".bin", binaryName);

// check if the binary exists
if (!fs.existsSync(binaryPath)) {
  throw new Error(
    `Binary not found. There might be a problem with the release files.`,
  );
}

console.log("This is a test for the binary ");

// run the binary
const args = process.argv.slice(2);
const childProcess = spawnSync(binaryPath, args, {
  stdio: "inherit",
});
