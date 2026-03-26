const test = require("node:test");
const assert = require("node:assert/strict");

const {
  assetNameFor,
  releaseURLFor,
  normalizeArch,
  normalizePlatform,
  parseChecksums,
  versionTagFor,
} = require("../lib/installer");

test("normalizePlatform maps supported platforms", () => {
  assert.equal(normalizePlatform("darwin"), "darwin");
  assert.equal(normalizePlatform("linux"), "linux");
  assert.throws(() => normalizePlatform("win32"), /unsupported platform/i);
});

test("normalizeArch maps supported architectures", () => {
  assert.equal(normalizeArch("x64"), "amd64");
  assert.equal(normalizeArch("arm64"), "arm64");
  assert.throws(() => normalizeArch("ia32"), /unsupported architecture/i);
});

test("versionTagFor prefixes versions with v", () => {
  assert.equal(versionTagFor("0.6.4"), "v0.6.4");
  assert.equal(versionTagFor("v0.6.4"), "v0.6.4");
});

test("assetNameFor builds the release archive name", () => {
  assert.equal(assetNameFor("0.6.4", "darwin", "amd64"), "petti_0.6.4_darwin_amd64.tar.gz");
  assert.equal(assetNameFor("0.6.4", "linux", "arm64"), "petti_0.6.4_linux_arm64.tar.gz");
});

test("releaseURLFor points at the GitHub release asset", () => {
  assert.equal(
    releaseURLFor("0.6.4", "darwin", "arm64"),
    "https://github.com/707/petti/releases/download/v0.6.4/petti_0.6.4_darwin_arm64.tar.gz",
  );
});

test("parseChecksums reads release checksum files", () => {
  const checksums = parseChecksums([
    "aaa  petti_0.6.4_darwin_amd64.tar.gz",
    "bbb  petti_0.6.4_darwin_arm64.tar.gz",
  ].join("\n"));

  assert.equal(checksums.get("petti_0.6.4_darwin_amd64.tar.gz"), "aaa");
  assert.equal(checksums.get("petti_0.6.4_darwin_arm64.tar.gz"), "bbb");
});
