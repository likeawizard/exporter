package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	// TargetType is the name of package type to generate method exports.
	TargetType string
	// TargetOut is the name of the exported target type.
	TargetOut string
	// OutputName is the name of the output file.
	OutputName string
	// BuildTag is the build tag to add to the output file (Optional).
	BuildTag string
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := readFlags()
	os.Remove(cfg.OutputName)

	e := NewExport(cfg)

	err := e.ReadPackage()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read package")
	}
	log.Info().Str("output", cfg.OutputName).Str("package", e.pkg.Name).Str("targetType", cfg.TargetType).Msg("exporting target")

	e.CollectImports()
	err = e.CollectTypes()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to collect types")
	}

	e.ParsePkg()

	output, err := e.createOutput()
	defer output.Close()

	err = e.writeFile(output)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to write output")
	}
	log.Info().Str("output", cfg.OutputName).Msg("exported target to file")
}

func readFlags() Config {
	cfg := Config{}
	flag.StringVar(&cfg.TargetType, "name", "", "target type to export")
	flag.StringVar(&cfg.TargetOut, "outname", "", "name of exported target")
	flag.StringVar(&cfg.OutputName, "output", "", "output file name")
	flag.StringVar(&cfg.BuildTag, "tag", "", "build tag")
	flag.Parse()

	if cfg.TargetType == "" {
		log.Fatal().Msg("target type is required")
	}

	if cfg.OutputName == "" {
		cfg.OutputName = fmt.Sprintf("%s_export.go", cfg.TargetType)
	}

	if cfg.TargetOut == "" {
		cfg.TargetOut = exportCase(cfg.TargetType, nil)
	}

	return cfg
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
