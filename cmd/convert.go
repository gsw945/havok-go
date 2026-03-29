package cmd

import (
	"fmt"
	"os"

	"github.com/gsw945/havok-go/converter"
	"github.com/spf13/cobra"
)

var (
	convertInput   string
	convertOutput  string
	convertPackage string
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
		return nil
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringVarP(&convertInput, "input", "i", "",
		"Path to HavokPhysics.d.ts (required)")
	convertCmd.Flags().StringVarP(&convertOutput, "output", "o", "./generated",
		"Directory for generated files")
	convertCmd.Flags().StringVarP(&convertPackage, "package", "p", "generated",
		"Go package name for generated files")
}
