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

## @kie-tools-core/drools-and-kogito

Builds Drools, OptaPlanner, Kogito Runtimes, and Kogito Apps without installing any Maven artifacts to the local Maven repository. This package will skip the `install` phase and deploy the build result to `./dist/1st-party-m2`, so that there's no pollution of the local Maven repository.

A build is only triggered when the Kogito version (defined in `@kie-tools/root-env`, via `KOGITO_RUNTIME_version`) ends with `-local` (E.g., `999-20250511-local`). Otherwise, this package assumes that the version is coming from a remote Maven repository, or is already installed locally in the local Maven repository. For versions not ending on `-local`, you can still have this package trigger a build by using the `DROOLS_AND_KOGITO__forceBuild` env var.

This package has no build scripts, rather a single `install` script, meant to run when the `kie-tools` repository is bootstrapping via `pnpm bootstrap`.

---

Apache KIE (incubating) is an effort undergoing incubation at The Apache Software
Foundation (ASF), sponsored by the name of Apache Incubator. Incubation is
required of all newly accepted projects until a further review indicates that
the infrastructure, communications, and decision making process have stabilized
in a manner consistent with other successful ASF projects. While incubation
status is not necessarily a reflection of the completeness or stability of the
code, it does indicate that the project has yet to be fully endorsed by the ASF.

Some of the incubating project’s releases may not be fully compliant with ASF
policy. For example, releases may have incomplete or un-reviewed licensing
conditions. What follows is a list of known issues the project is currently
aware of (note that this list, by definition, is likely to be incomplete):

- Hibernate, an LGPL project, is being used. Hibernate is in the process of
  relicensing to ASL v2
- Some files, particularly test files, and those not supporting comments, may
  be missing the ASF Licensing Header

If you are planning to incorporate this work into your product/project, please
be aware that you will need to conduct a thorough licensing review to determine
the overall implications of including this work. For the current status of this
project through the Apache Incubator visit:
https://incubator.apache.org/projects/kie.html
