package main

import (
	"flag"
	"fmt"
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

var (
	targetType, targetOut, buildTag, outputName, outName string
	replacements                                         []string

	exportTypes     = make([]string, 0)
	exportVariables = make([]string, 0)
	exportConstants = make([]string, 0)
	importsNeeded   = make(map[string]struct{})
	wrappedMethods  = make([]methodWrapper, 0)

	public   = make(map[string]struct{})
	toExport = make(map[string]struct{})
	imports  = make(map[string]string)
)

func main() {
	readFlags()
	os.Remove(outputName)

	pkg, err := readPackage()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	collectImports(pkg)
	err = collectTypes(pkg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, file := range pkg.Syntax {
		if file == nil {
			continue
		}
		walkFile(file)
	}

	output, err := createOutput(outputName, buildTag)
	defer output.Close()

	err = writeFile(output, pkg.Name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func readFlags() {
	flag.StringVar(&targetType, "name", "", "target type to export")
	flag.StringVar(&targetOut, "outname", "", "name of exported target")
	flag.StringVar(&outputName, "output", "", "output file name")
	flag.StringVar(&buildTag, "tag", "", "build tag")
	flag.Parse()

	fileName := fmt.Sprintf("%s_export.go", targetType)
	if outputName == "" {
		outputName = fileName
	}

	outName = exportCase(targetType, nil)
	if targetOut != "" {
		outName = targetOut
	}
	replacements = []string{targetType, outName}
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
