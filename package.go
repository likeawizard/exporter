package main

import (
	"fmt"
	"golang.org/x/tools/go/packages"
)

func readPackage() (*packages.Package, error) {
	loadCfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypesSizes}
	pkgs, err := packages.Load(loadCfg, ".")
	if err != nil {
		return nil, err
	}

	return pkgs[0], nil
}

func collectImports(pkg *packages.Package) map[string]string {
	imports := make(map[string]string)
	for _, imp := range pkg.Imports {
		imports[imp.Name] = imp.PkgPath
	}

	return imports
}

func collectTypes(pkg *packages.Package, cfg Config) (map[string]struct{}, map[string]struct{}, error) {
	toExport := make(map[string]struct{})
	public := make(map[string]struct{})
	names := pkg.Types.Scope().Names()
	for _, n := range names {
		obj := pkg.Types.Scope().Lookup(n)
		if !obj.Exported() {
			toExport[obj.Name()] = struct{}{}
		} else {
			public[obj.Name()] = struct{}{}
		}
	}

	if _, ok := public[cfg.TargetOut]; ok {
		return nil, nil, fmt.Errorf("generated type %q name already exists in package", cfg.TargetOut)
	}

	if _, ok := toExport[cfg.TargetType]; !ok {
		return nil, nil, fmt.Errorf("target type not found in package")
	}

	for k := range toExport {
		if _, ok := public[exportCase(k, nil, cfg.TargetType, cfg.TargetOut)]; ok {
			fmt.Printf("name collision for %q. skipping export...\n", k)
			delete(toExport, k)
		}
	}

	return toExport, public, nil
}
