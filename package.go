package main

import (
	"fmt"
	"golang.org/x/tools/go/packages"
)

func readPackage() (*packages.Package, error) {
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypesSizes}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, err
	}

	return pkgs[0], nil
}

func collectImports(pkg *packages.Package) {
	for _, imp := range pkg.Imports {
		imports[imp.Name] = imp.PkgPath
	}
}

func collectTypes(pkg *packages.Package) error {
	names := pkg.Types.Scope().Names()
	for _, n := range names {
		obj := pkg.Types.Scope().Lookup(n)
		if !obj.Exported() {
			toExport[obj.Name()] = struct{}{}
		} else {
			public[obj.Name()] = struct{}{}
		}
	}

	if _, ok := public[outName]; ok {
		return fmt.Errorf("generated type %q name already exists in package", outName)
	}

	if _, ok := toExport[targetType]; !ok {
		return fmt.Errorf("target type not found in package")
	}

	for k := range toExport {
		if _, ok := public[exportCase(k, nil, replacements...)]; ok {
			fmt.Printf("name collision for %q. skipping export...\n", k)
			delete(toExport, k)
		}
	}

	return nil
}
