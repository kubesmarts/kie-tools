# OSL Operator Bundle Image

## TODO:

- [x] - Make custom production CSV file as the basis for manifest generation
- [x] - Use `sonataflow-operator` make file to build prod manifests - we will need to access operator's API code.
- [ ] - Update images in the operator code base to use env
- [ ] - Update Makefile to not run `manifests` in the `bundle` recipe
- [ ] - Update Makefile to receive arguments for the package name
- [ ] - Update Makefile to receive arguments to the image descriptor in the `bundle-build` recipe
- [x] - Build Image using the metadata created via bundle
