import os
from ruamel.yaml import YAML

# This script adds additional annotations from the resources folder to the target bundle annotations file that will be added to the final image.
# This is necessary because the operator-sdk CLI bundle operation would not override the file if exists with custom annotation (missing up to date annotations), or override if including only partial annotations.

# Constants for file paths
SOURCE_BUNDLE_ANNOTATIONS_PATH = 'resources/bundle/metadata/annotations.yaml'
DEST_BUNDLE_ANNOTATIONS_PATH = 'generated/bundle/metadata/annotations.yaml'

# Validate if the source and destination file paths exist
if not os.path.exists(SOURCE_BUNDLE_ANNOTATIONS_PATH):
    raise FileNotFoundError(f"Source file '{SOURCE_BUNDLE_ANNOTATIONS_PATH}' not found.")
if not os.path.exists(DEST_BUNDLE_ANNOTATIONS_PATH):
    raise FileNotFoundError(f"Destination file '{DEST_BUNDLE_ANNOTATIONS_PATH}' not found.")

# Initialize YAML processor
yaml = YAML()
yaml.preserve_quotes = True

# Function to load and merge annotations
def merge_annotations():
    # Load source annotations (preserving comments and formatting)
    with open(SOURCE_BUNDLE_ANNOTATIONS_PATH, 'r') as source_file:
        source_data = yaml.load(source_file)

    # Load destination annotations (preserving comments and formatting)
    with open(DEST_BUNDLE_ANNOTATIONS_PATH, 'r') as dest_file:
        dest_data = yaml.load(dest_file)

    # Ensure both files contain the 'annotations' key
    if 'annotations' not in source_data or 'annotations' not in dest_data:
        raise KeyError("Both source and destination files must contain the 'annotations' key.")

    # Get the source annotations and destination annotations
    source_annotations = source_data['annotations']
    dest_annotations = dest_data['annotations']

    # Append source annotations to destination annotations
    for key, value in source_annotations.items():
        # Add the source annotations to the end of destination annotations
        dest_annotations[key] = value

    last_key = list(dest_annotations.keys())[-1]
    dest_annotations.yaml_set_comment_before_after_key(last_key, before="Additional Red Hat annotations", indent=2)

    # Write back the modified data to the destination file, preserving formatting
    with open(DEST_BUNDLE_ANNOTATIONS_PATH, 'w') as dest_file:
        yaml.dump(dest_data, dest_file)

    print("Annotations successfully merged! 😄")

# Execute the function
merge_annotations()
