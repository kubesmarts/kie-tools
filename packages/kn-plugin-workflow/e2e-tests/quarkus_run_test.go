//go:build e2e_tests

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

package e2e_tests

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/apache/incubator-kie-tools/packages/kn-plugin-workflow/pkg/command"
	"github.com/apache/incubator-kie-tools/packages/kn-plugin-workflow/pkg/command/quarkus"
	"github.com/apache/incubator-kie-tools/packages/kn-plugin-workflow/pkg/common"
	"github.com/stretchr/testify/require"
)

var cfgTestInputPrepareQuarkusCreateRun = CfgTestInputQuarkusCreate{
	input: quarkus.CreateQuarkusProjectConfig{ProjectName: "new-project"},
}

var cfgTestInputPrepareQuarkusBuildRun = CfgTestInputQuarkusBuild{
	input: quarkus.BuildCmdConfig{
		Image: "dev.local/new-project",
	},
}

type cfgTestInputQuarkusRun struct {
	input quarkus.RunCmdConfig
}

var cfgTestInputQuarkusRun_Success = []cfgTestInputQuarkusRun{
	{input: quarkus.RunCmdConfig{PortMapping: "8081", OpenDevUI: false}},
	{input: quarkus.RunCmdConfig{OpenDevUI: true}},
}

func transformQuarkusRunCmdCfgToArgs(cfg quarkus.RunCmdConfig) []string {
	args := []string{"run"}
	if !cfg.OpenDevUI {
		args = append(args, "--open-dev-ui=false")
	}
	if cfg.PortMapping != "" {
		args = append(args, "--port", cfg.PortMapping)
	}
	return args
}

func getRunQuarkusProjectPort(t *testing.T, config cfgTestInputQuarkusRun) string {
	if config.input.PortMapping != "" {
		return config.input.PortMapping
	} else {
		projectDefaultPort, err := LookupFlagDefaultValue("port", command.NewRunCommand())
		require.NoErrorf(t, err, "Error: %v", err)
		return projectDefaultPort
	}
}

func TestQuarkusRunCommand(t *testing.T) {
	for testIndex, test := range cfgTestInputQuarkusRun_Success {
		t.Run(fmt.Sprintf("Test quarkus run project success index: %d", testIndex), func(t *testing.T) {
			defer CleanUpAndChdirTemp(t)
			RunQuarkusRunTest(t, cfgTestInputPrepareQuarkusCreateRun, cfgTestInputPrepareQuarkusBuildRun, test)
		})
	}
}

func RunQuarkusRunTest(t *testing.T, cfgTestInputPrepareQuarkusCreateRun CfgTestInputQuarkusCreate, cfgTestInputPrepareQuarkusBuild CfgTestInputQuarkusBuild, test cfgTestInputQuarkusRun) string {
	var err error

	// Create and build the quarkus project
	projectName := RunQuarkusCreateTest(t, cfgTestInputPrepareQuarkusCreateRun)
	if t.Failed() {
		return projectName
	}
	projectDir := filepath.Join(TempTestsPath, projectName)

	err = os.Chdir(projectDir)
	require.NoErrorf(t, err, "Expected nil error, got %v", err)
	WriteMavenConfigFileWithTailDirs(projectDir)

	cmd := exec.Command(KnExecutable)

	var wg sync.WaitGroup
	wg.Add(1)

	// runFailed is closed by the goroutine if the run process outputs a fatal error line.
	runFailed := make(chan string, 1)
	processDone := make(chan error, 1)

	// Run the `quarkus run` command, watching stdout for early-failure markers.
	go func() {
		defer wg.Done()
		_, runErr := executeKnWorkflowQuarkusWithCmdAndErrWatch(cmd, runFailed, transformQuarkusRunCmdCfgToArgs(test.input)...)
		processDone <- runErr
	}()

	// Check if the project is successfully run and accessible within a specified time limit.
	readyCheckURL := fmt.Sprintf("http://localhost:%s/q/health/ready", getRunQuarkusProjectPort(t, test))
	pollInterval := 5 * time.Second
	timeout := 10 * time.Minute
	ready := make(chan bool)
	t.Logf("Checking if project is ready at %s", readyCheckURL)
	go common.PollReadyCheckURL(readyCheckURL, pollInterval, ready)
	select {
	case <-ready:
		cmd.Process.Signal(os.Interrupt)
	case runErr := <-processDone:
		if runErr != nil && !IsSignalInterrupt(runErr) {
			t.Fatalf("quarkus run process exited with error before becoming ready: %v", runErr)
		}
	case errLine := <-runFailed:
		cmd.Process.Signal(os.Interrupt)
		t.Fatalf("quarkus run process reported a fatal error before becoming ready: %s", errLine)
	case <-time.After(timeout):
		t.Fatalf("Test case timed out after %s. The project was not ready within the specified time.", timeout)
		cmd.Process.Signal(os.Interrupt)
	}

	wg.Wait()

	return projectName
}

// executeKnWorkflowQuarkusWithCmdAndErrWatch runs `kn-workflow quarkus <args>` and streams stdout.
// If a line matching a known fatal-error pattern is seen, it sends that line to errCh so the
// caller can fail fast without waiting for the process to exit on its own.
func executeKnWorkflowQuarkusWithCmdAndErrWatch(cmd *exec.Cmd, errCh chan<- string, args ...string) (string, error) {
	newArgs := append([]string{"quarkus"}, args...)
	cmd.Args = append([]string{cmd.Path}, newArgs...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start process: %w", err)
	}

	var buf strings.Builder
	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line + "\n")
		if *TestPrintCmdOutput {
			fmt.Println(line)
		}
		// Detect the kn-workflow error prefix — means Maven failed and the process
		// is stuck in ReadyCheck; signal failure immediately rather than timing out.
		if strings.HasPrefix(line, "❌ ERROR:") {
			select {
			case errCh <- line:
			default:
			}
		}
	}

	runErr := cmd.Wait()
	return buf.String(), runErr
}
