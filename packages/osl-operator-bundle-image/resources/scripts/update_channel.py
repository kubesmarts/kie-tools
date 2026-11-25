#!/usr/bin/env python3
import sys
from pathlib import Path
import ruamel.yaml

BUNDLE_ANNOTATIONS_PATH = Path("resources/bundle/metadata/annotations.yaml")
CHANNEL_KEYS = (
    "operators.operatorframework.io.bundle.channels.v1",
    "operators.operatorframework.io.bundle.channel.default.v1",
)


def update_channel(channel: str, bundle_image_path: Path) -> None:
    yaml = ruamel.yaml.YAML()
    yaml.preserve_quotes = True  # keep original quoting
    yaml.indent(mapping=2, sequence=4, offset=2)

    with BUNDLE_ANNOTATIONS_PATH.open("r") as f:
        annotations = yaml.load(f)
    for key in CHANNEL_KEYS:
        annotations["annotations"][key] = channel
    with BUNDLE_ANNOTATIONS_PATH.open("w") as f:
        yaml.dump(annotations, f)

    with bundle_image_path.open("r") as f:
        image = yaml.load(f)
    for label in image["labels"]:
        if label["name"] in CHANNEL_KEYS:
            label["value"] = channel
    with bundle_image_path.open("w") as f:
        yaml.dump(image, f)

    print(f"Bundle channel set to {channel} in {BUNDLE_ANNOTATIONS_PATH} and {bundle_image_path}.")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python update_channel.py <channel> <bundle_image_yaml>")
        sys.exit(1)

    update_channel(sys.argv[1], Path(sys.argv[2]))
