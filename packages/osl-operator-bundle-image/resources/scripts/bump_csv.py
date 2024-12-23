import sys
import ruamel.yaml

CSV_BASE_PATH = "resources/config/manifests/bases/logic-operator-rhel8.clusterserviceversion.yaml"

def update_versioning_csv(new_version):
    yaml = ruamel.yaml.YAML()
    with open(CSV_BASE_PATH, "r") as f:
        versioning = yaml.load(f)

    # Update the metadata.name with the new version suffix
    versioning["metadata"]["name"] = f"logic-operator-rhel8.v{new_version}"

    # Update the spec.version with the new version
    versioning["spec"]["version"] = new_version

    with open(CSV_BASE_PATH, "w") as f:
        yaml.dump(versioning, f)

    print(f"{CSV_BASE_PATH} updated successfully.")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python bump_csv.py <new_version>")
        sys.exit(1)

    new_version = sys.argv[1]

    update_versioning_csv(new_version)
