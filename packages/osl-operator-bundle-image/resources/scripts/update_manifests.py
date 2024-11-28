import sys
import ruamel.yaml

KUSTOMIZATION_DEFAULT_PATH = "generated/config/default/kustomization.yaml"
KUSTOMIZATION_MANIFESTS_PATH = "generated/config/manifests/kustomization.yaml"

def update_default_kustomization(new_namespace, new_name_prefix):
    yaml = ruamel.yaml.YAML()
    yaml.preserve_quotes = True

    with open(KUSTOMIZATION_DEFAULT_PATH, "r") as f:
        kustomization = yaml.load(f)

    kustomization["namespace"] = new_namespace
    kustomization["namePrefix"] = f"{new_name_prefix}-"

    with open(KUSTOMIZATION_DEFAULT_PATH, "w") as f:
        yaml.dump(kustomization, f)

    print(f"{KUSTOMIZATION_DEFAULT_PATH} updated successfully.")

def update_manifests_kustomization(new_csv_prefix):
    yaml = ruamel.yaml.YAML()
    with open(KUSTOMIZATION_MANIFESTS_PATH, "r") as f:
        kustomization = yaml.load(f)

    if 'resources' in kustomization:
        kustomization["resources"] = [
            replace_csv_prefix(resource, new_csv_prefix)
            for resource in kustomization["resources"]
        ]

    with open(KUSTOMIZATION_MANIFESTS_PATH, "w") as f:
        yaml.dump(kustomization, f)

    print(f"{KUSTOMIZATION_MANIFESTS_PATH} updated successfully.")

def replace_csv_prefix(resource, new_csv_prefix):
    if resource.endswith("clusterserviceversion.yaml"):
        return resource.replace("sonataflow-operator", new_csv_prefix)
    return resource

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python update_manifests.py <new_namespace> <new_name_prefix>")
        sys.exit(1)

    new_namespace = sys.argv[1]
    new_name_prefix = sys.argv[2]

    update_default_kustomization(new_namespace, new_name_prefix)
    update_manifests_kustomization(new_name_prefix)