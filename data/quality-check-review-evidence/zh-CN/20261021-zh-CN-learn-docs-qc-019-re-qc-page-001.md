===== INDEPENDENT RE-QC EVIDENCE =====

locale=zh-CN
review_id=20261021-zh-CN-learn-docs-qc-019-re-qc-page-001
rubric=translation-quality/v1
snapshot_id=learn-docs-v1-qc-019
previous_snapshot_id=learn-docs-v1-qc-018
working_set_kind=page

zip_sha256=1174df01081de8ba2bb06e81699c4001bf8bf6e78cccc5a69ed4ea8cdb3fe33b
bundle_identity_sha256=4a142bb4e0ff5485d222b4d0f5cf12971722333eafc77c2e71931379a29ab61d
input_gate=PASS
independent_review=yes
carry_forward_units=45
fresh_review_units=4

------------------------------------------------
stable_index=20
unit_id=d799614a9f1efd848505c615a88f2fb0f092ac22f743b3da042e12de1127fd8b
source_path=_content/doc/database/cancel-operations.md
rating=A
decision=approved
finding=none

qc-018 指出的“外层 上下文”“关联的 上下文”等不自然空格已经解决。
context.Context、context.WithTimeout、context.WithDeadline、QueryContext、queryCtx 与 ctx 的关系准确。
代码注释翻译完整，超时传播、取消行为和资源释放含义与 source 一致。

------------------------------------------------
stable_index=25
unit_id=872da6c618956d7ff25dbe8db7d898808b6a78fc1b91244ffcda0f59ce2b1983
source_path=_content/doc/database/open-handle.md
rating=B
decision=rejected

qc-018 中数据库凭据段落的语义关系已修复，开头的断裂标点也已明显改善。
完整 Page 仍存在局部中文排版和技术表达问题：
- DBMS自定义连接请求、sql.Open无法初始化句柄 缺少技术标识与中文之间的必要分隔。
- 由表示sql.Stmt预处理语句的对象持有的资源 中，技术标识与中文粘连，句法也不够清晰。
- 导入驱动程序段落仍保留英文双连字符 -- 充当中文句内标点，与周围中文表达不协调。

finding=仅修订上述局部标点、空格与资源说明的自然语言表达。保持 DBMS、sql.Open、sql.Stmt、database/sql、Connector、DSN、代码及链接 identity 不变。

------------------------------------------------
stable_index=27
unit_id=21946bff61d8f7c52c444edcd122ea398ba80574a8095c37befa936541a9cd4c
source_path=_content/doc/database/querying.md
rating=B
decision=rejected

qc-018 中句号与 inline token 分离的问题已经修复，两个函数表格中原先生硬的“保留的专用连接”也已修改。
仍存在以下局部问题：
- Rows结构、QueryRow最多、DBMS和驱动程序、TEXT转换、Rows.NextResultSet会准备 等技术标识直接粘连中文。
- Conn.QueryRowContext、Conn.QueryContext 的说明“用于预留连接”不能准确表达 source 的 For reserved connections；这些方法用于已预留的连接，并不负责预留连接本身。

finding=修正上述技术标识周围的中文空格，并收口两个 Conn 方法的用途说明，准确表达其在已预留连接上使用的语义。不得改变 API、HTML 表格结构、链接与 Go 代码。

------------------------------------------------
stable_index=47
unit_id=a69694fb9c89d47440249dfd760d315586297f634d199b13532ac95cbced201f
source_path=_content/doc/security/vuln/database.md
rating=C
decision=rejected

qc-018 所指出的两处 e.g 英文残留均已解决。
但“排除的报告”中的 NOT_IMPORTABLE 条目仍存在影响安全文档技术含义的问题。

canonical English source：
The vulnerability occurs in package main, an internal/ package only imported by package main, or some other location which can never be imported by another module.

current candidate：
漏洞出现在 main 包、某个 internal/ 包，仅由 main 包导入的包，或其他永远无法被其他模块导入的位置。

当前译文将 only imported by package main 与 internal/ package 的限定关系拆开，容易把任意 internal/ 包误列为独立条件，同时把“仅由 main 包导入的包”解释为另一类别，从而可能扩大安全漏洞排除规则的适用范围。

finding=精确修订 NOT_IMPORTABLE 条目，使 only imported by package main 的限定范围与 canonical English source 一致。保持 NOT_IMPORTABLE、main、internal/ 等技术 identity 和其余排除原因不变；不得扩大或缩小 source 定义的排除条件。

===== COUNTS =====
A=1
B=2
C=1
D=0
REVIEWED_UNITS=4
CARRY_FORWARD=45
MISSING=0
EXTRA=0

===== FINAL =====
RESULT=FAILED
LOCALE=zh-CN
SNAPSHOT=learn-docs-v1-qc-019
PREVIOUS_SNAPSHOT=learn-docs-v1-qc-018
REVIEW_ID=20261021-zh-CN-learn-docs-qc-019-re-qc-page-001
WORKING_SET_KIND=page
REVIEWED=4/4
A=1
B=2
C=1
D=0
CARRY_FORWARD=45
INPUT_GATE=PASS
INDEPENDENT_REVIEW=yes
REPLACEMENT_GENERATED=no
MUTATION=none
REMOTE_CALLS_USED=0
STOP=before-review-evidence-save-and-quality-check-record
