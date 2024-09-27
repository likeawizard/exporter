package main

import (
	"fmt"
	"go/ast"
)

func walkFile(file *ast.File) {

	ast.Inspect(file, func(n ast.Node) bool {
		switch nodeType := n.(type) {
		case *ast.FuncDecl:
			if nodeType.Recv == nil {
				exportFunction(nodeType)
			} else {
				exportMethod(nodeType)
			}

		case *ast.TypeSpec:
			exportType(nodeType)
		case *ast.ValueSpec:
			exportValue(nodeType)
		case nil:
		default:
		}
		return true
	})
}

// exportMethod exports target type methods by wrapping the private method
func exportMethod(methodDecl *ast.FuncDecl) {
	m := getMethodReceiver(methodDecl)
	if m.Receiver.Type != targetType {
		return
	}
	funcType := methodDecl.Type
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

// exportFunction exports a function via a variable
func exportFunction(funcDecl *ast.FuncDecl) {
	if _, ok := toExport[funcDecl.Name.Name]; ok {
		exportVariables = append(exportVariables, funcDecl.Name.Name)
	}
}

// exportValue exports variables and constants
func exportValue(valSpec *ast.ValueSpec) {
	for _, n := range valSpec.Names {
		if _, ok := toExport[n.Name]; ok {
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
}

// exportType exports a type
func exportType(typeSpec *ast.TypeSpec) {
	if _, ok := toExport[typeSpec.Name.Name]; ok {
		exportTypes = append(exportTypes, typeSpec.Name.Name)
	}
}

func getMethodReceiver(methodDecl *ast.FuncDecl) methodWrapper {
	m := methodWrapper{
		Name: methodDecl.Name.Name,
	}

	rec := methodDecl.Recv.List[0].Type
	switch t := rec.(type) {
	case *ast.StarExpr:
		m.Receiver = arg{
			Name: methodDecl.Recv.List[0].Names[0].Name,
			Type: t.X.(*ast.Ident).Name,
			Op:   "*",
		}
	case *ast.Ident:
		m.Receiver = arg{
			Name: methodDecl.Recv.List[0].Names[0].Name,
			Type: t.Name,
			Op:   "",
		}
	default:
		fmt.Println("Unknown receiver", methodDecl.Name.Name, t)

	}

	return m
}
