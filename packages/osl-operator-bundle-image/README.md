# OSL Operator Bundle Image

This bundle package inherits the ENV variables from `sonataflow-operator` package. Meaning that you must export that vars in order to build the bundle image since the registry, tag, and version are the same from the operator with the `-bundle` prefix in the name:

```shell
# REQUIRED ENV:
export KIE_TOOLS_BUILD__buildContainerImages=true

# OPTIONAL ENVs:
# Image name/tag information. This information is optional, this package has these set by default.
export OSL_OPERATOR_BUNDLE_IMAGE__registry=registry.access.redhat.com
export OSL_OPERATOR_BUNDLE_IMAGE__account=openshift-serverless-1
export OSL_OPERATOR_BUNDLE_IMAGE__name=logic-operator-bundle
# This version is also optional since once branched, we can update the `root-env` package env `KIE_TOOLS_BUILD__streamName` to the OSL version.
export OSL_OPERATOR_BUNDLE_IMAGE__buildTag=1.35

# Platform images
export OSL..



# Quarkus/Kogito version. This information will be set in the image labels and internal builds in `root-env`.
# Optionally you can also use Cekit overrides when building the final image in the internal systems.
export QUARKUS_PLATFORM_version=3.8.6
export KOGITO_RUNTIME_version=9.101-redhat

pnpm build:dev # build:prod also works, does the same currently.
```
