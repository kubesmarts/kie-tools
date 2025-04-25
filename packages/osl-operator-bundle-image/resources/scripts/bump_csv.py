#!/usr/bin/env python3
import sys
from pathlib import Path
import ruamel.yaml

CSV_BASE_PATH = Path(
    "resources/config/manifests/bases/logic-operator-rhel8.clusterserviceversion.yaml"
)


def previous_version(version: str) -> str:
    major, minor, patch = map(int, version.split("."))

    if patch:
        patch -= 1
    elif minor:
        minor -= 1
    else:
        if major == 0:
            raise ValueError("Cannot compute a previous version for 0.0.x")
        major -= 1
        minor = patch = 0

    return f"{major}.{minor}.{patch}"


def update_versioning_csv(new_version: str) -> None:
    yaml = ruamel.yaml.YAML()
    yaml.preserve_quotes = True          # keep original quoting
    yaml.indent(mapping=2, sequence=4)

    with CSV_BASE_PATH.open("r") as f:
        data = yaml.load(f)

    data["metadata"]["name"] = f"logic-operator-rhel8.v{new_version}"
    data["spec"]["version"] = new_version
    data["spec"]["replaces"] = f"logic-operator-rhel8.v{previous_version(new_version)}"

    with CSV_BASE_PATH.open("w") as f:
        yaml.dump(data, f)

    print(
        f"{CSV_BASE_PATH} updated to {new_version} "
        f"(replaces {previous_version(new_version)})."
    )


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python bump_csv.py <new_version>")
        sys.exit(1)

    try:
        update_versioning_csv(sys.argv[1])
    except ValueError as e:
        print(f"Error: {e}")
        sys.exit(1)
