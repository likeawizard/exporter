package main

import (
	"fmt"
	"github.com/dave/jennifer/jen"
	"github.com/rs/zerolog/log"
	"go/ast"
	"golang.org/x/tools/go/packages"
	"os"
)

type (
	Export struct {
		cfg Config
		pkg *packages.Package
		jen *jen.File

		replacements []string

		imports         map[string]string
		exportTypes     []string
		exportVariables []string
		exportConstants []string
		importsNeeded   map[string]struct{}

		wrappedMethods []methodWrapper
		public         map[string]struct{}
		toExport       map[string]struct{}
	}

	arg struct {
		Name string
		Type string
		Op   string
		Qual string
	}

	methodWrapper struct {
		Receiver  arg
		Name      string
		Arguments []arg
		Return    []arg
	}
)

func NewExport(cfg Config) *Export {
	return &Export{
		cfg:           cfg,
		replacements:  []string{cfg.TargetType, cfg.TargetOut},
		imports:       make(map[string]string),
		importsNeeded: make(map[string]struct{}),
		public:        make(map[string]struct{}),
		toExport:      make(map[string]struct{}),
	}
}

func (e *Export) ReadPackage() error {
	pkg, err := readPackage()
	if err != nil {
		return err
	}

	e.pkg = pkg
	return nil
}

func (e *Export) CollectImports() {
	e.imports = collectImports(e.pkg)
}

func (e *Export) CollectTypes() error {
	toExport, public, err := collectTypes(e.pkg, e.cfg)
	if err != nil {
		return err
	}

	e.toExport = toExport
	e.public = public

	return nil
}

func (e *Export) ParsePkg() {
	for _, file := range e.pkg.Syntax {
		if file == nil {
			continue
		}
		e.walkFile(file, e.cfg.TargetType)
	}
}

func (e *Export) walkFile(file *ast.File, targetType string) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch nodeType := n.(type) {
		case nil:
			return false
		case *ast.FuncDecl:
			if nodeType.Recv == nil {
				e.exportFunction(nodeType)
			} else {
				e.exportMethod(nodeType, targetType)
			}
		case *ast.TypeSpec:
			e.exportType(nodeType)
		case *ast.ValueSpec:
			e.exportValue(nodeType)
		}
		return true
	})
}

// exportMethod exports target type methods by wrapping the private method
func (e *Export) exportMethod(methodDecl *ast.FuncDecl, targetType string) {
	m := e.getMethodReceiver(methodDecl)
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
						e.importsNeeded[name] = struct{}{}
						typeExpr = tt.Sel.Name
						qual = e.imports[name]
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
						e.importsNeeded[name] = struct{}{}
						typeExpr = tt.Sel.Name
						qual = e.imports[name]
					}
				case *ast.SelectorExpr:
					name := t.X.(*ast.Ident).Name
					e.importsNeeded[name] = struct{}{}
					typeExpr = t.Sel.Name
					qual = e.imports[name]
				default:
					log.Warn().Str("type", fmt.Sprintf("%T", t)).Msg("unknown type")
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
					e.importsNeeded[name] = struct{}{}
					typeExpr = tt.Sel.Name
					qual = e.imports[name]
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
					e.importsNeeded[name] = struct{}{}
					typeExpr = tt.Sel.Name
					qual = e.imports[name]
				}
			case *ast.SelectorExpr:
				name := t.X.(*ast.Ident).Name
				e.importsNeeded[name] = struct{}{}
				typeExpr = t.Sel.Name
				qual = e.imports[name]
			default:
				log.Warn().Str("type", fmt.Sprintf("%T", t)).Msg("unknown type")
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
	e.wrappedMethods = append(e.wrappedMethods, m)
}

// exportFunction exports a function via a variable
func (e *Export) exportFunction(funcDecl *ast.FuncDecl) {
	if _, ok := e.toExport[funcDecl.Name.Name]; ok {
		e.exportVariables = append(e.exportVariables, funcDecl.Name.Name)
	}
}

// exportValue exports variables and constants
func (e *Export) exportValue(valSpec *ast.ValueSpec) {
	for _, n := range valSpec.Names {
		if _, ok := e.toExport[n.Name]; ok {
			switch n.Obj.Kind.String() {
			case "const":
				e.exportConstants = append(e.exportConstants, n.Name)
			case "var":
				e.exportVariables = append(e.exportVariables, n.Name)
			default:
				log.Warn().Str("kind", n.Obj.Kind.String()).Msg("unknown kind")
			}
		}

	}
}

// exportType exports a type
func (e *Export) exportType(typeSpec *ast.TypeSpec) {
	if _, ok := e.toExport[typeSpec.Name.Name]; ok {
		e.exportTypes = append(e.exportTypes, typeSpec.Name.Name)
	}
}

func (e *Export) getMethodReceiver(methodDecl *ast.FuncDecl) methodWrapper {
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
		log.Warn().Str("type", fmt.Sprintf("%T", t)).Str("name", methodDecl.Name.Name).Msg("unknown receiver")
	}

	return m
}

func (e *Export) createOutput() (*os.File, error) {
	output, err := os.Create(e.cfg.OutputName)
	if err != nil {
		return nil, err
	}
	if e.cfg.BuildTag != "" {
		_, err = output.Write([]byte(fmt.Sprintf("//go:build %s\n\n", e.cfg.BuildTag)))
		if err != nil {
			output.Close()
			return nil, err
		}
	}

	return output, nil
}

func (e *Export) writeFile(output *os.File) error {
	e.jen = jen.NewFile(e.pkg.Name)
	e.jen.HeaderComment("Code generated by github.com/likeawizard/exporter. DO NOT EDIT.")

	e.genTypes()
	e.genConstants()
	e.genVariables()
	e.genMethods()

	return e.jen.Render(output)
}

func (e *Export) genTypes() {
	e.exportTypes = removeCollisions(e.exportTypes, e.public)
	if len(e.exportTypes) > 0 {
		e.jen.Type().DefsFunc(func(g *jen.Group) {
			for _, t := range e.exportTypes {
				g.Id(exportCase(t, nil, e.cfg.TargetType, e.cfg.TargetOut)).Op("=").Id(t)
			}
		})
	}
}

func (e *Export) genConstants() {
	e.exportConstants = removeCollisions(e.exportConstants, e.public)
	if len(e.exportConstants) > 0 {
		e.jen.Const().DefsFunc(func(g *jen.Group) {
			for _, c := range e.exportConstants {
				g.Id(exportCase(c, nil)).Op("=").Id(c)
			}
		})
	}
}

func (e *Export) genVariables() {
	e.exportVariables = removeCollisions(e.exportVariables, e.public)
	if len(e.exportVariables) > 0 {
		e.jen.Var().DefsFunc(func(g *jen.Group) {
			for _, v := range e.exportVariables {
				g.Id(exportCase(v, nil)).Op("=").Id(v)
			}
		})
	}
}

func (e *Export) genMethods() {
	for _, m := range e.wrappedMethods {
		e.jen.Func().
			Params(jen.Id(m.Receiver.Name).
				Op(m.Receiver.Op).
				Id(exportCase(m.Receiver.Type, nil, e.cfg.TargetType, e.cfg.TargetOut))).
			Id(exportCase(m.Name, nil)).ParamsFunc(func(g *jen.Group) {
			for _, a := range m.Arguments {
				typeToUse := a.Type
				if _, ok := e.toExport[a.Type]; ok {
					typeToUse = exportCase(a.Type, nil, e.cfg.TargetType, e.cfg.TargetOut)
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
}
