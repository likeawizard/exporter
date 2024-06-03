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
	var targetType, buildTag, outputName string
	flag.StringVar(&targetType, "name", "", "target type to export")
	flag.StringVar(&outputName, "output", "", "output file name")
	flag.StringVar(&buildTag, "tag", "", "build tag")
	flag.Parse()

	fileName := fmt.Sprintf("%s_export.go", targetType)
	if outputName == "" {
		outputName = fileName
	}
	os.Remove(fileName)
	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer output.Close()
	if buildTag != "" {
		output.Write([]byte(fmt.Sprintf("//go:build %s\n\n", buildTag)))
	}

	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypesSizes}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		fmt.Println(err)
	}
	pkg := pkgs[0]

	toExport := make(map[string]struct{})
	imports := make(map[string]string)

	for _, imp := range pkg.Imports {
		imports[imp.Name] = imp.PkgPath
	}

	fmt.Println("Imports: ", imports)

	names := pkg.Types.Scope().Names()
	for _, n := range names {
		obj := pkg.Types.Scope().Lookup(n)
		if !obj.Exported() {
			toExport[obj.Name()] = struct{}{}

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
				if ok {
					if x.Recv == nil {
						exportVariables = append(exportVariables, x.Name.Name)
					}
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
									typeExpr = t.Elt.(*ast.Ident).Name
									op = "[]"
								case *ast.StarExpr:
									typeExpr = t.X.(*ast.Ident).Name
									op = "*"
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
								typeExpr = t.Elt.(*ast.Ident).Name
								op = "[]"
							case *ast.StarExpr:
								typeExpr = t.X.(*ast.Ident).Name
								op = "*"
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

	f := jen.NewFile("repository")
	f.HeaderComment("Code generated by github.com/likeawizard/exporter. DO NOT EDIT.")

	f.Type().DefsFunc(func(g *jen.Group) {
		for _, t := range exportTypes {
			g.Id(exportCase(t)).Op("=").Id(t)
		}
	})

	f.Const().DefsFunc(func(g *jen.Group) {
		for _, c := range exportConstants {
			g.Id(exportCase(c)).Op("=").Id(c)
		}
	})

	f.Var().DefsFunc(func(g *jen.Group) {
		for _, v := range exportVariables {
			g.Id(exportCase(v)).Op("=").Id(v)
		}
	})

	for _, m := range wrappedMethods {
		f.Func().Params(jen.Id(m.Receiver.Name).Op(m.Receiver.Op).Id(exportCase(m.Receiver.Type))).Id(exportCase(m.Name)).ParamsFunc(func(g *jen.Group) {
			for _, a := range m.Arguments {
				typeToUse := a.Type
				if _, ok := toExport[a.Type]; ok {
					typeToUse = exportCase(a.Type)
				}

				g.Id(a.Name).Qual(a.Qual, typeToUse)
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

func exportCase(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}
