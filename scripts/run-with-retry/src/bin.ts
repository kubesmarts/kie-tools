#!/usr/bin/env node

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import * as yargs from "yargs";
import { hideBin } from "yargs/helpers";
import { spawn } from "child_process";

function log(isSilent: boolean, ...args: any[]) {
  if (!isSilent) {
    console.info(`[run-with-retry]`, ...args);
  }
}

// pnpm/Yarn/NPM default to CMD on Windows, which is not ideal for the
// Command Substitution syntax used across kie-tools scripts. Force PowerShell,
// mirroring what @kie-tools-scripts/run-script-if does.
function shell() {
  return process.platform === "win32" ? { shell: "powershell.exe" } : {};
}

// Runs a single command string. Resolves on exit code 0, rejects with { code }
// otherwise. The command's own stdout/stderr are inherited so the original
// error is always visible to the caller.
function runCommand(commandString: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const parts = commandString.split(" ").filter((arg) => arg.trim().length > 0);
    const bin = parts[0];
    const args = parts.slice(1);
    const child = spawn(bin, args, { stdio: "inherit", ...shell() });
    child.on("error", () => reject({ code: 1 }));
    child.on("exit", (code) => (code === 0 ? resolve() : reject({ code: code ?? 1 })));
  });
}

// Runs all commands in order. Rejects as soon as one of them fails.
async function runBatch(commandStrings: string[]): Promise<void> {
  for (const commandString of commandStrings) {
    await runCommand(commandString);
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function main() {
  const argv = yargs(hideBin(process.argv))
    .epilog(
      `
CLI tool to run a command (or a batch of commands) with bounded retries and
backoff. Useful for steps that can fail transiently, e.g. resolving Go modules
against sum.golang.org during 'go mod tidy'.

The commands passed to --then are run in order, as a batch. If any of them
fails, the whole batch is retried from the start after a backoff delay. After
the last attempt the process exits with the failing command's original exit
code, so genuine (non-transient) breakage is NOT masked.

Delays are given in seconds between attempts (never after the last attempt).
The last delay value repeats, so raising --retries needs no other change.

Example:
$ run-with-retry --retries 5 --delays "5,15,30,60" --then "go work sync" --then "go mod tidy"
    `
    )
    .strict()
    .options({
      then: {
        array: true,
        required: true,
        type: "string",
        description: "Command(s) to run in order. Retried together as a batch.",
      },
      retries: {
        default: 5,
        type: "number",
        description: "Maximum number of attempts before failing.",
      },
      delays: {
        default: "5,15,30,60",
        type: "string",
        description: "Comma-separated backoff delays in seconds, applied between attempts. The last value repeats.",
      },
      silent: {
        default: false,
        type: "boolean",
        description: "Hide [run-with-retry] info logs. Logs from the commands themselves still show.",
      },
    })
    .parseSync();

  const commandStrings = argv.then;
  const maxAttempts = Math.max(1, Math.floor(argv.retries));

  const delaysSec = argv.delays
    .split(",")
    .map((d) => Number(d.trim()))
    .filter((d) => Number.isFinite(d) && d >= 0);
  if (delaysSec.length === 0) {
    delaysSec.push(0);
  }

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      log(argv.silent, `Attempt ${attempt}/${maxAttempts}: ${commandStrings.map((c) => `'${c}'`).join(" && ")}`);
      await runBatch(commandStrings);
      log(argv.silent, `Succeeded on attempt ${attempt}/${maxAttempts}.`);
      return;
    } catch (e: any) {
      const code = e && typeof e.code === "number" ? e.code : 1;
      if (attempt >= maxAttempts) {
        log(argv.silent, `Failed after ${maxAttempts} attempt(s). Exiting with code ${code}.`);
        process.exit(code); // original error already printed via inherited stdio
      }
      const waitSec = delaysSec[Math.min(attempt - 1, delaysSec.length - 1)];
      log(argv.silent, `Attempt ${attempt} failed (code ${code}). Retrying in ${waitSec}s...`);
      await sleep(waitSec * 1000);
    }
  }
}

main().catch((e) => {
  process.exit(e && typeof e.code === "number" ? e.code : 1);
});
