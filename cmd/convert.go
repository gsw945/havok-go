package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gsw945/havok-go/converter"
	"github.com/spf13/cobra"
)

var (
	convertInput   string
	convertOutput  string
	convertPackage string
	convertWasm    string
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Parse HavokPhysics.d.ts and generate Go binding scaffolding",
	Long: `convert scans a HavokPhysics.d.ts file using tree-sitter-typescript,
extracts every exported type alias, enum, and method from the
HavokPhysicsWithBindings interface, then emits two files:

  <output>/types_gen.go    – type declarations
  <output>/bindings_gen.go – function stubs with TODO bodies

Each stub follows the emscripten sret calling convention so the
generated code is a drop-in skeleton for a complete wazero binding.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if convertInput == "" {
			return fmt.Errorf("--input is required")
		}

		if _, err := os.Stat(convertInput); err != nil {
			return fmt.Errorf("input file not found: %w", err)
		}

		fmt.Printf("Parsing %s …\n", convertInput)
		schema, err := converter.Parse(convertInput)
		if err != nil {
			return fmt.Errorf("parse error: %w", err)
		}

		fmt.Printf("Found %d enums, %d type aliases, %d methods\n",
			len(schema.Enums), len(schema.Types), len(schema.Methods))

		opts := converter.DefaultOptions()
		opts.PackageName = convertPackage
		opts.OutputDir = convertOutput

		if err := converter.Generate(schema, opts); err != nil {
			return fmt.Errorf("generate error: %w", err)
		}

		typesPath := convertOutput + "/" + opts.TypesFile
		bindPath := convertOutput + "/" + opts.BindingFile
		fmt.Printf("Generated:\n  %s\n  %s\n", typesPath, bindPath)

		// Try to copy HavokPhysics.wasm into havok/wasm/ alongside the generated Go files.
		wasmDest := filepath.Join(filepath.Dir(convertOutput), "wasm", "HavokPhysics.wasm")
		if err := copyWasm(convertInput, convertWasm, wasmDest); err != nil {
			fmt.Printf("Note: %v\n", err)
			fmt.Printf("      Copy HavokPhysics.wasm manually to: %s\n", wasmDest)
		}
		return nil
	},
}

// copyWasm copies HavokPhysics.wasm to dest.
// It first checks explicitWasm (from --wasm flag), then searches near inputDts.
// Returns an error (informational) if wasm cannot be found by either method.
func copyWasm(inputDts, explicitWasm, dest string) error {
	var src string

	// 1. Use explicitly provided path if given and the file exists.
	if explicitWasm != "" {
		if _, err := os.Stat(explicitWasm); err == nil {
			src = explicitWasm
		} else {
			fmt.Printf("Warning: --wasm %s not found, falling back to auto-search\n", explicitWasm)
		}
	}

	// 2. Auto-search relative to the input .d.ts file.
	if src == "" {
		inputDir := filepath.Dir(inputDts)
		candidates := []string{
			filepath.Join(inputDir, "HavokPhysics.wasm"),
			filepath.Join(inputDir, "lib", "esm", "HavokPhysics.wasm"),
			filepath.Join(inputDir, "lib", "umd", "HavokPhysics.wasm"),
			filepath.Join(inputDir, "..", "lib", "esm", "HavokPhysics.wasm"),
			filepath.Join(inputDir, "..", "lib", "umd", "HavokPhysics.wasm"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				src = c
				break
			}
		}
	}

	if src == "" {
		return fmt.Errorf("HavokPhysics.wasm not found (tried --wasm flag and auto-search near %s)", inputDts)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("cannot create wasm directory: %w", err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", dest, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	fmt.Printf("Copied wasm: %s\n  → %s\n", src, dest)
	return nil
}

func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringVarP(&convertInput, "input", "i", "",
		"Path to HavokPhysics.d.ts (required)")
	convertCmd.Flags().StringVarP(&convertOutput, "output", "o", "./havok/generated",
		"Directory for generated files")
	convertCmd.Flags().StringVarP(&convertPackage, "package", "p", "generated",
		"Go package name for generated files")
	convertCmd.Flags().StringVarP(&convertWasm, "wasm", "w", "",
		"Path to HavokPhysics.wasm to copy into havok/wasm/ (auto-searched if not set)")
}
