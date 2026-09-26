# kn-IN 首次正式 Glossary Review
review-id: 20260926-kn-IN-glossary-001
reviewer: independent ChatGPT GPT-5.6 Sol High Reviewer
decision: passed
blocking findings: 0
locale / html_lang: kn-IN / kn-IN (ಕನ್ನಡ)
reviewer ZIP SHA-256: 8c38a4af7a37e64cd50004312acf4ac25f5753d219cdb026fb0ac938aa84b531
glossary path: locales/kn-IN/glossary.yaml
glossary SHA-256: 6599b51b6877ea6725d649a5c2e2c592dcf035654a9e07fa0200374c6fbb03cb
input identity: a5a4311bee4aa01ee77076ba83d6eba7a2501bb6c1bd42a2ee1f54a203df364b

## 完整输入和审核范围

经 SHA-256 与大小核对：manifest 所列 13/13 文件；7 份 authority、正式 glossary、locale identity、English UI、Page/Example catalogs 和 122/122 完整 English/source context。103/103 Page、19/19 Example；stable indexes 连续且无重复。已逐项审核 mandatory 55/55、preferred 56/56、forbidden 4/4、keep 15/15，全部术语规则本身无 blocking finding。

## 概念边界与课程源文证据

- 全站：welcome/1、welcome/4、welcome/5 与 English UI 的 tour.title、editor.run、editor.format、editor.reset、editor.kill。A Tour of Go 与 tour 的译名衔接；Go Playground 作为官方技术服务名保留英文。Run / Format / Reset / Kill 在 UI 中指向明确不同的动作，其中 Kill 的本地化表示停止当前执行，不误作暂停功能。
- 基础：basics/1、basics/3–4、basics/7、basics/11–14，methods/1–16，generics/1–2。package / function / method / receiver、pointer receiver / value receiver、basic type / underlying type / concrete type、type conversion / inference / assertion / switch、interface type / value、type parameter / constraint 均有可辨认的独立术语，没有混为同一概念。
- 集合与内存：moretypes/6–23，特别是 moretypes/7–8、moretypes/11、moretypes/15。array 与 slice 区分，underlying array 和 backing array 在该课程中指向同一底层存储数组；length 与 capacity、map 与 channel、代码身份与自然语言术语的边界一致。
- 并发：concurrency/1–10 及 concurrency/channels.go、concurrency/exercise-web-crawler.go、concurrency/mutex-counter.go。goroutine 维持 Go 原生名称；channel 和 pipeline 各用不同的自然语言概念；mutex 与 mutual exclusion 分开；concurrency 不被解释为必然在多核上同时执行，parallelism 是并行执行，两者译法明确区分。
- keep：gofmt、GOPATH、nil、URL、ASCII、CPU、UTC 在正式课程中可核；GitHub、Yongye、USDC、USDT、TRC20 见正式 English UI。未把 map / slice / channel 等可翻译教学概念错误放入 keep。
- forbidden：两个 goroutine 音译形式、两个 Go Playground 非规范形式仅作术语规范约束，不全局禁止普通卡纳达语词汇。
- 源文多义词：welcome/4 的 Go offline 与 concurrency/11 的 Where to Go from here 需要在后续逐 TU QC 保持标题语义；现有仓库代码对后者的普通动词 Go 有定向处理。此为后续候选核验提示，不是 glossary blocking finding。
- 代码保护：Go keywords、API identifiers、Go 代码、路径、链接目标及 present directive 继续由结构保护承担；自然语言术语由 glossary 决策承担。无已证实的语义反转、重复冲突、不当全局 forbidden 或非必要 keep 扩张。

## 全量逐项术语审核台账

PASS 表示该条术语决策本身没有 blocking finding，不代表后续译文通过 QC。

### mandatory

| English | 当前 target | 结果 |
| --- | --- | --- |
| A Tour of Go | Go ಕಲಿಕಾ ಪ್ರವಾಸ | PASS |
| Run | ಚಲಾಯಿಸಿ | PASS |
| Format | ಸ್ವರೂಪಗೊಳಿಸಿ | PASS |
| Reset | ಮರುಹೊಂದಿಸಿ | PASS |
| Kill | ನಿಲ್ಲಿಸಿ | PASS |
| module | ಘಟಕ | PASS |
| lesson | ಪಾಠ | PASS |
| exercise | ಅಭ್ಯಾಸ | PASS |
| package | ಪ್ಯಾಕೇಜ್ | PASS |
| import | ಇಂಪೋರ್ಟ್ | PASS |
| exported name | ಎಕ್ಸ್‌ಪೋರ್ಟ್ ಮಾಡಿದ ಹೆಸರು | PASS |
| unexported name | ಎಕ್ಸ್‌ಪೋರ್ಟ್ ಮಾಡದ ಹೆಸರು | PASS |
| function | ಫಂಕ್ಷನ್ | PASS |
| method | ಮೆಥಡ್ | PASS |
| parameter | ಪ್ಯಾರಾಮೀಟರ್ | PASS |
| argument | ಆರ್ಗ್ಯುಮೆಂಟ್ | PASS |
| receiver | ರಿಸೀವರ್ | PASS |
| pointer | ಪಾಯಿಂಟರ್ | PASS |
| pointer receiver | ಪಾಯಿಂಟರ್ ರಿಸೀವರ್ | PASS |
| value receiver | ಮೌಲ್ಯ ರಿಸೀವರ್ | PASS |
| variable | ಚರ | PASS |
| constant | ಸ್ಥಿರಾಂಕ | PASS |
| type | ಟೈಪ್ | PASS |
| basic type | ಮೂಲ ಟೈಪ್ | PASS |
| concrete type | ನಿರ್ದಿಷ್ಟ ಟೈಪ್ | PASS |
| underlying type | ಆಧಾರಭೂತ ಟೈಪ್ | PASS |
| type conversion | ಟೈಪ್ ಪರಿವರ್ತನೆ | PASS |
| type inference | ಟೈಪ್ ನಿರ್ಣಯ | PASS |
| type assertion | ಟೈಪ್ ಅಸರ್ಷನ್ | PASS |
| type switch | ಟೈಪ್ ಸ್ವಿಚ್ | PASS |
| type parameter | ಟೈಪ್ ಪ್ಯಾರಾಮೀಟರ್ | PASS |
| constraint | ನಿರ್ಬಂಧ | PASS |
| interface | ಇಂಟರ್‌ಫೇಸ್ | PASS |
| interface type | ಇಂಟರ್‌ಫೇಸ್ ಟೈಪ್ | PASS |
| interface value | ಇಂಟರ್‌ಫೇಸ್ ಮೌಲ್ಯ | PASS |
| empty interface | ಖಾಲಿ ಇಂಟರ್‌ಫೇಸ್ | PASS |
| array | ಅರೇ | PASS |
| slice | ಸ್ಲೈಸ್ | PASS |
| underlying array | ಆಧಾರ ಅರೇ | PASS |
| backing array | ಆಧಾರ ಅರೇ | PASS |
| map | ಮ್ಯಾಪ್ | PASS |
| channel | ಚಾನೆಲ್ | PASS |
| buffered channel | ಬಫರ್ ಹೊಂದಿದ ಚಾನೆಲ್ | PASS |
| unbuffered channel | ಬಫರ್ ಇಲ್ಲದ ಚಾನೆಲ್ | PASS |
| pipeline | ಪೈಪ್‌ಲೈನ್ | PASS |
| mutex | ಮ್ಯೂಟೆಕ್ಸ್ | PASS |
| mutual exclusion | ಪರಸ್ಪರ ಹೊರಗಿಡುವಿಕೆ | PASS |
| concurrency | ಸಹವರ್ತಿ ಕಾರ್ಯನಿರ್ವಹಣೆ | PASS |
| parallelism | ಸಮಾನಾಂತರ ಕಾರ್ಯನಿರ್ವಹಣೆ | PASS |
| zero value | ಶೂನ್ಯ ಮೌಲ್ಯ | PASS |
| closure | ಕ್ಲೋಶರ್ | PASS |
| generic type | ಜೆನೆರಿಕ್ ಟೈಪ್ | PASS |
| generic function | ಜೆನೆರಿಕ್ ಫಂಕ್ಷನ್ | PASS |
| length | ಉದ್ದ | PASS |
| capacity | ಸಾಮರ್ಥ್ಯ | PASS |

### preferred

| English | 当前 target | 结果 |
| --- | --- | --- |
| tour | ಕಲಿಕಾ ಪ್ರವಾಸ | PASS |
| programming language | ಪ್ರೋಗ್ರಾಮಿಂಗ್ ಭಾಷೆ | PASS |
| source code | ಮೂಲ ಕೋಡ್ | PASS |
| documentation | ದಸ್ತಾವೇಜು | PASS |
| standard library | ಪ್ರಮಾಣಿತ ಲೈಬ್ರರಿ | PASS |
| syntax | ಸಿಂಟ್ಯಾಕ್ಸ್ | PASS |
| syntax highlighting | ಸಿಂಟ್ಯಾಕ್ಸ್ ಹೈಲೈಟಿಂಗ್ | PASS |
| compiler | ಕಂಪೈಲರ್ | PASS |
| compile | ಕಂಪೈಲ್ ಮಾಡು | PASS |
| declaration | ಘೋಷಣೆ | PASS |
| initialization | ಆರಂಭಿಕೀಕರಣ | PASS |
| initializer | ಆರಂಭಿಕ ಮೌಲ್ಯ | PASS |
| assignment | ಮೌಲ್ಯ ನಿಯೋಜನೆ | PASS |
| short variable declaration | ಸಂಕ್ಷಿಪ್ತ ಚರ ಘೋಷಣೆ | PASS |
| return value | ಹಿಂತಿರುಗುವ ಮೌಲ್ಯ | PASS |
| named return value | ಹೆಸರಿಸಲಾದ ಹಿಂತಿರುಗುವ ಮೌಲ್ಯ | PASS |
| naked return | ಆರ್ಗ್ಯುಮೆಂಟ್ ಇಲ್ಲದ return | PASS |
| boolean | ಬೂಲಿಯನ್ | PASS |
| integer | ಪೂರ್ಣಾಂಕ | PASS |
| floating-point | ಫ್ಲೋಟಿಂಗ್ ಪಾಯಿಂಟ್ | PASS |
| complex number | ಸಂಕೀರ್ಣ ಸಂಖ್ಯೆ | PASS |
| string | ಸ್ಟ್ರಿಂಗ್ | PASS |
| struct | ಸ್ಟ್ರಕ್ಟ್ | PASS |
| struct field | ಸ್ಟ್ರಕ್ಟ್ ಫೀಲ್ಡ್ | PASS |
| struct literal | ಸ್ಟ್ರಕ್ಟ್ ಲಿಟರಲ್ | PASS |
| slice literal | ಸ್ಲೈಸ್ ಲಿಟರಲ್ | PASS |
| map literal | ಮ್ಯಾಪ್ ಲಿಟರಲ್ | PASS |
| field | ಫೀಲ್ಡ್ | PASS |
| index | ಇಂಡೆಕ್ಸ್ | PASS |
| low bound | ಕೆಳಗಿನ ಮಿತಿ | PASS |
| high bound | ಮೇಲಿನ ಮಿತಿ | PASS |
| half-open range | ಕೊನೆಯ ಮೌಲ್ಯವನ್ನು ಹೊರತುಪಡಿಸುವ ಶ್ರೇಣಿ | PASS |
| iteration | ಪುನರಾವರ್ತನೆ | PASS |
| loop | ಲೂಪ್ | PASS |
| scope | ವ್ಯಾಪ್ತಿ | PASS |
| operator | ಆಪರೇಟರ್ | PASS |
| dereferencing | ಪಾಯಿಂಟರ್ ಸೂಚಿಸಿದ ಮೌಲ್ಯವನ್ನು ಪಡೆಯುವುದು | PASS |
| deferred call | ಮುಂದೂಡಿದ ಫಂಕ್ಷನ್ ಕರೆ | PASS |
| function value | ಫಂಕ್ಷನ್ ಮೌಲ್ಯ | PASS |
| method signature | ಮೆಥಡ್ ಸಿಗ್ನೇಚರ್ | PASS |
| implementation | ಅನುಷ್ಠಾನ | PASS |
| generic | ಜೆನೆರಿಕ್ | PASS |
| comparable | ಹೋಲಿಸಬಹುದಾದ | PASS |
| linked list | ಲಿಂಕ್‌ಡ್ ಲಿಸ್ಟ್ | PASS |
| binary tree | ಬೈನರಿ ಟ್ರೀ | PASS |
| channel operator | ಚಾನೆಲ್ ಆಪರೇಟರ್ | PASS |
| buffer | ಬಫರ್ | PASS |
| synchronization | ಸಮನ್ವಯ | PASS |
| thread | ಥ್ರೆಡ್ | PASS |
| lightweight thread | ಹಗುರವಾದ ಥ್ರೆಡ್ | PASS |
| runtime | ರನ್‌ಟೈಮ್ | PASS |
| remote server | ದೂರಸ್ಥ ಸರ್ವರ್ | PASS |
| sandbox | ಪ್ರತ್ಯೇಕಿತ ಕಾರ್ಯಪರಿಸರ | PASS |
| error | ದೋಷ | PASS |
| panic | ಪ್ಯಾನಿಕ್ | PASS |
| web crawler | ವೆಬ್ ಕ್ರಾಲರ್ | PASS |

### forbidden

| 当前术语 | 结果 |
| --- | --- |
| ಗೋರೂಟೀನ್ | PASS |
| ಗೋ ರೂಟೀನ್ | PASS |
| Go ಪ್ಲೇಗ್ರೌಂಡ್ | PASS |
| Go Play Ground | PASS |

### keep

| 当前术语 | 结果 |
| --- | --- |
| Go | PASS |
| Go Playground | PASS |
| goroutine | PASS |
| gofmt | PASS |
| GOPATH | PASS |
| nil | PASS |
| URL | PASS |
| ASCII | PASS |
| CPU | PASS |
| UTC | PASS |
| GitHub | PASS |
| Yongye | PASS |
| USDC | PASS |
| USDT | PASS |
| TRC20 | PASS |

## English/source context 全量覆盖清单

下列 122 个 stable index 的完整 source 字段均来自已校验的 reviewer ZIP；每项 source_sha256 位于 glossary-review.json。

### page (103)

- 1 welcome/1
- 2 welcome/2
- 3 welcome/4
- 4 welcome/5
- 5 welcome/3
- 6 basics/1
- 7 basics/2
- 8 basics/3
- 9 basics/4
- 10 basics/5
- 11 basics/6
- 12 basics/7
- 13 basics/8
- 14 basics/9
- 15 basics/10
- 16 basics/11
- 17 basics/12
- 18 basics/13
- 19 basics/14
- 20 basics/15
- 21 basics/16
- 22 basics/17
- 23 flowcontrol/1
- 24 flowcontrol/2
- 25 flowcontrol/3
- 26 flowcontrol/4
- 27 flowcontrol/5
- 28 flowcontrol/6
- 29 flowcontrol/7
- 30 flowcontrol/8
- 31 flowcontrol/9
- 32 flowcontrol/10
- 33 flowcontrol/11
- 34 flowcontrol/12
- 35 flowcontrol/13
- 36 flowcontrol/14
- 37 moretypes/1
- 38 moretypes/2
- 39 moretypes/3
- 40 moretypes/4
- 41 moretypes/5
- 42 moretypes/6
- 43 moretypes/7
- 44 moretypes/8
- 45 moretypes/9
- 46 moretypes/10
- 47 moretypes/11
- 48 moretypes/12
- 49 moretypes/13
- 50 moretypes/14
- 51 moretypes/15
- 52 moretypes/16
- 53 moretypes/17
- 54 moretypes/18
- 55 moretypes/19
- 56 moretypes/20
- 57 moretypes/21
- 58 moretypes/22
- 59 moretypes/23
- 60 moretypes/24
- 61 moretypes/25
- 62 moretypes/26
- 63 moretypes/27
- 64 methods/1
- 65 methods/2
- 66 methods/3
- 67 methods/4
- 68 methods/5
- 69 methods/6
- 70 methods/7
- 71 methods/8
- 72 methods/9
- 73 methods/10
- 74 methods/11
- 75 methods/12
- 76 methods/13
- 77 methods/14
- 78 methods/15
- 79 methods/16
- 80 methods/17
- 81 methods/18
- 82 methods/19
- 83 methods/20
- 84 methods/21
- 85 methods/22
- 86 methods/23
- 87 methods/24
- 88 methods/25
- 89 methods/26
- 90 generics/1
- 91 generics/2
- 92 generics/3
- 93 concurrency/1
- 94 concurrency/2
- 95 concurrency/3
- 96 concurrency/4
- 97 concurrency/5
- 98 concurrency/6
- 99 concurrency/7
- 100 concurrency/8
- 101 concurrency/9
- 102 concurrency/10
- 103 concurrency/11

### example (19)

- 104 example:basics/numeric-constants.go
- 105 example:basics/type-inference.go
- 106 example:concurrency/channels.go
- 107 example:concurrency/exercise-equivalent-binary-trees.go
- 108 example:concurrency/exercise-web-crawler.go
- 109 example:concurrency/mutex-counter.go
- 110 example:flowcontrol/if-and-else.go
- 111 example:generics/index.go
- 112 example:generics/list.go
- 113 example:methods/exercise-reader.go
- 114 example:methods/exercise-stringer.go
- 115 example:methods/interfaces-are-satisfied-implicitly.go
- 116 example:methods/interfaces.go
- 117 example:moretypes/append.go
- 118 example:moretypes/exercise-fibonacci-closure.go
- 119 example:moretypes/pointers.go
- 120 example:moretypes/slice-len-cap.go
- 121 example:moretypes/slices-of-slice.go
- 122 example:moretypes/struct-literals.go

## 正式结论

decision=passed；blocking findings=0。记录前必须再次运行 reviewer-bundle-check；记录后 glossary-review check 必须达到 CURRENT PASS。本审核不批准 UI、TU、Course SEO 或 Production。
