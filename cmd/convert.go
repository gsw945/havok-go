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
	convertInput      string
	convertOutput     string
	convertPackage    string
	convertWasm       string
	convertSkipTypes  bool
	convertReceiver   string
	convertBindingPkg string
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Parse HavokPhysics.d.ts and generate Go binding scaffolding",
	Long: `convert scans a HavokPhysics.d.ts file using tree-sitter-typescript,
extracts every exported type alias, enum, and method from the
HavokPhysicsWithBindings interface, then emits two files:

  <output>/types_gen.go    – type declarations (only with --skip-types=false)
  <output>/bindings_gen.go – complete wazero bindings

The low-level calling convention (out-pointer placement, scalar widths) is read
from the shipped HavokPhysics.wasm, not guessed from the .d.ts. Methods that the
wasm does not export are reported and skipped.`,
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

		// Locate the wasm first: its export signatures drive the generated ABI.
		wasmSrc, err := locateWasm(convertInput, convertWasm)
		if err != nil {
			return fmt.Errorf("cannot generate bindings without HavokPhysics.wasm: %w", err)
		}
		fmt.Printf("Reading wasm signatures from %s …\n", wasmSrc)
		sigs, err := converter.ParseWasmSignatures(wasmSrc)
		if err != nil {
			return fmt.Errorf("wasm parse error: %w", err)
		}
		fmt.Printf("Wasm exports %d functions\n", len(sigs))

		opts := converter.DefaultOptions()
		opts.PackageName = convertPackage
		opts.OutputDir = convertOutput
		opts.SkipTypes = convertSkipTypes
		opts.ReceiverType = convertReceiver
		opts.BindingPkg = convertBindingPkg

		if err := converter.Generate(schema, sigs, opts); err != nil {
			return fmt.Errorf("generate error: %w", err)
		}

		typesPath := convertOutput + "/" + opts.TypesFile
		bindPath := convertOutput + "/" + opts.BindingFile
		if opts.SkipTypes {
			fmt.Printf("Generated:\n  %s\n", bindPath)
		} else {
			fmt.Printf("Generated:\n  %s\n  %s\n", typesPath, bindPath)
		}

		// Copy HavokPhysics.wasm into havok/wasm/ alongside the generated Go files.
		wasmDest := filepath.Join(filepath.Dir(convertOutput), "wasm", "HavokPhysics.wasm")
		if err := copyWasmFile(wasmSrc, wasmDest); err != nil {
			fmt.Printf("Note: %v\n", err)
			fmt.Printf("      Copy HavokPhysics.wasm manually to: %s\n", wasmDest)
		}
		return nil
	},
}

// locateWasm resolves the HavokPhysics.wasm source path.
// It first checks explicitWasm (from --wasm flag), then searches near inputDts.
func locateWasm(inputDts, explicitWasm string) (string, error) {
	if explicitWasm != "" {
		if _, err := os.Stat(explicitWasm); err == nil {
			return explicitWasm, nil
		}
		fmt.Printf("Warning: --wasm %s not found, falling back to auto-search\n", explicitWasm)
	}

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
			return c, nil
		}
	}
	return "", fmt.Errorf("HavokPhysics.wasm not found (tried --wasm flag and auto-search near %s)", inputDts)
}

// copyWasmFile copies src to dest, creating parent directories as needed.
func copyWasmFile(src, dest string) error {
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
		"Directory for generated files (default: ./havok/generated)")
	convertCmd.Flags().StringVarP(&convertPackage, "package", "p", "generated",
		"Go package name for generated files")
	convertCmd.Flags().StringVarP(&convertWasm, "wasm", "w", "",
		"Path to HavokPhysics.wasm to copy into havok/wasm/ (auto-searched if not set)")
	convertCmd.Flags().BoolVar(&convertSkipTypes, "skip-types", false,
		"Skip generating types_gen.go (use when types are already defined in the package)")
	convertCmd.Flags().StringVar(&convertReceiver, "receiver", "HP",
		"Go type name used as the method receiver in generated bindings")
	convertCmd.Flags().StringVar(&convertBindingPkg, "binding-pkg",
		"github.com/gsw945/havok-go/havok/binding",
		"Import path of the hand-written binding package (types + infrastructure)")
}
