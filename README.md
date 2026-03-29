# havok-go

Go 版 [Havok 物理引擎](https://github.com/BabylonJS/havok) 绑定，通过 [wazero](https://github.com/tetratelabs/wazero)（纯 Go WebAssembly 运行时）包装 `HavokPhysics.wasm`，无 CGo 依赖。

## 目录结构

```
havok-go/
├── main.go              # CLI 入口
├── cmd/                 # Cobra 子命令（convert / example）
├── converter/           # TypeScript d.ts 解析 & Go 代码生成器
│   ├── model.go         # 数据模型
│   ├── parser.go        # tree-sitter 解析器 + 类型辅助方法
│   └── generator.go     # 代码生成模板
└── havok/               # Havok WASM 绑定（子模块）
    ├── havok.go         # HavokPhysics 初始化 & 所有 HP_* 方法
    ├── types.go         # 共用类型（Result、Vector3、Quaternion 等）
    ├── memory.go        # WASM 内存读写工具
    ├── imports.go       # emscripten env 桩函数
    ├── wasm/
    │   ├── wasm.go      # //go:embed HavokPhysics.wasm
    │   └── HavokPhysics.wasm  # 真实 wasm（由 convert 命令写入）
    └── generated/       # convert 命令生成的脚手架代码
```

---

## 编译

> 依赖 Go 1.23+，tree-sitter 绑定需要 CGo（需安装 C 编译器，如 TDM-GCC、MinGW-w64 等）。

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
> 本项目通过 CGo 使用 [tree-sitter](https://tree-sitter.github.io/) 解析 TypeScript。
> 在 Windows 上使用 TDM-GCC（`--enable-threads=posix` / winpthread 变体）时，
> 默认的外部链接器（`ld`）会向 PE 可执行文件注入 `.CRT`、`.tls` 等节区，
> 与 Go 1.21+ 运行时的 SEH（结构化异常处理）机制冲突，导致编译出的 `.exe` 无法运行。
>
> **修复**：添加 `-ldflags "-linkmode internal"` 强制使用 Go 内置链接器，完全绕过 `ld`。
>
> 已提供 `build.cmd` 脚本（Windows）封装了该参数，直接运行即可：
>
> ```bat
> build.cmd
> ```

---

## 更新 Havok wasm

当需要升级 Havok 版本或首次初始化时，按以下步骤操作：

### 1. 准备 BabylonJS havok 源

```bash
# 克隆或下载 @babylonjs/havok 包
git clone https://github.com/BabylonJS/havok  BabylonJS-havok
```

或者直接使用 npm 下载：

```bash
cd BabylonJS-havok
npm install
```

### 2. 运行 convert 命令

```bash
# 默认输出到 ./havok/generated/，并自动将 wasm 复制到 ./havok/wasm/
go run . convert --input ../BabylonJS-havok/packages/havok/HavokPhysics.d.ts

# 如果 wasm 文件在非标准位置，用 --wasm 手动指定
go run . convert \
  --input ../BabylonJS-havok/packages/havok/HavokPhysics.d.ts \
  --wasm  ../BabylonJS-havok/packages/havok/lib/esm/HavokPhysics.wasm
```

命令完成后会输出：

```
Parsing HavokPhysics.d.ts …
Found 10 enums, 31 type aliases, 130 methods
Generated:
  ./havok/generated/types_gen.go
  ./havok/generated/bindings_gen.go
Copied wasm: ../BabylonJS-havok/packages/havok/lib/esm/HavokPhysics.wasm
  → havok\wasm\HavokPhysics.wasm
```

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--input` / `-i` | （必填）| `HavokPhysics.d.ts` 路径 |
| `--output` / `-o` | `./havok/generated` | 生成文件的输出目录 |
| `--package` / `-p` | `generated` | 生成文件的 Go 包名 |
| `--wasm` / `-w` | （自动搜索）| 指定 wasm 文件路径 |

---

## 运行示例

`example` 命令演示了一个完整的物理仿真流程：创建世界、添加刚体、施加重力、步进模拟、释放资源。

```bash
# 使用 havok/wasm/ 中内嵌的 wasm（先执行 convert 写入真实 wasm）
go run . example

# 或手动指定 wasm 路径（跳过内嵌检查）
go run . example --wasm path/to/HavokPhysics.wasm
```

正常输出示例：

```
Loading WASM (embedded)
WASM loaded in 1.46s

[HP_GetStatistics] result=OK
  NumBodies=0  NumShapes=0  NumConstraints=0

[HP_World_Create] result=OK  worldId=201488
[HP_World_SetGravity] result=OK
[HP_Body_Create] result=OK  bodyId=268912
[HP_Shape_CreateSphere] result=OK  shapeId=330544
[HP_Body_SetShape] result=OK
[HP_Body_SetMotionType] result=OK
[HP_World_AddBody] result=OK

Simulating 5 steps (dt=0.0167 s)…
  step 0: OK  ...  step 4: OK

Demo complete.
[HP_Shape_Release] result=OK
[HP_Body_Release] result=OK
[HP_World_Release] result=OK
```

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
// 1. 创建世界
res, worldId, err := hp.HP_World_Create(ctx)

// 2. 设置重力
hp.HP_World_SetGravity(ctx, worldId, havok.Vector3{X: 0, Y: -9.81, Z: 0})

// 3. 创建刚体
res, bodyId, err := hp.HP_Body_Create(ctx)

// 4. 创建形状（球体）
center := havok.Vector3{X: 0, Y: 10, Z: 0}
res, shapeId, err := hp.HP_Shape_CreateSphere(ctx, center, 1.0)

// 5. 绑定形状 & 设置运动类型（动态刚体）
hp.HP_Body_SetShape(ctx, bodyId, shapeId)
hp.HP_Body_SetMotionType(ctx, bodyId, havok.MotionType_DYNAMIC)
hp.HP_World_AddBody(ctx, worldId, bodyId, havok.ActivationState_ACTIVE)

// 6. 步进仿真
dt := float32(1.0 / 60.0)
for i := 0; i < 60; i++ {
    res, err = hp.HP_World_StepSimulation(ctx, worldId, dt)
}

// 7. 释放资源（建议用 defer）
hp.HP_Shape_Release(ctx, shapeId)
hp.HP_Body_Release(ctx, bodyId)
hp.HP_World_Release(ctx, worldId)
```

### 返回值约定

所有 `HP_*` 方法均返回 `(Result, <值>, error)`：

- `error` — wazero 调用层错误（罕见）
- `Result` — Havok 引擎级别的结果码，通过 `.IsOK()` 和 `.Error()` 判断
- `<值>` — 实际返回数据（如 `HP_WorldId`、`HP_BodyId` 等句柄）

---

## 项目搭建记录

```bash
mkdir havok-go
cd havok-go/
go mod init github.com/gsw945/havok-go
touch main.go
```