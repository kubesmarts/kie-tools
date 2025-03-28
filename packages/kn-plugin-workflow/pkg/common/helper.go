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

package common

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"html/template"
	"io"
	"os"
	"os/exec"
)

func RunCommand(command *exec.Cmd, commandName string) error {
	var stdoutBuf, stderrBuf bytes.Buffer

	command.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	command.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	if err := command.Start(); err != nil {
		fmt.Printf("ERROR: starting command \"%s\" failed\n", commandName)
		return err
	}

	if err := command.Wait(); err != nil {
		fmt.Printf("ERROR: something went wrong during command \"%s\"\n", commandName)
		return err
	}

	return nil
}

func RunExtensionCommand(extensionCommand string, extensions string) error {
	command := ExecCommand("mvn", extensionCommand, fmt.Sprintf("-Dextensions=%s", extensions))
	if err := RunCommand(command, extensionCommand); err != nil {
		fmt.Println("ERROR: It wasn't possible to add Quarkus extension in your pom.xml.")
		return err
	}
	return nil
}

func GetTemplate(cmd *cobra.Command, name string) *template.Template {
	var (
		body = cmd.Long + "\n\n" + cmd.UsageString()
		t    = template.New(name)
		tpl  = template.Must(t.Parse(body))
	)
	return tpl
}

func DefaultTemplatedHelp(cmd *cobra.Command, args []string) {
	tpl := GetTemplate(cmd, "help")
	var data = struct{ Name string }{Name: cmd.Root().Use}

	if err := tpl.Execute(cmd.OutOrStdout(), data); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "unable to display help text: %v", err)
	}
}
