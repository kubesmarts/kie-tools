# .github/scripts/inject-executions.py
import xml.etree.ElementTree as ET
import sys

# This script injects Midstream specific executions into the frontend-maven-plugin

# --- Configuration ---
POM_PATH = 'packages/sonataflow-quarkus-devui/pom.xml'
EXECUTIONS_SNIPPET_PATH = '.github/supporting-files/ci/osl/pom-manipulation/fmp-executions.xml'
# ArtifactId of targetted plugin
TARGET_ARTIFACT_ID = 'frontend-maven-plugin'
# ---------------------

print(f"--- Modifying {POM_PATH} for CI build ---")

# Registering the namespace is crucial to properly parse pom.xml
ns = {'m': 'http://maven.apache.org/POM/4.0.0'}
ET.register_namespace('', ns['m'])
tree = ET.parse(POM_PATH)
root = tree.getroot()

# Find the target <executions> element in the pom.xml
# We "walk" the XML tree instead of using a single complex XPath query.
# This is more robust and avoids limitations in Python's ElementTree library.
target_executions_element = None
build_element = root.find('m:build', ns)
if build_element is not None:
    plugins_element = build_element.find('m:plugins', ns)
    if plugins_element is not None:
        # Iterate through all plugins to find the one with the correct artifactId
        for plugin in plugins_element.findall('m:plugin', ns):
            artifactId_element = plugin.find('m:artifactId', ns)
            if artifactId_element is not None and artifactId_element.text == TARGET_ARTIFACT_ID:
                # Found the plugin, now get its <executions> block
                target_executions_element = plugin.find('m:executions', ns)
                break # Stop searching once found

if target_executions_element is None:
    print(f"Error: Could not find the target plugin executions block with XPath: {TARGET_PLUGIN_XPATH}")
    sys.exit(1)

# The snippet file is not a valid XML file on its own, so we wrap it
# with a dummy root tag to parse it correctly.
with open(EXECUTIONS_SNIPPET_PATH, 'r') as f:
    snippet_content = f.read()
    
snippet_root = ET.fromstring(f"<dummy>{snippet_content}</dummy>")

# We use a counter to insert executions at the beginning of the list,
# which effectively prepends them while preserving their order from the snippet file.
insertion_index = 0
for execution_to_add in snippet_root:                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       
    print(f"Prepending execution with ID: {execution_to_add.find('id').text}")
    target_executions_element.insert(insertion_index, execution_to_add)
    insertion_index += 1                                                                                                

# Write the changes back to the pom.xml file
tree.write(POM_PATH, encoding='utf-8', xml_declaration=True)

print(f"Successfully injected CI executions into {POM_PATH}")                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   