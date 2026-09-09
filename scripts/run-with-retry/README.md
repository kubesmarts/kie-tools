<!--
   Licensed to the Apache Software Foundation (ASF) under one
   or more contributor license agreements.  See the NOTICE file
   distributed with this work for additional information
   regarding copyright ownership.  The ASF licenses this file
   to you under the Apache License, Version 2.0 (the
   "License"); you may not use this file except in compliance
   with the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing,
   software distributed under the License is distributed on an
   "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
   KIND, either express or implied.  See the License for the
   specific language governing permissions and limitations
   under the License.
-->

# @kie-tools-scripts/run-with-retry

CLI tool to run a command (or a batch of commands) with bounded retries and
backoff, so a step survives transient failures such as flaky network access
during dependency resolution.

It exists because kie-tools CI occasionally fails at bootstrap when a Go
package's `install` hook runs `go mod tidy` / `go work sync` and the online
checksum verification against `sum.golang.org` hits a transient network error,
which aborts the whole job. Wrapping the resolving step in `run-with-retry`
makes a transient blip retry instead of killing the job, while genuine breakage
still surfaces after the attempts are exhausted (the original exit code is
propagated).

## Usage

```
run-with-retry [--retries 5] [--delays "5,15,30,60"] [--silent] --then "<command>" [--then "<command>" ...]
```

- `--then` (required, repeatable): command(s) to run in order, retried together
  as a batch. If any fails, the whole batch is retried from the start.
- `--retries` (default `5`): maximum number of attempts.
- `--delays` (default `"5,15,30,60"`): comma-separated backoff delays in seconds,
  applied **between** attempts (never after the last one). The last value
  repeats, so raising `--retries` needs no other change.
- `--silent`: hide `[run-with-retry]` info logs (command output still shows).

### Example

```
run-with-retry --then "go work sync" --then "go mod tidy"
```

With the defaults this runs the two commands, and on failure retries the batch
with backoff 5s, 15s, 30s, 60s between the 5 attempts, then exits with the
original error code if still failing.
