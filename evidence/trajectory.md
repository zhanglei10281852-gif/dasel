# 在查询中定义带独立调用作用域的函数

> **Workspace:** F:\swe\099924\project-01

---

**User**

维护一段反复处理不同对象的查询时，开发者应能给共同的变换表达式命名，并在同一查询的不同位置传参调用，例如 def label($suffix) { name + $suffix }; items.map(label("!"))。请支持在查询开头用 def 名称($形参, ...) { 函数体 }; 声明具名函数，函数体使用现有表达式和分号串联语义，以最后一个表达式的结果返回；声明本身不消耗或替换输入，CLI 和 Go 执行入口使用相同语法。函数可调用原生函数，也可沿非递归调用链调用本查询声明的其他函数，每个实参在调用方当前数据上仅求值一次，函数体从调用位置的当前数据开始执行。每次调用都拥有自己的形参和普通变量绑定：可以读取调用方变量，给形参或普通变量名赋值只改变本次调用的绑定，嵌套调用遮蔽同名变量后返回不得改变外层绑定；通过形参对象或调用方已提供的 $root 修改字段仍写回原对象，不能因隔离变量而复制掉这些修改，$this 和已有的 $key 继续反映函数体当前求值节点及迭代上下文。函数定义只在本次查询有效，不得写入宿主的原生函数集合；重复定义、重复形参、与可用原生函数同名、参数个数不符及直接或间接递归调用须返回明确错误，不能进入无限调用。函数体求值失败时应保留原错误信息、退出所有相关调用并恢复调用方绑定，已明确完成的对象字段修改不要求回滚；Go 调用方即使用相同 Options 再次执行，也不能看到上次查询的函数定义或局部变量，改正查询后可以正常调用。没有函数声明的查询、原生及宿主注册函数、CLI --var、普通变量的现有全局赋值语义，以及名为 def 的普通数据字段都保持兼容。

**Assistant**

我先了解项目结构，并查阅相关经验。

toolName: view_folder

status: success

f:\swe\099924\project-01

这是 Dasel（Go 实现的类 jq 查询引擎）项目。我先深入阅读核心架构文件。

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\parser.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\ast\ast.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\context.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\execute.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\func.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\execute_func.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\options.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\execute_assign.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_func.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_variable.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\lexer\token.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\lexer\tokenize.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\ast\expression_complex.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_symbol.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\denotations.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\execute_binary.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\api.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\internal\cli\query.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\model\value_set.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\internal\cli\run.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\internal\cli\variable.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\error.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_object.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_map.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\execute_each.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\execute_variable_test.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\execute_test.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_group.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\execute_key_var_test.go

核心机制已基本清楚。再看几个关键文件：`parseExpressionsFromTo` 定义、parser 测试风格、赋值测试、CLI 测试方式。

toolName: search_by_regex

status: success

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\parser_test.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\execute_assign_test.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_if.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\execute_map.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\internal\cli\generic_test.go

toolName: search_by_regex

status: success

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\execute_reduce.go

再看 model.Value 的结构和 CLI 测试脚手架，然后开始设计实现。

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\model\value.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\internal\cli\command_test.go

实现方案已成型。再确认几个测试约定和 AST 测试是否枚举了所有表达式类型。

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\execute_error_test.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\ast\ast_test.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\README.md

方案明确，建立任务清单并开始实现。先确认 parser 的所有调用入口和 Go 版本。

toolName: todo_write

status: success

Todos updated: 6 items

toolName: search_by_regex

status: success

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\go.mod

单一解析入口确认。开始实现——首先新增 AST 节点。

toolName: Write

status: success

filePath: f:\swe\099924\project-01\selector\ast\expression_function.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\selector\ast\ast_test.go

changes: undefined

现在实现 parser：新增 `parse_def.go` 并改造 `Parse()`。

toolName: Write

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_def.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\selector\parser\parser.go

changes: undefined

Parser 完成。现在改造执行层——先给 Options 增加作用域链、用户函数表和调用链。

toolName: todo_write

status: success

Todos updated: 6 items

toolName: Write

status: success

filePath: f:\swe\099924\project-01\execution\options.go

changes: undefined

现在把所有变量访问点切换到作用域接口。

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute.go

changes: undefined

现在改造赋值运算符——函数作用域内采用重绑定语义，顶层保持原有全局赋值语义。

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute_binary.go

changes: undefined

现在让 reduce 的 `$acc`/`$this` 处理也感知作用域。

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute_reduce.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute_reduce.go

changes: undefined

现在创建用户函数的执行器，并在调用分发处接入。

toolName: todo_write

status: success

Todos updated: 6 items

toolName: Write

status: success

filePath: f:\swe\099924\project-01\execution\execute_def.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute_func.go

changes: undefined

核心代码完成，先编译验证。

toolName: Shell

status: success

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute_def.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute_def.go

changes: undefined

toolName: Shell

status: success

编译通过。先跑一遍现有测试确认无回归。

toolName: Shell

status: success

现有测试全部通过。编写新测试前，确认两个细节：原生函数名和赋值是否需要 unstable 开关。

toolName: search_by_regex

status: success

toolName: search_by_regex

status: success

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\func_to_upper.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\func_len.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\parser_test.go

测试约定已清楚。先确认 CLI JSON 数组输出格式，然后编写三层测试。

toolName: search_by_regex

status: success

现在编写 parser 层测试。

toolName: Write

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_def_test.go

changes: undefined

toolName: Shell

status: success

两个问题：`map(...)` 是 MapExpr 而非 CallExpr；空函数体未报错。先在 parser 中补上空体校验。

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_def.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_def_test.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_def_test.go

changes: undefined

toolName: Shell

status: success

map 体内单表达式会被解包为直接的 CallExpr，修正测试断言。

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_def_test.go

changes: undefined

Parser 测试通过后，编写执行层综合测试。

toolName: Write

status: success

filePath: f:\swe\099924\project-01\execution\execute_def_test.go

changes: undefined

修正两个 `$root` 测试——执行层测试不会自动注入 `$root`，需要手动构造。

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute_def_test.go

changes: undefined

toolName: Shell

status: success

大部分失败是测试用例本身的问题（def 只能在查询开头、`add` 是原生函数名）。还有两个解析问题需要排查，先看二元解析逻辑。

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\parser_binary.go

直接写个最小复现来定位两个解析失败。

toolName: Shell

status: success

toolName: Write

status: success

filePath: f:\swe\099924\project-01\selector\parser\zz_repro_test.go

changes: undefined

toolName: Shell

status: success

toolName: Write

status: success

filePath: f:\swe\099924\project-01\selector\parser\zz_repro_test.go

changes: undefined

toolName: Shell

status: success

关键发现：`$x.y = 1` 这种点链变量赋值在现有语法里本身就不支持（与 def 无关），括号形式可以。但 `$o.name = "z"` 测试却通过了，需要进一步对比确认。

toolName: Write

status: success

filePath: f:\swe\099924\project-01\selector\parser\zz_repro_test.go

changes: undefined

toolName: Shell

status: success

真相大白：`count` 是 lexer 保留关键字（与 def 无关），且 `add` 是原生函数名。全部是测试用例命名/位置问题。修正测试。

toolName: delete_file

status: success

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute_def_test.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute_def_test.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute_def_test.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute_def_test.go

changes: undefined

还有一个泄漏测试的 def 位置也要前置。

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\execution\execute_def_test.go

changes: undefined

toolName: Shell

status: success

执行层全部通过。现在补 CLI 端到端测试，先看现有 `--var` 测试的输出约定。

toolName: search_by_regex

status: success

toolName: search_by_regex

status: success

toolName: search_by_regex

status: success

现在编写 CLI 端到端测试。

toolName: Write

status: success

filePath: f:\swe\099924\project-01\internal\cli\def_test.go

changes: undefined

修正函数名中的空格（会被切成两个 Symbol）。

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-01\internal\cli\def_test.go

changes: undefined

toolName: Shell

status: success

CLI 测试全部通过。运行完整测试套件和 vet 做全面回归。

toolName: todo_write

status: success

Todos updated: 6 items

toolName: Shell

status: success

全部通过。再跑 lint 和一次真实 CLI 冒烟验证。

toolName: Shell

status: success

toolName: Shell

status: success

lint 零问题。用 PowerShell 管道重跑冒烟。

toolName: Shell

status: success

三个冒烟场景全部符合预期（递归错误退出码 1 是正确行为）。最后通读一遍核心改动文件做收尾审查。

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\execution\execute_def.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-01\selector\parser\parse_def.go

再用 race 检测器跑执行层和 CLI 测试，并查看最终 diff 概览。

toolName: Shell

status: success

toolName: Shell

status: success

toolName: todo_write

status: success

Todos updated: 6 items

`def` 具名函数声明已完整实现，全部测试（含 `-race`）、`go vet`、`golangci-lint` 均通过。

## 实现概览

**语法层（上下文关键字，零 lexer 改动）**
- [parse_def.go](file:///f:/swe/099924/project-01/selector/parser/parse_def.go)：仅在查询开头且满足 `def name(` 形态时才识别为声明，因此名为 `def` 的普通数据字段仍按属性解析；支持零/多形参（`$` 变量）、`{}` 函数体内分号串联、声明末尾强制分号；解析期即报重复定义、重复形参、空函数体等错误。
- [expression_function.go](file:///f:/swe/099924/project-01/selector/ast/expression_function.go)：新增 `FunctionExpr` 与 `ProgramExpr`；无 def 的查询返回原 AST，结构完全不变。
- [parser.go](file:///f:/swe/099924/project-01/selector/parser/parser.go#L83-L102)：`Parse()` 先收集前导声明再解析主表达式。CLI 与 Go 入口共用 `selector.Parse`，语法天然一致。

**执行层（作用域链 + 隔离注册）**
- [options.go](file:///f:/swe/099924/project-01/execution/options.go)：新增 `varScope` 作用域链、`userFuncs` 用户函数表、`callChain` 调用链，及 `lookupVar/setVar/pushScope` 等辅助方法；顶层（无活动调用）全部回退到原 `Vars` 全局语义。
- [execute_def.go](file:///f:/swe/099924/project-01/execution/execute_def.go)：`ProgramExpr` 执行时换入全新函数表并 defer 恢复——不写入原生 `FuncCollection`，同一 Options 重跑看不到上次定义；调用时先做递归检测（直接/间接都报 `recursive call to function "x" is not allowed`，不会无限调用），实参经现有 `prepareArgs` 在调用方数据上**只求值一次**且按引用传递（不复制对象），再压入独立作用域执行函数体，defer 保证出错也恢复调用方绑定；参数个数不符、与原生/宿主函数同名均有明确错误。
- [execute_binary.go](file:///f:/swe/099924/project-01/execution/execute_binary.go#L108-L128)：函数作用域内 `$x = ...` 一律重绑定当前调用，不触碰外层绑定与共享包装值；而 `$o.field = ...`、`$root.field = ...` 走属性 `Set`，修改仍写回原对象。
- [execute.go](file:///f:/swe/099924/project-01/execution/execute.go)、[execute_reduce.go](file:///f:/swe/099924/project-01/execution/execute_reduce.go)：`$this`、`$key`、`$acc`、变量读取全部改走作用域接口，函数体内继续反映当前求值节点与迭代上下文。

**测试**：[parse_def_test.go](file:///f:/swe/099924/project-01/selector/parser/parse_def_test.go)、[execute_def_test.go](file:///f:/swe/099924/project-01/execution/execute_def_test.go)（含需求示例 `items.map(label("!"))`、实参一次求值、嵌套遮蔽、对象改回写、Options 复用无泄漏、错误信息保留等 30 个用例）、[def_test.go](file:///f:/swe/099924/project-01/internal/cli/def_test.go)（CLI 端到端，含 `--var` 与 `def` 字段兼容）。

冒烟验证：

```text
'{"items":[{"name":"a"},{"name":"b"}]}' | dasel query 'def label($suffix) { name + $suffix }; items.map(label("!"))'
# [ "a!", "b!" ]
'{"def":1}' | dasel query 'def'   # 1，字段兼容
'def f(){f()}; f()'              # 明确报错 recursive call ... is not allowed
```
