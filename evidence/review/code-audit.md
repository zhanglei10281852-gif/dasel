# 在查询中定义带独立调用作用域的函数：实现审计

题目 tomwright-dasel-54cf53db-9cab-4aa8-91ec-7cd5664a8023；开始 syncRevision=7；固定基线 521d3c39fe06b276f65695d3ae27bd46b76c4106。冻结七条 Rubric 的原文、顺序和类型见 verify_rubric.yaml；只评价原 Trae Session 6af00849443b5a2122696972 的代码，无模型续跑或生产代码修补。

## 入口与状态所有权

CLI internal/cli/run.go:96 与 Go api.go:14 均进入 execution.ExecuteSelector，再经 selector.Parse → parser.Parse:83 → ProgramExpr → ExecuteAST → programExprExecutor → callExprExecutor → userFuncCallExecutor。无声明时 Parse 返回原表达式，未引入 ProgramExpr。parse_def.go:16 仅识别前导 def name(，普通 def 字段保留属性语义。

Options.userFuncs 持有当前查询的声明；execute_def.go:20 保存旧表，构建新表，defer 恢复，不向 Options.Funcs 注册或删除。原生名称通过 Funcs.Get 验证。Options.scope 持有每次调用的绑定；options.go:87/95 压栈和退栈，103 沿父链读取。execute_binary.go:108 对纯变量左值在当前帧重新绑定，顶层分支仍执行原有 Value.Set 全局赋值。对象字段赋值继续经过 propertyExprExecutor、model.Value.GetMapKey:85 和 Value.Set；setFn 将修改写回原容器，未复制对象。将形参重新绑定为新对象不替换调用方对象。

## 执行和错误顺序

execute_def.go:46 在求参前拒绝活动调用链中的同名函数；53 经 prepareArgs 逐个在调用方当前数据上执行，随后检查参数数目；61 压入调用链并 defer 移除；69 将参数引用绑定到新帧，74 压帧并 defer 退帧；77 在调用位置数据上执行函数体。chainedExprExecutor 按现有分号语义传递前一表达式结果并返回最后结果。函数读取当前数据和既有迭代 key；withKeyVar 在当前帧保存、设定、恢复局部 key。

真实失败交错使用原生 toInt("bad")：外层绑定 x=11，内层绑定 x=99、local=3，原对象字段写为4，随后转换报错。错误通过 %w 包装保留 strconv.ParseInt 原因，内外调用依次执行 defer 退帧/退调用链，ProgramExpr 恢复函数表。返回后宿主 x=7、无 local、对象字段4、旧 outer 为未知函数；同一 Options 重定义后返回42。该检查走真实生产调用链，无 mock；已完成字段不回滚符合 Prompt。失败前后的资源是同步调用帧和表，没有新 goroutine、服务、socket、事务或子进程，取消/超时/回滚外部资源不属于本需求。

## 验证与环境适用性

Windows Go 完整套件使用 -count=1 实际执行通过；Linux 另在 unshare -n 无公网路由、仅 loopback 的网络命名空间中执行，GOTOOLCHAIN=local、隔离模块缓存和空构建缓存、-mod=vendor，依赖由 offline/vendor.tar.gz 固定提供。依赖归档摘要见 offline/materials.json，源文件逐项摘要见 source-sha256.json。Linux Go 1.27.2 满足 go.mod 的1.25要求。20个测试包通过，2112条测试及子测试通过，0失败；另四个包没有测试，KDL 两项既有 skip 由 parsing/kdl/internal/spec_v1_test.go:113 与 spec_v2_test.go:910 的兼容策略触发，这两个文件相对基线未改动，def 核心测试无 skip。16项独立检查和真实 CLI/Go 公共入口验证全部通过，原始输出已归档。

核心入口只处理本地值、文件和固定 Go 模块，不要求数据库、Docker daemon、容器 fixture、FUSE、挂载、云账号或在线分发。源码及测试检索无容器 fixture、网络访问或测试子进程，WSL 仅用于 Linux 原生验证，不启动 Docker 服务或容器。原题语言执行契约在普通 Linux 进程可复现，符合共享参考的纯本地验收范围。

逐条直接证据见 ../rubric-comparison.txt；所有七条通过，无本次改动引入的阻断回归。登记65轮小于100，完成度完成，因此不收录；仍进入交付质检，质检通过后保持该步骤，不进入生产交付。本次仅复盘，不执行质检通过或远程题目提交。
