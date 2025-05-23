#!/usr/bin/env bash
#
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCES_DIR=/tmp/artifacts
ADDED_DIR="${SCRIPT_DIR}"/added
LAUNCH_DIR="${KOGITO_HOME}"/launch

mkdir -p "${KOGITO_HOME}"/.m2/repository
mkdir -p "${KOGITO_HOME}/${PROJECT_ARTIFACT_ID}"

cp -v "${ADDED_DIR}"/* "${LAUNCH_DIR}"

# Unzip Quarkus app and Maven repository
tar xf "${SOURCES_DIR}"/kogito-swf-quarkus-app.tar -C "${KOGITO_HOME}/${PROJECT_ARTIFACT_ID}"
tar xf "${SOURCES_DIR}"/kogito-swf-maven-repo.tar -C "${KOGITO_HOME}"/.m2/repository

chown -R 1001:0 "${KOGITO_HOME}"
chmod -R ug+rwX "${KOGITO_HOME}"

# Cleanup Maven M2 Repo Via HTTP Settings XML
rm ${MAVEN_CONTAINER_BUILD_SETTINGS_PATH}
