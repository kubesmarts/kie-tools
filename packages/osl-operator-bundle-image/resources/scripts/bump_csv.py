#!/usr/bin/env python3
import sys
from pathlib import Path
import ruamel.yaml

CSV_BASE_PATH = Path(
    "resources/config/manifests/bases/logic-operator.clusterserviceversion.yaml"
)


def previous_version(version: str) -> str:
    major, minor, patch = map(int, version.split("."))

    if patch:
        patch -= 1
    elif minor:
        minor -= 1
    else:
        if major == 0:
            print(f"Won't change 'replaces' in CSV since version is incompatible {version}")
            return version
        major -= 1
        minor = patch = 0

    return f"{major}.{minor}.{patch}"


def update_versioning_csv(new_version: str, replaces_version: str = "", olm_skip_range: str = "") -> None:
    yaml = ruamel.yaml.YAML()
    yaml.preserve_quotes = True          # keep original quoting
    yaml.indent(mapping=2, sequence=4)

    with CSV_BASE_PATH.open("r") as f:
        data = yaml.load(f)

    replaces = replaces_version or previous_version(new_version)

    data["metadata"]["name"] = f"logic-operator.v{new_version}"
    data["spec"]["version"] = new_version
    data["spec"]["replaces"] = f"logic-operator.v{replaces}"

    annotations = data["metadata"]["annotations"]
    if olm_skip_range:
        annotations["olm.skipRange"] = olm_skip_range
    else:
        annotations.pop("olm.skipRange", None)

    with CSV_BASE_PATH.open("w") as f:
        yaml.dump(data, f)

    print(
        f"{CSV_BASE_PATH} updated to {new_version} "
        f"(replaces {replaces})."
    )


if __name__ == "__main__":
    if len(sys.argv) not in (2, 3, 4):
        print("Usage: python bump_csv.py <new_version> [<replaces_version> [<olm_skip_range>]]")
        sys.exit(1)

    try:
        update_versioning_csv(*sys.argv[1:])
    except ValueError as e:
        print(f"Error: {e}")
        sys.exit(1)
