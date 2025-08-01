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

const path = require("path");
const { runKogitoImageInstall } = require("@kie/kogito-images-install-helper");

const { env } = require("./env");

const { buildTag, registry, account, name: imageName } = env.kogitoJitRunnerImage;

runKogitoImageInstall({
  finalImageName: "kogito-jit-runner",
  imageTag: { buildTag, registry, account, name: imageName },
  resourceDir: path.resolve(__dirname, "./resources"),
  imagePkgDir: __dirname,
});

/// Maven app

const { setupMavenConfigFile, buildTailFromPackageJsonDependencies } = require("@kie-tools/maven-base");
setupMavenConfigFile(`
    -Drevision=${env.kogitoJitRunnerImage.version}
    -Dmaven.repo.local.tail=${buildTailFromPackageJsonDependencies()}
`);
