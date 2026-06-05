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

const JSON_EXTENSIONS = ".json";
const YAML_EXTENSIONS = ".ya?ml";

const BPMN_EXTENSIONS = ".bpmn";
const DMN_EXTENSIONS = ".dmn";
const SCESIM_EXTENSIONS = ".scesim";
const PMML_EXTENSIONS = ".pmml";

const SERVERLESS_WORKFLOW_JSON_EXTENSION = `.sw(${JSON_EXTENSIONS})`;
const SERVERLESS_WORKFLOW_EXTENSIONS = `.sw(${JSON_EXTENSIONS}|${YAML_EXTENSIONS})`;
const YARD_EXTENSIONS = `.yard(${JSON_EXTENSIONS}|${YAML_EXTENSIONS})`;

/**
 * Gets supported kie editors file extensions that opens in single view mode.
 *
 * @returns regular expression with supported extensions.
 */
export function getSingleViewExtensionsRegExp(): RegExp {
  return new RegExp(`${BPMN_EXTENSIONS}|${DMN_EXTENSIONS}|${SCESIM_EXTENSIONS}|${PMML_EXTENSIONS}$`);
}

/**
 * Gets supported kie editors file extensions that opens in dual view mode.
 *
 * @returns regular expression with supported extensions.
 */
export function getDualViewExtensionsRegExp(): RegExp {
  return new RegExp(`${SERVERLESS_WORKFLOW_EXTENSIONS}|${YARD_EXTENSIONS}$`);
}

/**
 * Gets dashbuilder file extensions.
 *
 * @deprecated Dashbuilder support has been removed
 * @returns always returns a regex that never matches
 */
export function getDashbuilderExtensionsRegExp(): RegExp {
  return /(?!)/; // Never matches anything
}

/**
 * Checks if the file is kie editor with single view.
 *
 * @param fileName a name of the file to be checked.
 * @returns true or false.
 */
export function isKieEditorWithSingleView(fileName: string): boolean {
  return getSingleViewExtensionsRegExp().test(fileName);
}

/**
 * Checks if the file is kie editor with dual view.
 *
 * @param fileName a name of the file to be checked.
 * @returns true or false.
 */
export function isKieEditorWithDualView(fileName: string): boolean {
  return getDualViewExtensionsRegExp().test(fileName);
}

/**
 * Checks if the file is dashbuilder editor.
 *
 * @deprecated Dashbuilder support has been removed
 * @param fileName a name of the file to be checked.
 * @returns always returns false
 */
export function isDashbuilderEditor(fileName: string): boolean {
  return false;
}

/**
 * Checks if the file is serverless workflow JSON file.
 *
 * @param fileName a name of the file to be checked.
 * @returns true or false.
 */
export function isWorkflowJSONFile(fileName: string): boolean {
  return new RegExp(`${SERVERLESS_WORKFLOW_JSON_EXTENSION}$`).test(fileName);
}
