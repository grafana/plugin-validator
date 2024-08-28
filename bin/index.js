#!/usr/bin/env node
const fs = require("fs");
const path = require("path");
const zlib = require("zlib");
const https = require("https");
const tar = require("tar");
const { spawnSync } = require("child_process");

const packageJson = require("./package.json");
// const version = packageJson.version;
const version = "0.17.3";
const urlTemplate = packageJson.binWrapper.urlTemplate;
const binaryName =
  process.platform === "win32"
    ? packageJson.binWrapper.name + ".exe"
    : packageJson.binWrapper.name;

const downloadPath = path.join(__dirname, ".bin");
const binaryPath = path.join(downloadPath, binaryName);

const ARCH_MAPPING = {
  ia32: "386",
  x64: "amd64",
  arm: "arm",
  arm64: "arm64",
};

function getPlatformSpecificDownloadUrl(platform, arch) {
  let finalArch = ARCH_MAPPING[arch] ?? arch;

  return urlTemplate
    .replaceAll("{{version}}", version)
    .replaceAll("{{platform}}", platform)
    .replaceAll("{{arch}}", finalArch);
}

function downloadFile(fileUrl, outputFolder) {
  const fileName = path.basename(new URL(fileUrl).pathname);
  const outputPath = path.join(outputFolder, fileName);

  // Check if the file already exists
  if (fs.existsSync(outputPath)) {
    return Promise.resolve(outputPath);
  }

  return new Promise((resolve, reject) => {
    const download = (urlToDownload) => {
      https
        .get(urlToDownload, (response) => {
          if (
            response.statusCode >= 300 &&
            response.statusCode < 400 &&
            response.headers.location
          ) {
            // Handle redirection
            const redirectedUrl = new URL(
              response.headers.location,
              urlToDownload,
            ).toString();
            download(redirectedUrl);
          } else if (response.statusCode === 200) {
            const fileStream = fs.createWriteStream(outputPath);
            response.pipe(fileStream);

            fileStream.on("finish", () => {
              fileStream.close(() => {
                resolve(outputPath);
              });
            });
          } else {
            reject(
              new Error(
                `Failed to download '${fileUrl}' (${response.statusCode})`,
              ),
            );
          }
        })
        .on("error", reject);
    };

    download(fileUrl);
  });
}

function extractTarGz(tarGzPath, outputDir) {
  const tarGzStream = fs.createReadStream(tarGzPath);
  const unzip = zlib.createGunzip();
  const extract = tar.x({ cwd: outputDir });

  tarGzStream
    .pipe(unzip)
    .pipe(extract)
    .on("error", (err) => console.error("Extraction error:", err))
    .on("finish", () => console.log("Extraction complete."));

  return outputDir;
}

async function ensureBinary() {
  if (fs.existsSync(binaryPath)) {
    return;
  }

  const platformSpecificDownloadUrl = getPlatformSpecificDownloadUrl(
    process.platform,
    process.arch,
  );

  if (!fs.existsSync(downloadPath)) {
    fs.mkdirSync(downloadPath);
  }

  console.log(`Downloading ${platformSpecificDownloadUrl}`);

  let tarGzPath;
  try {
    tarGzPath = await downloadFile(platformSpecificDownloadUrl, downloadPath);
  } catch (e) {
    console.error(e);
    throw new Error(`Failed to download ${platformSpecificDownloadUrl}`);
  }
  try {
    console.log(`Extracting ${tarGzPath} to ${downloadPath}`);
    extractTarGz(tarGzPath, downloadPath);
  } catch (e) {
    console.error(e);
    throw new Error(`Failed to extract ${tarGzPath} to ${downloadPath}`);
  }

  // Check if the binary exists
  if (!fs.existsSync(path.join(downloadPath, binaryName))) {
    throw new Error(
      `Binary not found at ${downloadPath}. There might be a problem with the release files.`,
    );
  }

  // make the binary executable
  fs.chmodSync(path.join(downloadPath, binaryName), 0o755);

  async function main() {

    try {
      await ensureBinary();
      // run the binary
      const args = process.argv.slice(2);
      spawnSync(downloadPath, args, {
        stdio: "inherit",
      });
    } catch (e) {
      console.error(e);
      process.exit(1);
    }

  }

  main();
