# Exporter - Code generation tool for exporting the private

### Why?
In software design it often helps to expose only part of your API and keep the internals hidden.
This helps to iterate your code - making a promise that the public interfaces can be relied on, while giving the freedom to make changes under the hood.
However, sometimes this can be a little limiting. Once in a while it can be tempting to peek under the hood and poker around a bit.
Exporter allows to export the internals of a package under a build tag. Exposing the internals for testing but without compromising good design.

#### Example
It's a common pattern in Domain Driven Design to have a public service interface that relies on a private repository.
The service upholds all business logic rules and the repository is just a dumb store for saving and retrieving data.
If you want to test the repository in a separate test package, or, furthermore, use a repository in a different domain package, you can export the repository under a build tag intended for testing.

For a practical example see [repository](/repository) package

### How?

```
//go:generate go tool exporter --name=repository --outname=Repo --output=repository_export.go --tag=tests
```

- `name` type to export methods for.
- `outname` alias for the exported type.
- `output` output file name.
- `tag` build tag to use when generating the file.

The exporter will also attempt to export all private types, variables, constants and functions from the entire package and not just the file where the directive is located.
Only methods indicated by `name` will be wrapped and exported.