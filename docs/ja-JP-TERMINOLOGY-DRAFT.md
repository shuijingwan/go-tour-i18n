# ja-JP Terminology Draft

## 1. Purpose

本文档不是正式 glossary，而是在创建 `locales/ja-JP/glossary.yaml` 前进行术语评估的草案。它用于记录日语 Go 技术术语的候选译法、选择理由和待确认事项，为后续正式 glossary 的规则提供依据。

## 2. Terminology decision principles

术语选择按以下优先级进行：

1. Go 官方日语资料；
2. 日本 Go 社区常用表达；
3. 日语技术文档自然表达；
4. 项目内部一致性。

当高优先级来源与低优先级来源存在差异时，优先采用高优先级来源；尚无充分依据的术语保留为候选，暂不进入正式 glossary。

## 3. Initial terminology candidates

| English term | Candidate Japanese | Decision | Notes |
| --- | --- | --- | --- |
| package | パッケージ | 候选 | 日语技术文档中常见，需与 Go 官方日语资料核对。 |
| interface | インターフェース | 候选 | 常见技术术语；需确认 Go 语境中的写法与相关复合术语的一致性。 |
| method | メソッド | 候选 | 常用片假名表达；需确认与 receiver 等相关术语的搭配。 |
| struct | 構造体 | 候选 | 日语编程资料常见；需核对 Go 社区是否也使用「構造体型」。 |
| slice | スライス | 候选 | Go 专有概念通常保留片假名，需参考社区惯例。 |
| map | マップ | 候选 | 可能与一般地图含义混淆；需确认是否采用「マップ」或其他表达。 |
| pointer | ポインタ | 候选 | 常见技术术语；需确认长音符号和项目内写法。 |
| channel | チャネル | 候选 | Go 并发核心术语，需重点参考日本 Go 社区资料。 |
| goroutine | ゴルーチン | 候选 | 需确认日语社区是否倾向片假名化或保留英文 `goroutine`。 |
| module | モジュール | 候选 | 常见技术术语；需与 Go Modules 的产品/功能名称用法一致。 |
| constraint | 制約 | 候选 | 泛型语境可能采用「制約」；需核对官方日语资料。 |
| type parameter | 型パラメータ | 候选 | 泛型术语；需确认「型パラメーター」等表记差异。 |
| type assertion | 型アサーション | 候选 | 常见候选为片假名复合词，需确认社区是否使用意译。 |
| type switch | 型スイッチ | 候选 | 需确认与 `switch` 关键字说明中的日语表达是否一致。 |
| standard library | 標準ライブラリ | 候选 | 常见技术文档表达；需确认官方表记。 |
| exercise | 練習問題 | 候选 | 面向教程内容的自然表达，需确认页面语境是否适合。 |
| Run | 実行 | 候选 | UI 动词/按钮文本，需结合实际界面文风确认。 |
| Format | フォーマット | 候选 | UI 动词可能更适合「整形」；需结合 gofmt 功能和界面文风确认。 |
| slide | ページ | 候选 | 需确认教程单元在日语中应称「ページ」还是「スライド」。 |

## 4. Terms requiring community verification

以下术语需要进一步参考日本 Go 社区资料后再作正式决定：

- `channel`：确认「チャネル」及其在并发说明中的常见搭配。
- `goroutine`：确认保留英文还是采用「ゴルーチン」。
- `slide`：确认教程导航与内容单元更自然的日语称呼。
- `exercise`：确认教程中的练习内容应使用「練習問題」或其他表达。
- `map`：确认 Go 语境下的常用表记，并避免与非编程含义混淆。

## 5. Migration to glossary.yaml

完成术语确认后，按以下流程迁移：

```text
ja-JP terminology draft
        ↓
locales/ja-JP/glossary.yaml
```

只有已经确认、且具备明确依据的规则才进入正式 glossary；仍有分歧或缺少社区依据的候选术语继续保留在本草案中。
