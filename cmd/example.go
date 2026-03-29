package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gsw945/havok-go/havok"
	"github.com/spf13/cobra"
)

var exampleWasm string

var exampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Load HavokPhysics.wasm and run a physics demo",
	Long: `example instantiates the Havok physics engine (via wazero) and runs
a short simulation:

  1. HP_GetStatistics  – verify WASM is alive
  2. HP_World_Create   – create a physics world
  3. HP_Body_Create    – create two bodies
  4. HP_Shape_CreateSphere – create a sphere shape
  5. HP_Body_SetShape / SetMotionType / World_AddBody
  6. HP_World_SetGravity + Step loop (5 steps)
  7. Clean up (Release)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return runExample(ctx, exampleWasm)
	},
}

func init() {
	rootCmd.AddCommand(exampleCmd)

	exampleCmd.Flags().StringVarP(&exampleWasm, "wasm", "w",
		"",
		"Path to HavokPhysics.wasm (tries several default locations if not set)")
}

// defaultWasmPaths contains common relative locations for HavokPhysics.wasm.
var defaultWasmPaths = []string{
	"HavokPhysics.wasm",
	"../BabylonJS-havok/packages/havok/lib/esm/HavokPhysics.wasm",
	"../BabylonJS-havok/packages/havok/lib/umd/HavokPhysics.wasm",
}

func resolveWasm(flagPath string) (string, error) {
	if flagPath != "" {
		return flagPath, nil
	}
	for _, p := range defaultWasmPaths {
		if fileExists(p) {
			return p, nil
		}
	}
	return "", fmt.Errorf("HavokPhysics.wasm not found; pass --wasm <path>")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func runExample(ctx context.Context, wasmFlag string) error {
	wasmPath, err := resolveWasm(wasmFlag)
	if err != nil {
		return err
	}
	fmt.Printf("Loading WASM: %s\n", wasmPath)

	start := time.Now()
	hp, err := havok.New(ctx, wasmPath)
	if err != nil {
		return fmt.Errorf("havok.New: %w", err)
	}
	defer hp.Close()
	fmt.Printf("WASM loaded in %v\n\n", time.Since(start))

	// -------------------------------------------------------------------------
	// Step 1: Statistics (sanity check)
	// -------------------------------------------------------------------------
	res, stats, err := hp.HP_GetStatistics(ctx)
	if err != nil {
		return fmt.Errorf("HP_GetStatistics: %w", err)
	}
	fmt.Printf("[HP_GetStatistics] result=%s\n", res.Error())
	if res.IsOK() {
		fmt.Printf("  NumBodies=%d  NumShapes=%d  NumConstraints=%d\n",
			stats.NumBodies, stats.NumShapes, stats.NumConstraints)
		fmt.Printf("  NumWorlds=%d  NumQueryCollectors=%d\n",
			stats.NumWorlds, stats.NumQueryCollectors)
	}
	fmt.Println()

	// -------------------------------------------------------------------------
	// Step 2: Create world
	// -------------------------------------------------------------------------
	wRes, worldId, err := hp.HP_World_Create(ctx)
	if err != nil {
		return fmt.Errorf("HP_World_Create: %w", err)
	}
	fmt.Printf("[HP_World_Create] result=%s  worldId=%d\n", wRes.Error(), worldId[0])
	if !wRes.IsOK() {
		return fmt.Errorf("HP_World_Create failed: %s", wRes.Error())
	}
	defer func() {
		if res, err := hp.HP_World_Release(ctx, worldId); err != nil {
			fmt.Printf("HP_World_Release error: %v\n", err)
		} else {
			fmt.Printf("[HP_World_Release] result=%s\n", res.Error())
		}
	}()

	// Set gravity to (0, -9.81, 0)
	if gRes, err := hp.HP_World_SetGravity(ctx, worldId, havok.Vector3{0, -9.81, 0}); err != nil {
		return fmt.Errorf("HP_World_SetGravity: %w", err)
	} else {
		fmt.Printf("[HP_World_SetGravity] result=%s\n", gRes.Error())
	}

	// -------------------------------------------------------------------------
	// Step 3: Create a dynamic body with a sphere shape
	// -------------------------------------------------------------------------
	bRes, bodyId, err := hp.HP_Body_Create(ctx)
	if err != nil {
		return fmt.Errorf("HP_Body_Create: %w", err)
	}
	fmt.Printf("[HP_Body_Create] result=%s  bodyId=%d\n", bRes.Error(), bodyId[0])
	if !bRes.IsOK() {
		return fmt.Errorf("HP_Body_Create failed: %s", bRes.Error())
	}
	defer func() {
		if res, err := hp.HP_Body_Release(ctx, bodyId); err != nil {
			fmt.Printf("HP_Body_Release error: %v\n", err)
		} else {
			fmt.Printf("[HP_Body_Release] result=%s\n", res.Error())
		}
	}()

	// Create a sphere shape at origin, radius 0.5
	sRes, shapeId, err := hp.HP_Shape_CreateSphere(ctx, havok.Vector3{0, 0, 0}, 0.5)
	if err != nil {
		return fmt.Errorf("HP_Shape_CreateSphere: %w", err)
	}
	fmt.Printf("[HP_Shape_CreateSphere] result=%s  shapeId=%d\n", sRes.Error(), shapeId[0])
	if !sRes.IsOK() {
		return fmt.Errorf("HP_Shape_CreateSphere failed: %s", sRes.Error())
	}
	defer func() {
		if res, err := hp.HP_Shape_Release(ctx, shapeId); err != nil {
			fmt.Printf("HP_Shape_Release error: %v\n", err)
		} else {
			fmt.Printf("[HP_Shape_Release] result=%s\n", res.Error())
		}
	}()

	// Attach shape to body
	if r, err := hp.HP_Body_SetShape(ctx, bodyId, shapeId); err != nil {
		return fmt.Errorf("HP_Body_SetShape: %w", err)
	} else {
		fmt.Printf("[HP_Body_SetShape] result=%s\n", r.Error())
	}

	// Make it dynamic
	if r, err := hp.HP_Body_SetMotionType(ctx, bodyId, havok.MotionType_DYNAMIC); err != nil {
		return fmt.Errorf("HP_Body_SetMotionType: %w", err)
	} else {
		fmt.Printf("[HP_Body_SetMotionType] result=%s\n", r.Error())
	}

	// Add to world
	if r, err := hp.HP_World_AddBody(ctx, worldId, bodyId, true); err != nil {
		return fmt.Errorf("HP_World_AddBody: %w", err)
	} else {
		fmt.Printf("[HP_World_AddBody] result=%s\n", r.Error())
	}
	fmt.Println()

	// -------------------------------------------------------------------------
	// Step 4: Simulate 5 steps
	// -------------------------------------------------------------------------
	const dt = 1.0 / 60.0
	fmt.Printf("Simulating 5 steps (dt=%.4f s)…\n", dt)
	for i := range 5 {
		if r, err := hp.HP_World_Step(ctx, worldId, dt); err != nil {
			return fmt.Errorf("HP_World_Step[%d]: %w", i, err)
		} else if !r.IsOK() {
			fmt.Printf("  step %d: result=%s\n", i, r.Error())
		} else {
			fmt.Printf("  step %d: OK\n", i)
		}
	}
	fmt.Println()

	// Stats after simulation
	if res, stats, err := hp.HP_GetStatistics(ctx); err == nil && res.IsOK() {
		fmt.Printf("[HP_GetStatistics after sim] NumBodies=%d  NumShapes=%d  NumWorlds=%d\n",
			stats.NumBodies, stats.NumShapes, stats.NumWorlds)
	}

	fmt.Println("\nDemo complete.")
	return nil
}
