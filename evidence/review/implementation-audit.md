# 原 Prompt 范围与真实生产路径审计

题目 tomwright-dasel-963a36fb-d904-45a5-ae55-b49bf36fe169，基线 521d3c39fe06b276f65695d3ae27bd46b76c4106。只审计 project-02 当前模型产物，不修改生产实现。冻结标准及完整 Session 见 question-start.json 和 trajectory.metadata.json；扫描87轮只核对来源、范围和Session，不重算。

## 入口、状态与副作用

cmd/dasel/main.go 注册格式，调用 cli.MustRun → cli.Run（internal/cli/command.go:50）→ QueryCmd.Run（query.go:31）。--stream 参数检查先于标准输入读取，格式默认JSON，--it直接拒绝。普通分支仍走原run。流分支 runStream（run.go:123）每次 Next 得到一个model.Value，recordNum从1递增，随后深复制初始变量，新建execution.Options及Vars，root绑定当前输入，再ExecuteSelector。外部状态边界只有stdin读取和stdout.Write；没有数据库、容器、挂载、云账号、HTTP服务或异步worker。

初始--var由variableMapper解析；runStream只读取初始变量，copyStreamValue逐层重建map/slice并复制标量（run.go:196），不把上条Options写回。字段修改经executeAssign → Value.Set → setFn/可寻址模型，在当前root及当前变量副本内生效。新建变量和this属于每条新Options，下一条不可见；probe的条件创建变量验证第二条返回variable z not found。这里不要求撤销已执行记录内字段修改。

## 输出提交、失败与背压

每条真实查询后，writer.Write在内部bytes.Buffer完整生成整个结果，返回错误时不调用stdout.Write；序列化成功后同步stdout.Write，返回后才开始下一轮Next/ExecuteSelector。JSON writer的分支和展开结果由既有writer语义处理。输入n=0使查询[a,1.0/a]产生+Inf，调用真实goccy JSON Encoder报unsupported value；第一条输出完整保留，第二条无半成品，第三条未执行。此路径没有替换编码器或强制mock错误。

受控writer第二次Write建立channel屏障；观察原生函数计数停在2后释放，成功场景计数才变3；失败场景接受一个字节后返回controlled sink failure，count保持2。测试注册reviewTick只测量真实执行次数，不替换查询、解析、编码或发布逻辑。测试辅助goroutine均由done/finished channel等待，io.Pipe在完成后关闭；生产执行循环不创建goroutine。无新增取消或服务关闭契约。

## Rubric 4 的确定缺口

run.go:149 在recordNum递增及解析错误包装之前，对任何errors.Is(err,io.EOF)返回nil。decodeObject/decodeArray在半文档内Token耗尽返回EOF，StreamReader.Next将其传回，最终该错误被runStream吞为成功。另一个真实分支：goccy Stream.More（依赖stream.go:115）遇']'或'}'直接false，json_reader.go:89-93将其当EOF。真实二进制输入{"a":1}{"a":、{"a":1}[1,、{"a":1}]均输出1换行、stderr空、退出0，未报告record 2。原TestStreamReader只验证Next错误非nil，未走runStream的errors.Is分支；原坏JSON测试用{bad}，覆盖的是非EOF错误。

该缺口判在新增stream场景的失败报告契约，不将基线普通读取器的已有语法宽松另算回归。真实toInt转换失败会正确保留ParseInt错误、报record 2并停止；更正后重新调用从初态正常执行。这些通过子路径不能抵消截断及非法闭合尾部不报错。

## Rubric 6 的确定缺口

流reader持有同一个goccy Decoder，递归decodeValue只调用Token，Next调用More；goccy decode.go:244-245直接委托s.Token，无Reset；只有DecodeWithOption路径在236行Reset，该入口未被生产流调用。依赖stream.go:191-209 readBuf在已填满时增大bufSize、复制原buf前缀，cursor随历史数据累计增长。固定大小39字节记录：1024条时buf长度65536；32768条时buf长度2097152、cursor1277951，总输入1277952。反射读取仅观察实际私有缓冲长度与cursor，不修改解码器。10次重复结果一致。持续流的输入历史被累计保存，违反原Prompt“不累计整个输入”，不以新增容量上限判错。

背压与输出失败处理子路径通过；输出每条独立序列化不累计全部输出。该条因为累计输入而未通过。

## 默认兼容、环境与证据边界

非流run的原逻辑未改变，只新增独立分支；完整原套件及原多格式矩阵实际运行。cmd/dasel/main.go注册全部格式；独立Go探针也显式注册YAML，真实CLI JSON转YAML返回a: 1。基线KDL两项已有兼容性skip保持，核心stream全部实际执行。没有testcontainers/RedisContainer/DockerContainer或核心网络服务fixture；这是纯本地文件/值/管道任务，属于甲方支持范围。WSL只执行原生Linux进程，不启动任何Docker服务或容器；用unshare -n阻断公网，路由探测Network unreachable，只存在lo；Go模块和构建缓存均为空目录，GOPROXY/GOSUMDB关闭，使用当前go.mod/go.sum生成且随产物提供的vendor归档。脚本与原始日志保存在evidence/offline和evidence/review。

探针开发时修正了辅助writer第三次重复关闭channel，以及探针缺少YAML注册这两项验证工具问题；未修改生产代码。最终可用证据是independent-probe.log、repeated-probe.log及real-cli-probe.jsonl；历史探针尝试日志只保存在工作目录。结果为五条通过、Rubric 4及6未通过，存在可用主路径，完成度部分完成；87轮推导收录：难题。
