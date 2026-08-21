# Translation Terminology Guide

## 1. Purpose

`glossary.yaml` 不是简单的英文到目标语言自动替换表。它记录的是目标语言项目维护者经过评估后作出的术语决策，用于在同一 locale 中保持技术表达的一致性、可读性和可维护性。

术语决策应服务于目标语言读者的理解，而不是追求逐词对应或将某一语言的表达机械迁移到另一种语言。

## 2. Terminology decision principles

### 2.1 Community consensus

如果目标语言社区已经存在稳定、广泛使用的技术表达，应优先参考社区共识。评估时可以参考该语言的官方技术资料、主流社区文档、长期维护的开源项目与教学材料。

社区共识不是绝对规则，但它通常比脱离使用场景的直译更能降低读者理解成本。

### 2.2 Multiple accepted translations

如果一个术语存在多个长期使用的译法，项目需要选择一个统一表达，并记录选择原因，例如目标读者习惯、与相关术语的区分能力或与现有教材的一致性。

选择统一表达不代表其他译法错误。术语表的职责是为本项目建立稳定的表达边界，而不是裁定目标语言社区中所有可能译法的对错。

### 2.3 Native technical terms

对于 Go 生态或编程领域中已经有稳定英文形式的术语，不强制翻译。术语可以保留英文形式，也可以采用目标语言表达；具体选择应根据目标语言社区的阅读与技术交流习惯决定。

尤其应避免为了形式上的“全量翻译”而改变读者已熟悉的标识、命令、包名或技术名称。

## 3. Glossary categories

每个 locale 的 `glossary.yaml` 使用以下类别表达术语政策：

### mandatory

必须保持一致的术语规则。适用于在课程中反复出现、需要统一显示形式的核心概念。

### preferred

推荐模型和译者优先使用的表达。它提供上下文指导，但不应代替对完整句意和目标语言自然表达的判断。

### forbidden

明确避免使用的表达。通常用于会造成歧义、与项目既定术语冲突、明显不符合目标语言习惯或不适合当前教学语境的形式。

### keep

必须保持原样的技术标识，例如：

- Go 标识符；
- 包名；
- 命令；
- 路径。

## 4. Language-specific terminology process

每一种语言应独立制定术语。建议过程为：

```text
目标语言社区资料
        ↓
Terminology review
        ↓
TERMINOLOGY_GUIDE
        ↓
locales/<locale>/glossary.yaml
```

禁止采用以下方式：

```text
zh-CN glossary
        ↓
machine translate
        ↓
ja-JP glossary
```

不同语言社区可能存在不同的技术习惯、借词传统、读写方式和教学表达。将一个 locale 的术语表机器翻译为另一个 locale 的术语表，会丢失这些语言特有的判断，也可能制造看似一致但不自然或不准确的术语。

## 5. Updating terminology

新增或修改术语时，应按以下顺序判断：

1. 判断该术语是否语言无关；
2. 判断是否属于 locale 专属规则；
3. 更新对应 locale 的 `glossary.yaml`；
4. 如果术语变更影响自动校验规则，再同步 validator。

术语维护应保持 locale 隔离：一个语言的术语调整不应自动改变其他语言的 glossary。需要跨语言共享的内容，应先确认它确实是语言无关的技术标识或项目规则。
