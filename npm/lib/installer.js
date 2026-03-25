const crypto = require("node:crypto");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const https = require("node:https");
const { pipeline } = require("node:stream/promises");
const { spawn } = require("node:child_process");

const REPO = "707/petti";
const BINARY = "petti";

function normalizePlatform(platform) {
  if (platform === "darwin" || platform === "linux") {
    return platform;
  }
  throw new Error(`unsupported platform: ${platform}`);
}

function normalizeArch(arch) {
  if (arch === "x64") {
    return "amd64";
  }
  if (arch === "arm64") {
    return "arm64";
  }
  throw new Error(`unsupported architecture: ${arch}`);
}

function versionTagFor(version) {
  return version.startsWith("v") ? version : `v${version}`;
}

function assetNameFor(version, platform, arch) {
  return `${BINARY}_${version.replace(/^v/, "")}_${platform}_${arch}.tar.gz`;
}

function releaseURLFor(version, platform, arch) {
  const cleanVersion = version.replace(/^v/, "");
  return `https://github.com/${REPO}/releases/download/${versionTagFor(cleanVersion)}/${assetNameFor(cleanVersion, platform, arch)}`;
}

function checksumsURLFor(version) {
  return `https://github.com/${REPO}/releases/download/${versionTagFor(version.replace(/^v/, ""))}/checksums.txt`;
}

function parseChecksums(contents) {
  const checksums = new Map();
  for (const line of contents.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }
    const [sha, name] = trimmed.split(/\s+/, 2);
    if (sha && name) {
      checksums.set(name, sha);
    }
  }
  return checksums;
}

function cacheRoot() {
  if (process.env.XDG_CACHE_HOME) {
    return path.join(process.env.XDG_CACHE_HOME, BINARY);
  }
  return path.join(os.homedir(), ".cache", BINARY);
}

function binaryPathFor(version, platform, arch) {
  return path.join(cacheRoot(), version.replace(/^v/, ""), `${platform}-${arch}`, BINARY);
}

function fetch(url) {
  return new Promise((resolve, reject) => {
    https
      .get(url, (response) => {
        if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
          response.resume();
          resolve(fetch(response.headers.location));
          return;
        }
        if (response.statusCode !== 200) {
          reject(new Error(`download failed for ${url}: HTTP ${response.statusCode}`));
          response.resume();
          return;
        }
        resolve(response);
      })
      .on("error", reject);
  });
}

async function downloadText(url) {
  const response = await fetch(url);
  let data = "";
  response.setEncoding("utf8");
  for await (const chunk of response) {
    data += chunk;
  }
  return data;
}

async function downloadFile(url, destination) {
  const response = await fetch(url);
  await fs.promises.mkdir(path.dirname(destination), { recursive: true });
  await pipeline(response, fs.createWriteStream(destination));
}

async function sha256For(filePath) {
  const hash = crypto.createHash("sha256");
  const stream = fs.createReadStream(filePath);
  for await (const chunk of stream) {
    hash.update(chunk);
  }
  return hash.digest("hex");
}

function extractTarGz(archivePath, destination) {
  return new Promise((resolve, reject) => {
    fs.mkdirSync(destination, { recursive: true });
    const child = spawn("tar", ["-xzf", archivePath, "-C", destination], { stdio: "inherit" });
    child.on("exit", (code) => {
      if (code === 0) {
        resolve();
        return;
      }
      reject(new Error(`tar exited with code ${code}`));
    });
    child.on("error", reject);
  });
}

async function ensureBinary(version) {
  const platform = normalizePlatform(process.platform);
  const arch = normalizeArch(process.arch);
  const binaryPath = binaryPathFor(version, platform, arch);
  if (fs.existsSync(binaryPath)) {
    return binaryPath;
  }

  const cleanVersion = version.replace(/^v/, "");
  const assetName = assetNameFor(cleanVersion, platform, arch);
  const workDir = path.join(cacheRoot(), ".downloads", cleanVersion, `${platform}-${arch}`);
  const archivePath = path.join(workDir, assetName);
  const extractDir = path.join(workDir, "extract");

  await fs.promises.rm(workDir, { recursive: true, force: true });
  await fs.promises.mkdir(workDir, { recursive: true });

  const [checksumsText] = await Promise.all([
    downloadText(checksumsURLFor(cleanVersion)),
    downloadFile(releaseURLFor(cleanVersion, platform, arch), archivePath),
  ]);

  const checksums = parseChecksums(checksumsText);
  const expected = checksums.get(assetName);
  if (!expected) {
    throw new Error(`missing checksum for ${assetName}`);
  }

  const actual = await sha256For(archivePath);
  if (actual !== expected) {
    throw new Error(`checksum mismatch for ${assetName}`);
  }

  await extractTarGz(archivePath, extractDir);
  const extractedBinary = path.join(extractDir, BINARY);
  await fs.promises.mkdir(path.dirname(binaryPath), { recursive: true });
  await fs.promises.copyFile(extractedBinary, binaryPath);
  await fs.promises.chmod(binaryPath, 0o755);
  await fs.promises.rm(workDir, { recursive: true, force: true });
  return binaryPath;
}

module.exports = {
  assetNameFor,
  binaryPathFor,
  checksumsURLFor,
  ensureBinary,
  normalizeArch,
  normalizePlatform,
  parseChecksums,
  releaseURLFor,
  versionTagFor,
};
