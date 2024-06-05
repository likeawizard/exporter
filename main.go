package main

import (
	"flag"
	"fmt"
	"github.com/dave/jennifer/jen"
	"go/ast"
	"golang.org/x/tools/go/packages"
	"os"
	"strings"
)

type arg struct {
	Name string
	Type string
	Op   string
	Qual string
}

type methodWrapper struct {
	Receiver  arg
	Name      string
	Arguments []arg
	Return    []arg
}

func main() {
	var targetType, targetOut, buildTag, outputName string
	flag.StringVar(&targetType, "name", "", "target type to export")
	flag.StringVar(&targetOut, "outname", "", "name of exported target")
	flag.StringVar(&outputName, "output", "", "output file name")
	flag.StringVar(&buildTag, "tag", "", "build tag")
	flag.Parse()

	fileName := fmt.Sprintf("%s_export.go", targetType)
	if outputName == "" {
		outputName = fileName
	}

	outName := exportCase(targetType, nil)
	if targetOut != "" {
		outName = targetOut
	}
	replacements := []string{targetType, outName}
	os.Remove(outputName)

	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypesSizes}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		fmt.Println(err)
	}
	pkg := pkgs[0]

	public := make(map[string]struct{})
	toExport := make(map[string]struct{})
	imports := make(map[string]string)

	for _, imp := range pkg.Imports {
		imports[imp.Name] = imp.PkgPath
	}

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
		fmt.Printf("generated type %q name already exists in package\n", outName)
		os.Exit(1)
	}

	if _, ok := toExport[targetType]; !ok {
		fmt.Println("target type not found in package")
		os.Exit(1)
	}

	for k := range toExport {
		if _, ok := public[exportCase(k, nil, replacements...)]; ok {
			fmt.Printf("name collision for %q. skipping export...\n", k)
			delete(toExport, k)
		}
	}

	exportTypes := make([]string, 0)
	exportVariables := make([]string, 0)
	exportConstants := make([]string, 0)
	importsNeeded := make(map[string]struct{})
	wrappedMethods := make([]methodWrapper, 0)

	for _, file := range pkg.Syntax {
		if file == nil {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			//case *ast.GenDecl:
			//	_, ok := toExport[x.Tok.String()]
			//	if ok {
			//		fmt.Println("Exported GenDecl: ", x.Tok.String())
			//	} else {
			//		fmt.Println("Not Exported GenDecl: ", x.Tok.String())
			//	}
			case *ast.FuncDecl:
				_, ok := toExport[x.Name.Name]
				if ok && x.Recv == nil {
					exportVariables = append(exportVariables, x.Name.Name)
					break
				}

				if x.Recv != nil {
					m := methodWrapper{
						Name: x.Name.Name,
					}

					rec := x.Recv.List[0].Type
					switch t := rec.(type) {
					case *ast.StarExpr:
						m.Receiver = arg{
							Name: x.Recv.List[0].Names[0].Name,
							Type: t.X.(*ast.Ident).Name,
							Op:   "*",
						}
					case *ast.Ident:
						m.Receiver = arg{
							Name: x.Recv.List[0].Names[0].Name,
							Type: t.Name,
							Op:   "",
						}
					default:
						fmt.Println("Unknown receiver", x.Name.Name, t)

					}
					if m.Receiver.Type != targetType {
						return false
					}
					funcType := x.Type
					if funcType.Params != nil {
						for _, p := range funcType.Params.List {
							for _, n := range p.Names {
								op := ""
								typeExpr := ""
								qual := ""
								switch t := p.Type.(type) {
								case *ast.Ident:
									typeExpr = t.Name
								case *ast.ArrayType:
									op = "[]"
									switch tt := t.Elt.(type) {
									case *ast.SelectorExpr:
										name := tt.X.(*ast.Ident).Name
										importsNeeded[name] = struct{}{}
										typeExpr = tt.Sel.Name
										qual = imports[name]
									case *ast.Ident:
										typeExpr = tt.Name
									case *ast.StarExpr:
										typeExpr = tt.X.(*ast.Ident).Name
										op += "*"
									}
								case *ast.StarExpr:
									op = "*"
									switch tt := t.X.(type) {
									case *ast.Ident:
										typeExpr = tt.Name
									case *ast.SelectorExpr:
										name := tt.X.(*ast.Ident).Name
										importsNeeded[name] = struct{}{}
										typeExpr = tt.Sel.Name
										qual = imports[name]
									}
								case *ast.SelectorExpr:
									name := t.X.(*ast.Ident).Name
									importsNeeded[name] = struct{}{}
									typeExpr = t.Sel.Name
									qual = imports[name]
								default:
									fmt.Printf("Unknown type: %T %+v\n", t, t)
								}
								m.Arguments = append(m.Arguments, arg{
									Name: n.Name,
									Type: typeExpr,
									Op:   op,
									Qual: qual,
								})
							}
						}
					}

					if funcType.Results != nil {
						for _, r := range funcType.Results.List {
							op := ""
							typeExpr := ""
							qual := ""
							switch t := r.Type.(type) {
							case *ast.Ident:
								typeExpr = t.Name
							case *ast.ArrayType:
								op = "[]"
								switch tt := t.Elt.(type) {
								case *ast.SelectorExpr:
									name := tt.X.(*ast.Ident).Name
									importsNeeded[name] = struct{}{}
									typeExpr = tt.Sel.Name
									qual = imports[name]
								case *ast.Ident:
									typeExpr = tt.Name
								case *ast.StarExpr:
									typeExpr = tt.X.(*ast.Ident).Name
									op += "*"
								}
							case *ast.StarExpr:
								op = "*"
								switch tt := t.X.(type) {
								case *ast.Ident:
									typeExpr = tt.Name
								case *ast.SelectorExpr:
									name := tt.X.(*ast.Ident).Name
									importsNeeded[name] = struct{}{}
									typeExpr = tt.Sel.Name
									qual = imports[name]
								}
							case *ast.SelectorExpr:
								name := t.X.(*ast.Ident).Name
								importsNeeded[name] = struct{}{}
								typeExpr = t.Sel.Name
								qual = imports[name]
							default:
								fmt.Printf("Unknown type: %T\n", t)
							}
							namedReturn := ""
							if len(r.Names) > 0 {
								namedReturn = r.Names[0].Name
							}
							m.Return = append(m.Return, arg{
								Name: namedReturn,
								Type: typeExpr,
								Op:   op,
								Qual: qual,
							})
						}
					}
					wrappedMethods = append(wrappedMethods, m)
				}
			case *ast.TypeSpec:
				_, ok := toExport[x.Name.Name]
				if ok {
					exportTypes = append(exportTypes, x.Name.Name)
				}
			case *ast.ValueSpec:
				for _, n := range x.Names {
					_, ok := toExport[n.Name]
					if ok {
						switch n.Obj.Kind.String() {
						case "const":
							exportConstants = append(exportConstants, n.Name)
						case "var":
							exportVariables = append(exportVariables, n.Name)
						default:
							fmt.Println("Unknown ValueSpec Kind: ", n.Obj.Kind.String())
						}
					}
				}
			case nil:
			default:
			}
			return true
		})
	}

	output, err := os.Create(outputName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer output.Close()
	if buildTag != "" {
		output.Write([]byte(fmt.Sprintf("//go:build %s\n\n", buildTag)))
	}

	f := jen.NewFile(pkg.Name)
	f.HeaderComment("Code generated by github.com/likeawizard/exporter. DO NOT EDIT.")

	exportTypes = removeCollisions(exportTypes, public)
	if len(exportTypes) > 0 {
		f.Type().DefsFunc(func(g *jen.Group) {
			for _, t := range exportTypes {
				g.Id(exportCase(t, nil, replacements...)).Op("=").Id(t)
			}
		})
	}

	exportConstants = removeCollisions(exportConstants, public)
	if len(exportConstants) > 0 {
		f.Const().DefsFunc(func(g *jen.Group) {
			for _, c := range exportConstants {
				g.Id(exportCase(c, nil)).Op("=").Id(c)
			}
		})
	}

	exportVariables = removeCollisions(exportVariables, public)
	if len(exportVariables) > 0 {
		f.Var().DefsFunc(func(g *jen.Group) {
			for _, v := range exportVariables {
				g.Id(exportCase(v, nil)).Op("=").Id(v)
			}
		})
	}

	for _, m := range wrappedMethods {
		f.Func().
			Params(jen.Id(m.Receiver.Name).
				Op(m.Receiver.Op).
				Id(exportCase(m.Receiver.Type, nil, replacements...))).
			Id(exportCase(m.Name, nil)).ParamsFunc(func(g *jen.Group) {
			for _, a := range m.Arguments {
				typeToUse := a.Type
				if _, ok := toExport[a.Type]; ok {
					typeToUse = exportCase(a.Type, nil, replacements...)
				}

				g.Id(a.Name).Op(a.Op).Qual(a.Qual, typeToUse)
			}
		}).ParamsFunc(func(g *jen.Group) {
			for _, r := range m.Return {
				g.Id(r.Name).Op(r.Op).Qual(r.Qual, r.Type)
			}
		}).Block(
			jen.Return().Id(m.Receiver.Name).Dot(m.Name).CallFunc(func(g *jen.Group) {
				for _, a := range m.Arguments {
					g.Id(a.Name)
				}
			}),
		)
	}

	err = f.Render(output)
	if err != nil {
		fmt.Println(err)
	}
}

func exportCase(s string, collisions map[string]struct{}, replace ...string) string {
	if len(replace) == 2 && s == replace[0] {
		return replace[1]
	}
	exp := strings.ToUpper(s[:1]) + s[1:]

	if collisions == nil {
		return exp
	}

	if _, ok := collisions[exp]; ok {
		return exp + "Export"
	}

	return exp
}

func removeCollisions(exports []string, collisions map[string]struct{}) []string {
	cleaned := make([]string, 0)
	for _, e := range exports {
		if _, ok := collisions[e]; !ok {
			cleaned = append(cleaned, e)
		}
	}

	return cleaned
}
