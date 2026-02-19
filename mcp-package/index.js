#!/usr/bin/env node
const fs = require("fs");
const path = require("path");
const zlib = require("zlib");
const https = require("https");
const tar = require("tar");
const { spawn } = require("child_process");

const packageJson = require("./package.json");
const version = packageJson.version;
const urlTemplate = packageJson.binWrapper.urlTemplate;
const binaryName =
  process.platform === "win32"
    ? packageJson.binWrapper.name + ".exe"
    : packageJson.binWrapper.name;

const downloadPath = path.join(__dirname, ".bin");
const binaryPath = path.join(downloadPath, binaryName);

const PLATFORM_MAPPING = {
  win32: "windows",
};

const ARCH_MAPPING = {
  ia32: "386",
  x64: "amd64",
  arm: "arm",
  arm64: "arm64",
};

function getPlatformSpecificDownloadUrl(platform, arch) {
  const finalPlatform = PLATFORM_MAPPING[platform] || platform;
  const finalArch = ARCH_MAPPING[arch] || arch;

  return urlTemplate
    .replaceAll("{{version}}", version)
    .replaceAll("{{platform}}", finalPlatform)
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

function extractTarGz(filePath, outputDir) {
  return new Promise((resolve, reject) => {
    fs.createReadStream(filePath)
      .pipe(zlib.createGunzip())
      .pipe(tar.extract({ cwd: outputDir }))
      .on("error", reject)
      .on("finish", resolve);
  });
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
    fs.mkdirSync(downloadPath, { recursive: true });
  }

  let tarGzPath;
  try {
    tarGzPath = await downloadFile(platformSpecificDownloadUrl, downloadPath);
  } catch (e) {
    console.error(e);
    throw new Error(`Failed to download ${platformSpecificDownloadUrl}`);
  }
  try {
    await extractTarGz(tarGzPath, downloadPath);
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
}

async function main() {
  try {
    await ensureBinary();
    // run the binary
    const args = process.argv.slice(2);
    const child = spawn(binaryPath, args, {
      stdio: "inherit",
    });
    child.on("exit", (code) => {
      process.exit(code);
    });
  } catch (e) {
    console.error(e);
    process.exit(1);
  }
}

main();
