package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sealdice/smallseal/tools/convert/converter"
)

func main() {
	inputPath := flag.String("input", "", "path to the v1 template file; omit to read from stdin")
	outputPath := flag.String("output", "", "path to write the converted template; omit to write to stdout")
	formatFlag := flag.String("format", "", "output format: yaml or json (default detects from output path)")
	templateVerFlag := flag.String("template-ver", "2.0", "templateVer to write; leave empty to reuse source value")
	flag.Parse()

	data, err := readInput(*inputPath)
	if err != nil {
		exitWithError(err)
	}

	if len(data) == 0 {
		exitWithError(errors.New("no input provided; specify -input or pipe data"))
	}

	inputFormat := detectInputFormat(*inputPath, data)

	tmplV1, err := converter.ParseTemplate(data, inputFormat)
	if err != nil {
		exitWithError(err)
	}

	converted := converter.Convert(tmplV1, *templateVerFlag)

	outputFormat := pickOutputFormat(*formatFlag, *outputPath)

	outputBytes, err := converter.MarshalOutput(converted, outputFormat)
	if err != nil {
		exitWithError(err)
	}

	if err := writeOutput(outputBytes, *outputPath); err != nil {
		exitWithError(err)
	}
}

func readInput(path string) ([]byte, error) {
	if path != "" {
		return os.ReadFile(path)
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func detectInputFormat(path string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	}

	for _, b := range data {
		if b == ' ' || b == '\n' || b == '\r' || b == '\t' {
			continue
		}
		if b == '{' || b == '[' {
			return "json"
		}
		break
	}
	return "yaml"
}

func pickOutputFormat(flagValue, outputPath string) string {
	switch strings.ToLower(flagValue) {
	case "json":
		return "json"
	case "yaml", "yml":
		return "yaml"
	case "":
		// fall through
	default:
		exitWithError(fmt.Errorf("unsupported output format: %s", flagValue))
	}

	ext := strings.ToLower(filepath.Ext(outputPath))
	if ext == ".json" {
		return "json"
	}
	return "yaml"
}

func writeOutput(data []byte, path string) error {
	if path == "" {
		if len(data) == 0 {
			return nil
		}
		if _, err := os.Stdout.Write(data); err != nil {
			return err
		}
		if data[len(data)-1] != '\n' {
			_, err := os.Stdout.Write([]byte("\n"))
			return err
		}
		return nil
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
	}

	return os.WriteFile(path, data, 0o644)
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
