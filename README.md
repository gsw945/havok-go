# havok-go
[![Go Reference](https://pkg.go.dev/badge/github.com/gsw945/havok-go/havok.svg)](https://pkg.go.dev/github.com/gsw945/havok-go/havok) [![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/license/MIT)


Go 版 [Havok 物理引擎](https://github.com/BabylonJS/havok) 绑定，通过 [wazero](https://github.com/tetratelabs/wazero)（纯 Go WebAssembly 运行时）包装 `HavokPhysics.wasm`，无 CGo 依赖（运行时）。

> 基准版本：`@babylonjs/havok` **1.3.14**（`HavokPhysics.wasm` sha256 `02691776…`）。

---

## 关键修复（本次更新）

旧版本与 Babylon.js 前端“算得不一样”的根因有两个，均已修复并加了回归测试：

### 1. 调用约定错误 —— `[Result, data]` 返回值

emscripten embind 生成的真实 ABI 是：

| 项目 | 旧实现（错误） | 实际 ABI（已修正） |
|------|----------------|--------------------|
| 返回值 | 假设 wasm 返回 `void`，`Result` 放在输出缓冲区开头 | **wasm 返回 i32 = `Result`** |
| 输出指针位置 | 放在参数**最前** | 放在参数**最后**（每个返回元素一个指针） |
| 数据偏移 | 从缓冲区 `+4` 开始读 | 从缓冲区 **`+0`** 开始读（没有 Result 前缀） |

后果：`HP_Body_GetPosition`、`HP_World_GetStepLimit`、`HP_GetStatistics` 等**所有读取类接口**都读到全 0 或垃圾数据，且把句柄当成指针传进去，函数返回 `INVALIDHANDLE` 却被忽略（因为代码从缓冲区里读那个“Result”，而缓冲区是零初始化的，于是永远显示 OK）。

### 2. 枚举值与 wasm 不一致 —— 这是“物理算不对”的直接原因

`HavokPhysics.d.ts` 里的枚举是**声明顺序**（0,1,2,…），但 wasm 通过 embind 实际注册的值完全不同：

| 枚举 | d.ts 写法（错误） | wasm 实际值（正确） |
|------|------------------|--------------------|
| `Result` | `OK=0, FAIL=1, INVALIDHANDLE=2, INVALIDARGS=3, NOTIMPLEMENTED=4` | `OK=0, FAIL=0x80004005, INVALIDHANDLE=0x80070006, INVALIDARGS=0x80070057, NOTIMPLEMENTED=0x80004001`（HRESULT） |
| `ShapeType` | `COLLIDER=0, CONTAINER=1` | `COLLIDER=1, CONTAINER=2` |
| `MotionType` | `STATIC=0, KINEMATIC=1, DYNAMIC=2` | **`STATIC=1, KINEMATIC=2, DYNAMIC=4`** |
| `EventType` | `0,1,2,3,4` | `COLLISION_STARTED=1, CONTINUED=2, FINISHED=4`（位标志） |

后果：调用 `HP_Body_SetMotionType(body, 2)` 本想设为 DYNAMIC，实际设成了 **KINEMATIC** —— 运动学刚体只按设定速度积分，**完全不受重力、不受碰撞响应**。于是“物体不掉落、碰撞没反应”，与浏览器表现完全不同；而 `SetMotionType(body, 0)` 会直接返回 `INVALIDARGS`。

> 枚举值由 wasm 启动时的 `_embind_register_enum_value` 注册调用**权威决定**。`havok/verify_test.go` 里的 `TestEnumsMatchWasm` 会实例化 wasm 并把注册的枚举全部抓出来与 Go 常量逐一比对，防止今后再次漂移。

### 3. 补齐了原先缺失/占位的能力

- 所有结构体的字节级 (反)序列化（`MassProperties`、`PhysicsMaterial`、`FilterInfo`、`Aabb`、`ObjectStatistics`、`ContactPoint`、各类查询输入/输出……），原先 `MassProperties` / `PhysicsMaterial` 是 `// TODO` 空实现。
- 多返回值（`[Result, Vector3, Vector3]` 等）不再生成 TODO 桩，而是每个元素一个输出指针。
- `HP_Shape_PathIterator_GetNext` 名称与 wasm 导出不一致的问题（wasm 为 `HP_ShapePathIterator_GetNext`），生成器现在按规范化名称容错匹配。
- 生成器改为**从 `HavokPhysics.wasm` 读取导出签名**来驱动 ABI，不再猜测；`HavokPhysics.d.ts` 中 wasm 未导出的 5 个方法会被跳过并打印：

```
Bindings: 125 methods matched against wasm exports
Skipped 5 method(s) not exported by this HavokPhysics.wasm:
  - HP_Event_AsCollision
  - HP_Event_AsTrigger
  - HP_World_CastRay
  - HP_World_GetCollisionEvents
  - HP_World_GetTriggerEvents
```

---

## 目录结构

```
havok-go/
├── main.go              # CLI 入口
├── build.cmd            # Windows 一键编译（含 -linkmode internal）
├── convert.cmd          # Windows 一键运行 convert（生成绑定代码）
├── example.cmd          # Windows 一键运行 example（物理仿真演示）
├── cmd/                 # Cobra 子命令（convert / example）
├── converter/           # TypeScript d.ts 解析 & Go 代码生成器
│   ├── model.go         # 数据模型
│   ├── parser.go        # tree-sitter 解析器 + 类型辅助方法
│   ├── wasm.go          # wasm 导出签名解析（调用约定的唯一事实来源）
│   └── generator.go     # 代码生成模板
└── havok/               # Havok WASM 绑定（独立子模块）
    ├── havok.go         # HavokPhysics 初始化（嵌入 *generated.HP）
    ├── types.go         # 从 binding 包 re-export 的类型别名
    ├── imports.go       # emscripten env 桩函数
    ├── verify_test.go   # 行为验证 + 枚举漂移守卫（go test）
    ├── binding/         # 手写基础设施（不可删除）
    │   ├── binding.go   # Binding struct + Alloc/FreePtr/Mem/CallResultI32
    │   ├── helpers.go   # WASM 内存读写工具（ReadU64、WriteVector3 等）
    │   ├── codec.go     # 所有结构体的字节级编解码 + 尺寸常量
    │   └── types.go     # 所有类型定义（HP_*Id、枚举、Vector3 等）
    ├── generated/       # convert 命令自动生成（可删除后重新生成）
    │   └── bindings_gen.go  # HP struct + 全部 125 个 HP_* 方法
    └── wasm/
        ├── wasm.go      # //go:embed HavokPhysics.wasm
        └── HavokPhysics.wasm  # 真实 wasm（由 convert 命令写入）
```

---

## 编译

> 依赖 Go 1.23+。`convert` 命令使用 tree-sitter 解析 TypeScript，**需要 CGo**（C 编译器）；
> `example` 命令与 `havok` 包本身是纯 Go，不需要 CGo。

```bash
# 下载依赖
go mod download

# 编译为可执行文件（Windows + TDM-GCC 必须加 -ldflags "-linkmode internal"）
go build -ldflags "-linkmode internal" -o havok-go.exe .

# 或直接运行（无需单独编译）
go run . <子命令>
```

> **Windows CGo 注意事项**
>
> 默认的外部链接器（`ld`）会向 PE 可执行文件注入 `.CRT`、`.tls` 等节区，
> 与 Go 1.21+ 运行时的 SEH（结构化异常处理）机制冲突，导致编译出的 `.exe` 无法运行。
> **修复**：添加 `-ldflags "-linkmode internal"` 强制使用 Go 内置链接器（`build.cmd` 已封装）。
>
> 若 C 编译器不在 PATH 上，需要显式指定，例如 MSYS2 MinGW-w64：
>
> ```bat
> set "CC=D:\msys64\mingw64\bin\gcc.exe"
> set "PATH=D:\msys64\mingw64\bin;%PATH%"
> ```

---

## 更新 Havok wasm / 重新生成绑定

当需要升级 Havok 版本时：

### 1. 准备 BabylonJS havok 源

```bash
git clone https://github.com/BabylonJS/havok  BabylonJS-havok
```

### 2. 运行 convert 命令

**Windows 一键运行（推荐）：**

```bat
REM 自动探测 d.ts 位置（../BabylonJS-havok/ 或 ../refs/BabylonJS-havok/）
convert.cmd

REM 手动指定
convert.cmd --input path\to\HavokPhysics.d.ts
convert.cmd --wasm  path\to\HavokPhysics.wasm
```

**跨平台（go run）：**

```bash
# 注意：convert 使用 go build tag gen_only，不需要编译 havok 包本身
go run -tags gen_only . convert --skip-types \
  --input ../refs/BabylonJS-havok/packages/havok/HavokPhysics.d.ts
```

命令完成后会输出：

```
Parsing D:/tmp/refs/BabylonJS-havok/packages/havok/HavokPhysics.d.ts …
Found 10 enums, 31 type aliases, 130 methods
Reading wasm signatures from D:\tmp\refs\...\lib\esm\HavokPhysics.wasm …
Wasm exports 141 functions
Bindings: 125 methods matched against wasm exports
Skipped 5 method(s) not exported by this HavokPhysics.wasm:
  - HP_Event_AsCollision
  - HP_Event_AsTrigger
  - HP_World_CastRay
  - HP_World_GetCollisionEvents
  - HP_World_GetTriggerEvents
Generated:
  ./havok/generated/bindings_gen.go
Copied wasm: …\lib\esm\HavokPhysics.wasm
  → havok\wasm\HavokPhysics.wasm
```

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--input` / `-i` | （必填）| `HavokPhysics.d.ts` 路径 |
| `--output` / `-o` | `./havok/generated` | 生成文件的输出目录 |
| `--package` / `-p` | `generated` | 生成文件的 Go 包名 |
| `--skip-types` | `false` | 跳过生成 `types_gen.go`（类型已在 `binding/` 中定义时使用，**推荐**）|
| `--receiver` | `HP` | 生成方法的接收器类型名 |
| `--binding-pkg` | `github.com/gsw945/havok-go/havok/binding` | 手写 binding 包的 import 路径 |
| `--wasm` / `-w` | （自动搜索）| 指定 wasm 文件路径（同时用于读取 ABI 签名）|

> **重要**：升级 Havok 后**务必**重新跑一次 `go test ./...`（见下方“验证”）。
> 如果新版本调整了任何枚举值，`TestEnumsMatchWasm` 会立刻失败并指出差异。

---

## 运行示例

`example` 命令复现了前端 Babylon.js 场景的关键参数：重力 `(0,-9.81,0)`、固定 1/60 s 步长、
静态地面 + 动态 1 m 立方体（质量 1、restitution 0.2、friction 0.2、线性阻尼 5、角阻尼 50），
并打印轨迹供与浏览器输出对比。

**Windows 一键运行：**

```bat
example.cmd
example.cmd --wasm path\to\HavokPhysics.wasm
```

**跨平台：**

```bash
go run . example
```

正常输出（节选）：

```
Loading WASM (embedded)
WASM loaded in 902.0617ms

[HP_GetStatistics] result=OK
  NumBodies=0  NumShapes=0  NumConstraints=0  NumWorlds=0

[HP_World_Create] result=OK  worldId=201488
[HP_World_SetGravity] result=OK  g=(0, -9.81, 0)
[HP_World_SetIdealStepTime] result=OK  dt=0.01667

[ground] static box 100x1x100 at (0,-0.5,0) added
[cube] dynamic box 1x1x1, mass=1, at (0,2,0)

Simulating 180 steps (dt=0.01667 s) — free fall, landing, rest:
  step      time   position (x,y,z)           linear velocity
     0     0.000   (  0.000,   2.000,   0.000)   (  0.000,   0.000,   0.000)
    30     0.500   (  0.000,   1.383,   0.000)   (  0.000,  -1.666,   0.000)
    60     1.000   ( -0.000,   0.502,  -0.000)   (  0.008,   0.144,   0.016)
    90     1.500   (  0.000,   0.500,   0.001)   (  0.000,  -0.000,   0.000)
   180     3.000   (  0.000,   0.500,   0.001)   (  0.000,  -0.000,   0.000)

Resting height y=0.5000 (expected ~0.50, |error|=0.0000)

Applying impulse (8, 0, 0) at the centre of mass…
  velocity immediately after impulse: (8.000, 0.000, 0.000)  [expect x ~8]
   240     4.000   (  1.285,   0.500,   0.001)   ( -0.044,  -0.003,   0.003) after 60 more steps

[HP_GetStatistics after sim] NumBodies=2  NumShapes=2  NumWorlds=1

Demo complete.
```

要点：立方体从 2 m 落到地面后**精确停在 y=0.5000**（半边长），速度归零；
施加 8 N·s 冲量后瞬时速度精确变为 8 m/s，随后在摩擦+阻尼下停止。

---

## 验证

```bash
cd havok
go test ./... -v
```

包含 7 项测试：

| 测试 | 验证内容 |
|------|----------|
| `TestEnumsMatchWasm` | Go 枚举常量与 wasm embind 注册值逐一比对（防漂移守卫） |
| `TestFallingBoxSettlesOnGround` | 动态体受重力下落、与静态地面碰撞、精确停在半边长处 |
| `TestImpulseSetsVelocity` | 冲量 → 速度（v = J/m）、`SetLinearVelocity` 往返 |
| `TestTransformAndStatic` | `QTransform` 往返（含非单位四元数）、静态体在重力下不动 |
| `TestMassPropertiesRoundTrip` | `MassProperties` f32 布局往返 |
| `TestMaterialAndRayCast` | `PhysicsMaterial` 往返；射线命中距离/法线/命中体；`ignoreBody` 生效 |
| `TestDeterministic` | 相同初始条件下两次模拟结果逐位一致 |

---

## 使用 havok 包

`havok` 是一个独立子模块（`github.com/gsw945/havok-go/havok`），可直接在其他 Go 项目中引用。

### 初始化

```go
import (
    "context"
    "github.com/gsw945/havok-go/havok"
    havokwasm "github.com/gsw945/havok-go/havok/wasm"
)

ctx := context.Background()

// 方式一：从文件加载
hp, err := havok.New(ctx, "HavokPhysics.wasm")

// 方式二：从内嵌字节加载（需先运行 convert 写入真实 wasm）
if havokwasm.IsReal() {
    hp, err = havok.NewFromBytes(ctx, havokwasm.WasmBytes)
}

if err != nil {
    log.Fatal(err)
}
defer hp.Close()
```

### 基本使用流程

```go
// 1. 创建世界 + 重力 + 固定步长
_, worldId, _ := hp.HP_World_Create(ctx)
hp.HP_World_SetGravity(ctx, worldId, havok.Vector3{0, -9.81, 0})
hp.HP_World_SetIdealStepTime(ctx, worldId, 1.0/60.0)

// 2. 形状（盒子：中心 / 朝向 / 半边长）
_, shapeId, _ := hp.HP_Shape_CreateBox(ctx,
    havok.Vector3{0, 0, 0},
    havok.Quaternion{0, 0, 0, 1},
    havok.Vector3{1, 1, 1})
hp.HP_Shape_SetMaterial(ctx, shapeId, havok.PhysicsMaterial{
    StaticFriction: 0.2, DynamicFriction: 0.2, Restitution: 0.2,
})

// 3. 刚体：形状 → 运动类型 → 质量属性 → 位姿 → 加入世界
_, bodyId, _ := hp.HP_Body_Create(ctx)
hp.HP_Body_SetShape(ctx, bodyId, shapeId)
hp.HP_Body_SetMotionType(ctx, bodyId, havok.MotionType_DYNAMIC)   // = 4，不是 2！
i := 1.0 / 6.0
hp.HP_Body_SetMassProperties(ctx, bodyId, havok.MassProperties{
    Mass: 1, InertiaTensorDiag: havok.Vector3{i, i, i},
    InertiaOrientation: havok.Quaternion{0, 0, 0, 1},
})
hp.HP_Body_SetQTransform(ctx, bodyId, havok.QTransform{
    Translation: havok.Vector3{0, 5, 0}, Rotation: havok.Quaternion{0, 0, 0, 1},
})
hp.HP_World_AddBody(ctx, worldId, bodyId, false)

// 4. 步进仿真
for i := 0; i < 60; i++ {
    hp.HP_World_Step(ctx, worldId, 1.0/60.0)
}

// 5. 读回状态
_, pos, _ := hp.HP_Body_GetPosition(ctx, bodyId)
_, vel, _ := hp.HP_Body_GetLinearVelocity(ctx, bodyId)

// 6. 释放资源
hp.HP_Shape_Release(ctx, shapeId)
hp.HP_Body_Release(ctx, bodyId)
hp.HP_World_Release(ctx, worldId)
```

### 返回值约定

所有 `HP_*` 方法均返回 `(Result, <值…>, error)`：

- `error` — wazero 调用层错误（罕见）
- `Result` — Havok 引擎结果码，用 `.IsOK()` / `.Error()` 判断（值为 HRESULT，见上表）
- `<值…>` — 实际返回数据（`HP_*Id`、`Vector3`、`float64` 等）

### 调用约定速查（生成代码遵循）

| 场景 | 规则 |
|------|------|
| 返回 `Result` | wasm 返回 i32 = Result |
| 返回 `[Result, T]` | 追加一个输出指针（参数**最后**），数据从该缓冲区 `+0` 读 |
| 返回 `[Result, T1, T2]` | 追加**两个**输出指针，按顺序对应 T1、T2 |
| `HP_*Id` / `bigint` 参数 | 按值传 i64 |
| `enum` / `boolean` / 计数 参数 | 按值传 i32 |
| `float` 参数 | 按值传 f32 |
| 结构体参数 | 传指针（f32 布局，自然 C++ 对齐：i64 对齐 8，f32/i32 对齐 4） |

---

## 项目搭建记录

```bash
mkdir havok-go
cd havok-go/
go mod init github.com/gsw945/havok-go
touch main.go
```
