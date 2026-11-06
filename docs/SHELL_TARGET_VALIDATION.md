# Shell 冻结 source 与 target validation

`ExtractShellLanguageSource` 只负责 source discovery：保守的 English teaching-comment eligibility、`hasLanguage` 与 source protector classification 建立冻结 slots、encodings、machine bindings 和非语言 bytes。target 不再重新 discovery，也不比较 target lexical protection classification。

reconstruction 使用这些冻结 source slots。独立 candidate validation 以 source gaps 的完整锚定匹配取得 target slot，按冻结 encoding 解码、重新绑定 exact protected raw identity，再用既有 token exact-set/pair/template/link、slot syntax 与 encoding validator 校验，并要求 re-encoding 与 candidate slot bytes 一致。Go code、模板控制流、HTML 标签/属性、URL、class/id 与其他非语言 gaps 必须逐字节保持；新增结构、UTF-8 错误、quote/YAML/Go comment framing、缺少或重复 machine identity、ASCII identity extension 均 fail closed。中文、日文或其他语言的自然标点/语序不改变 frozen binding。

Glossary visibility 使用同一冻结 slot capture 和 protection，包含已翻译 teaching comment，排除 machine bytes。source roundtrip 保留原 encoding/whitespace。

既有 source adoption 的 reconstruction validator 代码 SHA 是历史实现身份，不是当前语言质量或 source discovery authority。reader 在完整 source inventory、全部冻结 representation 字段、parser/corpus 与其他 runtime dependency 仍严格一致时保留原 validator dependency，既有 adoption bytes/identity 不重写。新 adoption 仍记录当时当前代码 SHA。仅声明的 validator dependency 可以复用历史身份；其他依赖继续 current-check。当前 candidate 始终执行当前 canonical validator，adapter re-attestation 继续绑定当前代码。这不授权 activation、Preview、Production 或语言 acceptance。

validator-fix recovery 复用 [Revalidate](RETRANSLATION_RUNBOOK.md#4-revalidatevalidator-修复后重验或恢复同一-attempt)，保存旧 failure 的 exact bytes，同一 raw response/attempt 重新 restore/validate，不新增 shell receipt 或生命周期。
