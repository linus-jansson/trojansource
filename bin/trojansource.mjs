#!/usr/bin/env node

import { createRequire } from "node:module";
import { dirname, join } from "node:path";
import { spawnSync } from "node:child_process";

const require = createRequire(import.meta.url);
const platformPackages = {
	"darwin-arm64": "@linus_janns/trojansource-darwin-arm64",
	"darwin-x64": "@linus_janns/trojansource-darwin-x64",
	"linux-arm64": "@linus_janns/trojansource-linux-arm64",
	"linux-x64": "@linus_janns/trojansource-linux-x64",
	"win32-arm64": "@linus_janns/trojansource-win32-arm64",
	"win32-x64": "@linus_janns/trojansource-win32-x64",
};

const target = `${process.platform}-${process.arch}`;
const packageName = platformPackages[target];

if (!packageName) {
	console.error(`trojansource does not support ${target}.`);
	process.exitCode = 1;
} else {
	let packageManifest;
	try {
		packageManifest = require.resolve(`${packageName}/package.json`);
	} catch {
		console.error(
			`The native package for ${target} (${packageName}) was not installed. Reinstall trojansource for this platform.`,
		);
		process.exitCode = 1;
	}

	if (packageManifest) {
		const executable = join(
			dirname(packageManifest),
			"bin",
			process.platform === "win32" ? "trojansource.exe" : "trojansource",
		);
		const child = spawnSync(executable, process.argv.slice(2), {
			stdio: "inherit",
		});
		if (child.error) {
			console.error(`Unable to start trojansource: ${child.error.message}`);
			process.exitCode = 1;
		} else {
			process.exitCode = child.status ?? 1;
		}
	}
}
