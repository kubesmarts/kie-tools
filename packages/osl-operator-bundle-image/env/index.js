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

const { varsWithName, composeEnv, getOrDefault } = require("@kie-tools-scripts/build-env");
const rootEnv = require("@kie-tools/root-env/env");
const redHatEnv = require("@osl/redhat-env/env");
const sonataflowOperatorEnv = require("@kie-tools/sonataflow-operator/env");

module.exports = composeEnv([rootEnv, redHatEnv, sonataflowOperatorEnv], {
  vars: varsWithName({
    OSL_OPERATOR_BUNDLE_IMAGE__registry: {
      default: "registry.redhat.io",
      description: "The image registry.",
    },
    OSL_OPERATOR_BUNDLE_IMAGE__account: {
      default: "openshift-serverless-1",
      description: "The image registry account.",
    },
    OSL_OPERATOR_BUNDLE_IMAGE__name: {
      default: "logic-operator-bundle",
      description: "The image name.",
    },
    OSL_OPERATOR_BUNDLE_IMAGE__buildTag: {
      default: rootEnv.env.root.streamName,
      description: "The image tag.",
    },
    OSL_OPERATOR_BUNDLE__proxyImage: {
      default:
        "registry.redhat.io/openshift4/ose-kube-rbac-proxy@sha256:60c82c96d93e575eeddf576318425c9fdc2e0e86a234ae63734dd7ac8d59e350",
      description: "Internal proxy image for the Operator Manager Proxy Pod. Must come from a valid Red Hat registry.",
    },
    OSL_OPERATOR_BUNDLE__namespace: {
      default: "logic-operator-system",
      description: "Namespace name for the installation bundle",
    },
    OSL_OPERATOR_BUNDLE__namePrefix: {
      default: "logic-operator",
      description: "Name prefix for objects in the bundle",
    },
    OSL_OPERATOR_BUNDLE__version: {
      default: rootEnv.env.root.version,
      description:
        "Version of the bundle (CSV name and version). Can differ from the stream name when only the bundle is rebuilt.",
    },
    OSL_OPERATOR_BUNDLE__replacesVersion: {
      default: "",
      description:
        "Version of the bundle this release replaces (e.g. 1.37.0). Empty computes the previous version of the bundle version.",
    },
    OSL_OPERATOR_BUNDLE__olmSkipRange: {
      default: "",
      description: "OLM skipRange annotation of the bundle CSV (e.g. '>=1.38.2 <1.38.4'). Empty adds no annotation.",
    },
    OSL_OPERATOR_BUNDLE__channel: {
      default: "stable",
      description:
        "OLM channel of the bundle, set in the channels.v1 and channel.default.v1 annotations (e.g. stable-1.38).",
    },
  }),
  get env() {
    return {
      oslOperatorBundleImage: {
        registry: getOrDefault(this.vars.OSL_OPERATOR_BUNDLE_IMAGE__registry),
        account: getOrDefault(this.vars.OSL_OPERATOR_BUNDLE_IMAGE__account),
        name: getOrDefault(this.vars.OSL_OPERATOR_BUNDLE_IMAGE__name),
        buildTag: getOrDefault(this.vars.OSL_OPERATOR_BUNDLE_IMAGE__buildTag),
        version: getOrDefault(this.vars.OSL_OPERATOR_BUNDLE__version),
        proxyImage: getOrDefault(this.vars.OSL_OPERATOR_BUNDLE__proxyImage),
        namespace: getOrDefault(this.vars.OSL_OPERATOR_BUNDLE__namespace),
        namePrefix: getOrDefault(this.vars.OSL_OPERATOR_BUNDLE__namePrefix),
        replacesVersion: getOrDefault(this.vars.OSL_OPERATOR_BUNDLE__replacesVersion),
        olmSkipRange: getOrDefault(this.vars.OSL_OPERATOR_BUNDLE__olmSkipRange),
        channel: getOrDefault(this.vars.OSL_OPERATOR_BUNDLE__channel),
      },
    };
  },
});
