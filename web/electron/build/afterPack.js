// Guard against Electron's default app icon leaking into unsigned local builds.
const fs = require("node:fs");
const path = require("node:path");
const { execFileSync } = require("node:child_process");

module.exports = async function afterPack(context) {
  if (context.electronPlatformName !== "darwin") return;

  const appName = context.packager.appInfo.productFilename;
  const appDir = path.join(context.appOutDir, `${appName}.app`);
  const infoPlist = path.join(appDir, "Contents", "Info.plist");
  const electronIcon = path.join(appDir, "Contents", "Resources", "electron.icns");

  let iconFile = "";
  try {
    iconFile = execFileSync(
      "/usr/libexec/PlistBuddy",
      ["-c", "Print :CFBundleIconFile", infoPlist],
      {
        encoding: "utf8",
        stdio: ["ignore", "pipe", "ignore"],
      },
    ).trim();
  } catch {
    // The key may be absent on future builder versions or non-mac targets.
  }

  if (iconFile === "electron.icns" || iconFile === "electron") {
    try {
      execFileSync("/usr/libexec/PlistBuddy", ["-c", "Delete :CFBundleIconFile", infoPlist], {
        stdio: "ignore",
      });
    } catch {
      // Best effort only; the build should not fail on a future plist shape.
    }
  }

  fs.rmSync(electronIcon, { force: true });
};
