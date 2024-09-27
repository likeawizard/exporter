package main

import (
	"fmt"
	"github.com/dave/jennifer/jen"
	"os"
)

func createOutput(outputName, buildTag string) (*os.File, error) {
	output, err := os.Create(outputName)
	if err != nil {
		return nil, err
	}
	if buildTag != "" {
		_, err = output.Write([]byte(fmt.Sprintf("//go:build %s\n\n", buildTag)))
		if err != nil {
			output.Close()
			return nil, err
		}
	}

	return output, nil
}

func writeFile(output *os.File, pkgName string) error {
	f := jen.NewFile(pkgName)
	f.HeaderComment("Code generated by github.com/likeawizard/exporter. DO NOT EDIT.")

	genTypes(f)
	genConstants(f)
	genVariables(f)
	genMethods(f)

	return f.Render(output)
}

func genTypes(f *jen.File) {
	exportTypes = removeCollisions(exportTypes, public)
	if len(exportTypes) > 0 {
		f.Type().DefsFunc(func(g *jen.Group) {
			for _, t := range exportTypes {
				g.Id(exportCase(t, nil, replacements...)).Op("=").Id(t)
			}
		})
	}
}

func genConstants(f *jen.File) {
	exportConstants = removeCollisions(exportConstants, public)
	if len(exportConstants) > 0 {
		f.Const().DefsFunc(func(g *jen.Group) {
			for _, c := range exportConstants {
				g.Id(exportCase(c, nil)).Op("=").Id(c)
			}
		})
	}
}

func genVariables(f *jen.File) {
	exportVariables = removeCollisions(exportVariables, public)
	if len(exportVariables) > 0 {
		f.Var().DefsFunc(func(g *jen.Group) {
			for _, v := range exportVariables {
				g.Id(exportCase(v, nil)).Op("=").Id(v)
			}
		})
	}
}

func genMethods(f *jen.File) {
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
}
