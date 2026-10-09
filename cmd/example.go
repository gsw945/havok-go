//go:build !gen_only

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/gsw945/havok-go/havok"
	havokwasm "github.com/gsw945/havok-go/havok/wasm"
	"github.com/spf13/cobra"
)

var exampleWasm string

var exampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Load HavokPhysics.wasm and run a physics demo",
	Long: `example instantiates the Havok physics engine (via wazero) and runs a
simulation whose setup mirrors the Babylon.js reference scene:

  1. HP_GetStatistics        – verify the WASM is alive
  2. HP_World_Create         – create a physics world
  3. HP_World_SetGravity     – gravity (0, -9.81, 0)
  4. HP_World_SetIdealStepTime – fixed 1/60 s solver step (as the front-end does)
  5. static ground box (100 x 1 x 100) + dynamic 1 m cube, mass 1,
     restitution 0.2, friction 0.2, linear damping 5, angular damping 50
  6. step the world and print the trajectory so it can be compared with the
     browser output
  7. clean up (Release)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return runExample(ctx, exampleWasm)
	},
}

func init() {
	rootCmd.AddCommand(exampleCmd)

	exampleCmd.Flags().StringVarP(&exampleWasm, "wasm", "w",
		"",
		"Path to HavokPhysics.wasm (uses embedded wasm when omitted)")
}

func runExample(ctx context.Context, wasmFlag string) error {
	start := time.Now()
	var hp *havok.HavokPhysics
	var err error

	if wasmFlag != "" {
		fmt.Printf("Loading WASM (flag): %s\n", wasmFlag)
		hp, err = havok.New(ctx, wasmFlag)
	} else if havokwasm.IsReal() {
		fmt.Println("Loading WASM (embedded)")
		hp, err = havok.NewFromBytes(ctx, havokwasm.WasmBytes)
	} else {
		return fmt.Errorf("no HavokPhysics.wasm available; run 'havok-go convert' or pass --wasm <path>")
	}
	if err != nil {
		return fmt.Errorf("havok init: %w", err)
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
		fmt.Printf("  NumBodies=%d  NumShapes=%d  NumConstraints=%d  NumWorlds=%d\n",
			stats.NumBodies, stats.NumShapes, stats.NumConstraints, stats.NumWorlds)
	}
	fmt.Println()

	// -------------------------------------------------------------------------
	// Step 2: Create world, configure gravity and solver step
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
		if r, err := hp.HP_World_Release(ctx, worldId); err != nil {
			fmt.Printf("HP_World_Release error: %v\n", err)
		} else {
			fmt.Printf("[HP_World_Release] result=%s\n", r.Error())
		}
	}()

	if gRes, err := hp.HP_World_SetGravity(ctx, worldId, havok.Vector3{0, -9.81, 0}); err != nil {
		return fmt.Errorf("HP_World_SetGravity: %w", err)
	} else {
		fmt.Printf("[HP_World_SetGravity] result=%s  g=(0, -9.81, 0)\n", gRes.Error())
	}
	// Fixed solver step keeps the effective stiffness identical to a browser
	// running Babylon's scene at a locked 60 Hz physics rate.
	const dt = 1.0 / 60.0
	if r, err := hp.HP_World_SetIdealStepTime(ctx, worldId, dt); err != nil {
		return fmt.Errorf("HP_World_SetIdealStepTime: %w", err)
	} else {
		fmt.Printf("[HP_World_SetIdealStepTime] result=%s  dt=%.5f\n", r.Error(), dt)
	}
	fmt.Println()

	// -------------------------------------------------------------------------
	// Step 3: Static ground box (top face at y = 0)
	// -------------------------------------------------------------------------
	groundRes, groundShape, err := hp.HP_Shape_CreateBox(ctx,
		havok.Vector3{0, -0.5, 0},
		havok.Quaternion{0, 0, 0, 1},
		havok.Vector3{100, 1, 100})
	if err != nil {
		return fmt.Errorf("HP_Shape_CreateBox(ground): %w", err)
	}
	if !groundRes.IsOK() {
		return fmt.Errorf("ground shape failed: %s", groundRes.Error())
	}
	defer hp.HP_Shape_Release(ctx, groundShape) //nolint:errcheck
	if r, err := hp.HP_Shape_SetMaterial(ctx, groundShape, havok.PhysicsMaterial{
		StaticFriction: 0.6, DynamicFriction: 0.6, Restitution: 0.05,
	}); err != nil {
		return fmt.Errorf("HP_Shape_SetMaterial(ground): %w", err)
	} else if !r.IsOK() {
		return fmt.Errorf("ground material failed: %s", r.Error())
	}
	groundRes2, groundBody, err := hp.HP_Body_Create(ctx)
	if err != nil {
		return fmt.Errorf("HP_Body_Create(ground): %w", err)
	}
	if !groundRes2.IsOK() {
		return fmt.Errorf("ground body failed: %s", groundRes2.Error())
	}
	defer hp.HP_Body_Release(ctx, groundBody) //nolint:errcheck
	if r, err := hp.HP_Body_SetShape(ctx, groundBody, groundShape); err != nil {
		return err
	} else if !r.IsOK() {
		return fmt.Errorf("ground SetShape failed: %s", r.Error())
	}
	if r, err := hp.HP_Body_SetMotionType(ctx, groundBody, havok.MotionType_STATIC); err != nil {
		return err
	} else if !r.IsOK() {
		return fmt.Errorf("ground SetMotionType failed: %s", r.Error())
	}
	if r, err := hp.HP_World_AddBody(ctx, worldId, groundBody, false); err != nil {
		return err
	} else if !r.IsOK() {
		return fmt.Errorf("ground AddBody failed: %s", r.Error())
	}
	fmt.Printf("[ground] static box 100x1x100 at (0,-0.5,0) added\n")

	// -------------------------------------------------------------------------
	// Step 4: Dynamic 1 m cube — same parameters as the Babylon.js player
	// -------------------------------------------------------------------------
	const (
		cubeSize    = 1.0
		cubeMass    = 1.0
		startHeight = 2.0
	)
	cubeRes, cubeShape, err := hp.HP_Shape_CreateBox(ctx,
		havok.Vector3{0, 0, 0},
		havok.Quaternion{0, 0, 0, 1},
		havok.Vector3{cubeSize, cubeSize, cubeSize})
	if err != nil {
		return fmt.Errorf("HP_Shape_CreateBox(cube): %w", err)
	}
	if !cubeRes.IsOK() {
		return fmt.Errorf("cube shape failed: %s", cubeRes.Error())
	}
	defer hp.HP_Shape_Release(ctx, cubeShape) //nolint:errcheck
	if r, err := hp.HP_Shape_SetMaterial(ctx, cubeShape, havok.PhysicsMaterial{
		StaticFriction: 0.2, DynamicFriction: 0.2, Restitution: 0.2,
	}); err != nil {
		return fmt.Errorf("HP_Shape_SetMaterial(cube): %w", err)
	} else if !r.IsOK() {
		return fmt.Errorf("cube material failed: %s", r.Error())
	}

	bodyRes, bodyId, err := hp.HP_Body_Create(ctx)
	if err != nil {
		return fmt.Errorf("HP_Body_Create: %w", err)
	}
	if !bodyRes.IsOK() {
		return fmt.Errorf("HP_Body_Create failed: %s", bodyRes.Error())
	}
	defer hp.HP_Body_Release(ctx, bodyId) //nolint:errcheck
	if r, err := hp.HP_Body_SetShape(ctx, bodyId, cubeShape); err != nil {
		return err
	} else if !r.IsOK() {
		return fmt.Errorf("cube SetShape failed: %s", r.Error())
	}
	if r, err := hp.HP_Body_SetMotionType(ctx, bodyId, havok.MotionType_DYNAMIC); err != nil {
		return err
	} else if !r.IsOK() {
		return fmt.Errorf("cube SetMotionType failed: %s", r.Error())
	}

	// Explicit mass properties (a solid 1 m cube: I = m*(a^2+b^2)/12 = 1/6).
	inertia := cubeMass * (cubeSize*cubeSize + cubeSize*cubeSize) / 12.0
	if r, err := hp.HP_Body_SetMassProperties(ctx, bodyId, havok.MassProperties{
		CenterOfMass:       havok.Vector3{0, 0, 0},
		Mass:               cubeMass,
		InertiaTensorDiag:  havok.Vector3{inertia, inertia, inertia},
		InertiaOrientation: havok.Quaternion{0, 0, 0, 1},
	}); err != nil {
		return err
	} else if !r.IsOK() {
		return fmt.Errorf("cube SetMassProperties failed: %s", r.Error())
	}

	// Damping — identical to the front-end player aggregate.
	if r, err := hp.HP_Body_SetLinearDamping(ctx, bodyId, 5.0); err != nil {
		return err
	} else if !r.IsOK() {
		return fmt.Errorf("SetLinearDamping failed: %s", r.Error())
	}
	if r, err := hp.HP_Body_SetAngularDamping(ctx, bodyId, 50.0); err != nil {
		return err
	} else if !r.IsOK() {
		return fmt.Errorf("SetAngularDamping failed: %s", r.Error())
	}

	// Initial transform: 5 m above the ground, axis-aligned.
	if r, err := hp.HP_Body_SetQTransform(ctx, bodyId, havok.QTransform{
		Translation: havok.Vector3{0, startHeight, 0},
		Rotation:    havok.Quaternion{0, 0, 0, 1},
	}); err != nil {
		return err
	} else if !r.IsOK() {
		return fmt.Errorf("cube SetQTransform failed: %s", r.Error())
	}

	if r, err := hp.HP_World_AddBody(ctx, worldId, bodyId, false); err != nil {
		return err
	} else if !r.IsOK() {
		return fmt.Errorf("cube AddBody failed: %s", r.Error())
	}
	fmt.Printf("[cube] dynamic box %.0fx%.0fx%.0f, mass=%.0f, at (0,%.0f,0)\n\n",
		cubeSize, cubeSize, cubeSize, cubeMass, startHeight)

	// -------------------------------------------------------------------------
	// Step 5: Free fall, landing and rest
	// -------------------------------------------------------------------------
	fmt.Printf("Simulating 180 steps (dt=%.5f s) — free fall, landing, rest:\n", dt)
	fmt.Printf("  %4s  %8s   %-26s %-26s\n", "step", "time", "position (x,y,z)", "linear velocity")
	printState := func(step int, label string) error {
		r1, pos, err := hp.HP_Body_GetPosition(ctx, bodyId)
		if err != nil {
			return err
		}
		r2, vel, err := hp.HP_Body_GetLinearVelocity(ctx, bodyId)
		if err != nil {
			return err
		}
		if !r1.IsOK() || !r2.IsOK() {
			return fmt.Errorf("GetPosition/GetLinearVelocity returned %s / %s", r1.Error(), r2.Error())
		}
		fmt.Printf("  %4d  %8.3f   (%7.3f, %7.3f, %7.3f)   (%7.3f, %7.3f, %7.3f) %s\n",
			step, float64(step)*dt, pos[0], pos[1], pos[2], vel[0], vel[1], vel[2], label)
		return nil
	}
	if err := printState(0, ""); err != nil {
		return err
	}
	for i := 1; i <= 180; i++ {
		if r, err := hp.HP_World_Step(ctx, worldId, dt); err != nil {
			return fmt.Errorf("HP_World_Step[%d]: %w", i, err)
		} else if !r.IsOK() {
			return fmt.Errorf("HP_World_Step[%d] result=%s", i, r.Error())
		}
		if i%15 == 0 {
			if err := printState(i, ""); err != nil {
				return err
			}
		}
	}

	_, restPos, err := hp.HP_Body_GetPosition(ctx, bodyId)
	if err != nil {
		return err
	}
	expected := cubeSize / 2.0
	fmt.Printf("\nResting height y=%.4f (expected ~%.2f, |error|=%.4f)\n",
		restPos[1], expected, abs64(restPos[1]-expected))

	// -------------------------------------------------------------------------
	// Step 6: Horizontal impulse — same "applyImpulse" call the browser uses
	// -------------------------------------------------------------------------
	fmt.Printf("\nApplying impulse (8, 0, 0) at the centre of mass…\n")
	if r, err := hp.HP_Body_ApplyImpulse(ctx, bodyId, havok.Vector3{0, 0, 0}, havok.Vector3{8, 0, 0}); err != nil {
		return err
	} else if !r.IsOK() {
		return fmt.Errorf("ApplyImpulse failed: %s", r.Error())
	}
	if _, vel, err := hp.HP_Body_GetLinearVelocity(ctx, bodyId); err == nil {
		fmt.Printf("  velocity immediately after impulse: (%.3f, %.3f, %.3f)  [expect x ~8]\n",
			vel[0], vel[1], vel[2])
	}
	for i := 0; i < 60; i++ {
		if r, err := hp.HP_World_Step(ctx, worldId, dt); err != nil {
			return err
		} else if !r.IsOK() {
			return fmt.Errorf("HP_World_Step failed: %s", r.Error())
		}
	}
	if err := printState(240, "after 60 more steps"); err != nil {
		return err
	}

	// -------------------------------------------------------------------------
	// Final statistics
	// -------------------------------------------------------------------------
	if res, stats, err := hp.HP_GetStatistics(ctx); err == nil && res.IsOK() {
		fmt.Printf("\n[HP_GetStatistics after sim] NumBodies=%d  NumShapes=%d  NumWorlds=%d\n",
			stats.NumBodies, stats.NumShapes, stats.NumWorlds)
	}

	fmt.Println("\nDemo complete.")
	return nil
}

func abs64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
