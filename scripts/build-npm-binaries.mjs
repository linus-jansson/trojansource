import { execFileSync } from "node:child_process";
import { mkdirSync, rmSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const targets = [
	["darwin", "arm64"],
	["darwin", "amd64"],
	["linux", "arm64"],
	["linux", "amd64"],
	["windows", "arm64"],
	["windows", "amd64"],
];

for (const [os, architecture] of targets) {
	const npmOS = os === "windows" ? "win32" : os;
	const npmArchitecture = architecture === "amd64" ? "x64" : architecture;
	const outputDirectory = resolve(
		root,
		"packages",
		`${npmOS}-${npmArchitecture}`,
		"bin",
	);
	const executable = resolve(
		outputDirectory,
		os === "windows" ? "trojansource.exe" : "trojansource",
	);

	rmSync(outputDirectory, { force: true, recursive: true });
	mkdirSync(outputDirectory, { recursive: true });
	execFileSync("go", ["build", "-o", executable, "./cmd/trojansource"], {
		cwd: root,
		env: {
			...process.env,
			CGO_ENABLED: "0",
			GOARCH: architecture,
			GOOS: os,
		},
		stdio: "inherit",
	});
}
