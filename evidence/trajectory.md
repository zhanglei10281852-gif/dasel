# 开放 JSON 管道的逐文档独立执行

> **Workspace:** F:\swe\099924\project-02

---

**User**

向长期保持打开的管道连续写入 JSON 数据时，流水线作者希望先拿到已完成文档的查询结果，而不是等待输入端关闭。为 query 增加显式 --stream 模式，仅接受 JSON 输入和 JSON 输出，省略输出格式时沿用 JSON；不支持的格式及与 --it 的组合应在读取输入前报错。这个模式把每个顶层 JSON 值作为一条输入记录：文档可以跨多次读取或占多行，数组仍是一条记录，标量按 JSON 语法提供必要分隔符；一条完整记录到达后即可执行查询并输出，不等待下一条或 EOF，空流不执行查询也不输出 null。每条记录都从自己的 $root 和同一份 --var 初始值开始，前条记录对变量内部对象的修改、新建变量或输入对象的修改不能带入后条；记录内的现有查询、字段修改、--root、--compact 和 JSON 写出选项继续生效，查询产生分支或展开结果时保留现有多文档输出语义。按输入顺序完成每条记录的查询和完整序列化后再发布结果，不累计整个输入或全部输出，输出受阻时不得继续执行后续记录的查询。若第 n 条记录解析、查询或序列化失败，应保留先前已经输出的结果，当前记录不得输出半成品，停止处理后续记录并以从 1 开始的输入记录序号说明错误；输出端写入失败也应报告错误并停止后续查询，但不要求撤回输出端已经接受的字节。用户修正输入或查询后重新调用，应从本次输入和变量初态正常执行，不延续上次失败的状态。未使用 --stream 时，普通查询的整批多文档处理、空输入行为及跨格式转换均维持原样。

**Assistant**

toolName: view_folder

status: success

f:\swe\099924\project-02

This is the **dasel** project. Let me explore the key files for the query command and input/output handling.

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\internal\cli\query.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\internal\cli\run.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\internal\cli\read_write_flag.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_reader.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_writer.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\json\json.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\internal\cli\variable.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\format.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\reader.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\writer.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\execution\options.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\execution\execute.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\execution\context.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\execution\execute_assign.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\internal\cli\command.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\model\value.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\model\go_value.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\model\value_map.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\model\value_set.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\execution\execute_binary.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\selector\parser\parse_variable.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\model\value_slice.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\model\value_literal.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\internal\cli\config.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\internal\cli\command_test.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\model\value_metadata.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\go.mod

Let me check the goccy go-json Decoder streaming behavior in the module cache.

toolName: Shell

status: success

toolName: search_by_regex

status: success

toolName: view_files

status: success

filePath: C:\Users\czxy4\go\pkg\mod\github.com\goccy\go-json@v0.10.6\decode.go

toolName: search_by_regex

status: success

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\internal\cli\interactive.go

toolName: search_by_regex

status: success

toolName: view_files

status: success

filePath: C:\Users\czxy4\go\pkg\mod\github.com\goccy\go-json@v0.10.6\internal\decoder\stream.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\d\reader.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

Let me check a few more details: null math behavior, the generic test file, and the README docs for flags.

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\model\value_math.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\internal\cli\generic_test.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\execution\execute_all.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\execution\execute_spread.go

Now let me check assignment test examples and group parsing to confirm query syntax for tests.

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\execution\execute_assign_test.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\selector\parser\parse_group.go

Now I have a full understanding. Let me set up a task plan and start implementing.

toolName: todo_write

status: success

Todos updated: 5 items

toolName: file_search

status: success

toolName: search_by_regex

status: success

Now implementing. First, the streaming JSON reader in [json_reader.go](file:///f:/swe/099924/project-02/parsing/json/json_reader.go):

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_reader.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_reader.go

changes: undefined

Now update [query.go](file:///f:/swe/099924/project-02/internal/cli/query.go) to add the `--stream` flag and validation:

toolName: Write

status: success

filePath: f:\swe\099924\project-02\internal\cli\query.go

changes: undefined

Now add `runStream` to [run.go](file:///f:/swe/099924/project-02/internal/cli/run.go):

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\internal\cli\run.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\internal\cli\run.go

changes: undefined

Let me verify compilation before writing tests:

toolName: Shell

status: success

Now mark progress and write tests. First, extend [json_test.go](file:///f:/swe/099924/project-02/parsing/json/json_test.go):

toolName: todo_write

status: success

Todos updated: 5 items

toolName: Shell

status: success

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

changes: undefined

The edit landed in the wrong place — the original file continued. Let me inspect and fix:

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

I need to move the inserted block to after the depth-limit test function. First remove it from the wrong spot:

toolName: edit_file_search_replace

status: failed

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

changes: undefined

toolName: search_by_regex

status: success

The earlier large replacement failed on whitespace. I'll fix the structure with two smaller edits:

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

changes: undefined

Now remove the orphaned duplicate block that follows `TestStreamReader`:

toolName: edit_file_search_replace

status: failed

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

changes: undefined

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

changes: undefined

Now add the `io` import to the test file:

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

changes: undefined

toolName: Shell

status: success

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

Line 843 should be `}` not `})`:

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

changes: undefined

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

Structure is fixed. Let me verify I didn't unintentionally rename that subtest via git diff:

toolName: Shell

status: success

I accidentally renamed that subtest — restore the original name:

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\parsing\json\json_test.go

changes: undefined

toolName: Shell

status: success

JSON tests pass. Now let me check the end of [command_test.go](file:///f:/swe/099924/project-02/internal/cli/command_test.go) to append CLI stream tests:

toolName: Shell

status: success

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\internal\cli\command_test.go

Let me verify how property access behaves on a scalar (to craft a query-error test):

toolName: search_by_regex

status: success

Now append the CLI stream tests to [command_test.go](file:///f:/swe/099924/project-02/internal/cli/command_test.go):

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\internal\cli\command_test.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\internal\cli\command_test.go

changes: undefined

Let me verify the actual dasel query syntax for indexing and `each` before running tests:

toolName: search_by_regex

status: success

toolName: search_by_regex

status: success

toolName: search_by_regex

status: success

Index syntax is `[0]`, not `.[0]`. Let me check `each`/`map` behavior and how queries start:

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\execution\execute_each.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\internal\cli\command_test.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\execution\execute_map.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\execution\execute_branch.go

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\selector\parser\parse_branch.go

toolName: search_by_regex

status: success

Let me confirm a bare `[0]` query parses at the start of a selector:

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\selector\parser.go

toolName: search_by_regex

status: success

Fix the two test queries to use valid syntax (`[0]` and `branch(...)`):

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\internal\cli\command_test.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\internal\cli\command_test.go

changes: undefined

Now run the new CLI tests:

toolName: Shell

status: success

Two test bugs: bare `[0]` is an array literal (need `$root[0]`), and the compact test queries a string scalar (compact makes no difference). Fix both:

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\internal\cli\command_test.go

changes: undefined

toolName: edit_file_search_replace

status: success

filePath: f:\swe\099924\project-02\internal\cli\command_test.go

changes: undefined

toolName: Shell

status: success

All tests pass. Now run the full suite, vet, and build:

toolName: todo_write

status: success

Todos updated: 5 items

toolName: Shell

status: success

toolName: Shell

status: success

All tests pass. Let me do an end-to-end smoke test with the real binary, including the "results before pipe closes" scenario:

toolName: Shell

status: success

toolName: Shell

status: success

First result emitted at t=0.0s while the pipe stays open — streaming works end-to-end. A few more CLI smoke checks:

toolName: Shell

status: success

All behaviors verified. Let me also confirm non-stream mode is unchanged (cross-format conversion still works) and that a fixed re-run succeeds:

toolName: Shell

status: success

Everything works. Clean up the smoke-test artifacts:

toolName: delete_file

status: success

toolName: Shell

status: success

Final verification — gofmt check and full test suite:

toolName: Shell

status: success

toolName: Shell

status: success

The gofmt listing is purely a CRLF artifact (`core.autocrlf=true`) affecting every file, including untouched ones. Verify my files are gofmt-clean apart from line endings:

toolName: Shell

status: success

toolName: Shell

status: success

The PowerShell pipeline may mangle input. Verify properly with a temp file:

toolName: Shell

status: success

gofmt genuinely wants a change there. Let me look at the exact block:

toolName: view_files

status: success

filePath: f:\swe\099924\project-02\internal\cli\command_test.go

toolName: Shell

status: success

The displayed diff was a PowerShell artifact. Do an authoritative byte-level comparison:

toolName: Shell

status: success

gofmt-clean. Final full test run (force fresh) and diagnostics:

toolName: Shell

status: success

toolName: Shell

status: success

toolName: GetDiagnostics

status: success

toolName: todo_write

status: success

Todos updated: 5 items

已完成 `query --stream` 模式，全部测试通过。

## 改动文件

**[parsing/json/json_reader.go](file:///f:/swe/099924/project-02/parsing/json/json_reader.go)** — 新增 `StreamReader`
- 基于 goccy 的流式 `json.Decoder` 直接读 `io.Reader`，`Next()` 每次返回一个完整的顶层 JSON 值；复用既有的 token 解码逻辑（保序 map、int/float 区分）。
- 一条记录可跨多次 `Read`、占多行；数组/对象/标量各算一条；干净结束（含纯空白流）返回 `io.EOF`。
- 附带 `errorTrackingReader`，防止底层读错误被解码器静默吞成 EOF。

**[internal/cli/query.go](file:///f:/swe/099924/project-02/internal/cli/query.go)** — 新增 `--stream` 标志与前置校验
- 仅接受 JSON 输入/输出；省略格式一律按 JSON（不看 config 默认格式）。
- `-i/-o` 非 JSON、与 `--it` 组合都在读取任何输入前报错（`ErrStreamInteractive`）。

**[internal/cli/run.go](file:///f:/swe/099924/project-02/internal/cli/run.go)** — 新增 `runStream` 与深拷贝
- 每条记录：解析 → 以独立 `$root` 和**深拷贝的初始 `--var`** 执行查询 → 整体序列化到字节 → 一次性写出，再处理下一条；空流不查询、不输出 `null`。
- 前条对 `--var` 内部对象/标量的修改、新建变量、输入对象修改均不泄漏；`--root`、字段修改、`--compact`、`--write-flag/--rw-flag` 继续生效；分支（`branch()`）/展开（`...`）结果仍走多文档写出。
- 同步循环天然实现背压：输出受阻时不会执行后续记录的查询；不累积整个输入或全部输出。
- 解析/查询/序列化/写入失败均停止后续处理，错误带从 1 开始的记录序号（如 `error parsing input record 3: ...`）；当前记录因先完整序列化再发布而不会出半成品，先前结果保留；写入失败不回滚已接受字节。进程无状态，修正后重新调用从初态执行。
- 未加 `--stream` 时，整批多文档、空输入输出 `null`、跨格式转换等行为完全不变。

## 验证

- 新增测试：`TestStreamReader`（混合值、数组单记录、标量分隔、空/空白流、逐字节送达的跨读文档、坏 JSON）与 `TestStream`（顺序、空流、多行、标量、compact、`--root`、变量重置与嵌套修改隔离、spread/branch 多文档、解析/查询/写入失败的序号与停止语义、格式与 `--it` 组合在读前报错），并加了非流式回归用例。
- `go build ./...`、`go vet ./...`、gofmt 均干净；全部 20 个包 `-count=1` 测试通过。
- 真实二进制 + `io.Pipe` 冒烟：第一条记录在 t=0.0s 即输出（管道保持开启 2.5s），第二条在 t=2.0s 到达后立即输出，不等待 EOF。
