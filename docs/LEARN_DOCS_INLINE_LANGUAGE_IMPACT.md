# Learn/Docs inline language parser 影响分析

历史 predecessor：`learn-docs-v1-qc-001`。本报告是诊断，不是 Generation、Snapshot 或 QC evidence。

正式 Unit source identity（index 2）：old=`a2e6551e186953fa9f90f87f8bff4bd814d38aaada95e398d239b39d6417d8fb`；new=`40e961149f3934c6d9b544c3deb0c41f878d9f0ae88bfc5db58872ab2b3de07f`。

## 2 — _content/doc/tutorial/add-a-test.html

- old Unit ID: `222d561fdcfcc7b1bae13ce24757be32dee3cf49602463c5f82525d806fc358f`
- new Unit ID: `a58b321bd863b55022f34c8b3f5b8fb48b6fa77ec9e8b252a88095df48a2d66f`
- old input SHA: `d0e1f53792f2634adf6ba51a69446010ba28b1512cd12a404a3e35b79532aac3`
- new input SHA: `736803d99cd5be72d58fbaaf184e695686e96a2c8dc58faad7e68a80cae98f5b`
- source SHA（bytes 不变）: `d5ea155f25736df5051fddc35df81af3600b57700972d8fde4c344345c9a0ca0`
- old structure identity: `e93af80f8252a37ef6222cc90f1de54b5fb329080885bf3665e02ad374a07ea5`
- new structure identity: `028a5cf4fb99310328727814ec4138e39b12fbcf5f14a464f580ba2c48573c1a`
- internal slots: 20 → 23；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：直接自然语言 run 包含完整 inline protected element，此前被整体排除；现在作为语言槽位和 inline tokens。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 3）：old=`6aae55c1317a8a579ae16b266228ec38c488e08e907cc8e739a58436997d80fd`；new=`0a780645e0cb0432144c801b95299466ca352573be208e104403e6e221ccb3b1`。

## 3 — _content/doc/tutorial/call-module-code.html

- old Unit ID: `a8a7bcccd2947c72dafc56ad38debb7cd45cf3a9787b3f7389b853009dac2326`
- new Unit ID: `c85ef7834d7badd0decefc11c5e655fe930e2d9c2059f72c8b3b4329514bc129`
- old input SHA: `59f46bbbcd869cc7a77a7ed0621249266bba46f58abf48a1a4febd9503ee8fda`
- new input SHA: `0480db27d3558cb90dd8e8c19df91f6456a3158499c5be6dee16e904b6d92ca1`
- source SHA（bytes 不变）: `e15097929f143836d6b0ba6f7ed56b1a6f61fec64d8e4fbab78407af439a2460`
- old structure identity: `c3239ce98e35663abb04159b8fa71343fbe92c4fec64e9950deec8460e694322`
- new structure identity: `8008e161924ed6b278765300dec5bbc7eeb05375f10d79ccfe3fd0b9801e8d08`
- internal slots: 27 → 31；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：直接自然语言 run 包含完整 inline protected element，此前被整体排除；现在作为语言槽位和 inline tokens。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 4）：old=`f85f6c374196fc4d830f8d6f4e3c3953a44ee7eb3d23b1acc5a5e47dd1d55d07`；new=`11db7a97a9f33ab1edf46e8663dc5e72d84ce2b23cf53b079b4f7cb6fc43716d`。

## 4 — _content/doc/tutorial/compile-install.html

- old Unit ID: `9682704b58c5523b5e4f4d7734274f63395e43b359a3256129f4c1d2a626b95d`
- new Unit ID: `ac96ff288b73304d3c61dcee7965ec6e2ff3da2d71518b9b952b44af38ec8a22`
- old input SHA: `72cd2c8b05826366589e871231a593e512a31ec399f0aa9779e0c59b5be61130`
- new input SHA: `4242c9e70ef87102c27cf6bba7e3b2a05add314e5d7668a5ea7b9121f1961973`
- source SHA（bytes 不变）: `207de8898ceffd8fe454d8d94ffabe348b1579dd0c5dbd31cd5625f39291d3a0`
- old structure identity: `211a3dbaa98aa7522f6fc1ec9ef7cecdbb7af9ce82681768661cb432615e2b59`
- new structure identity: `415b0e472ca5eaa499d13bf36647c13eca8fe651c70b0fd6c0542b8d63e2b5b7`
- internal slots: 24 → 26；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：直接自然语言 run 包含完整 inline protected element，此前被整体排除；现在作为语言槽位和 inline tokens。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 5）：old=`629da23294eed6547300c10fb1ba655e18282f345a9975af2b0094d9ae409c52`；new=`c568ce226407de82fb9e64cde2501c0ad1dc207a11b169a37bb001a53f899bd5`。

## 5 — _content/doc/tutorial/create-module.html

- old Unit ID: `8c52c070c06dc5808cb82c1743e20a78424aefc2d595eceb309c234cb9dee0dd`
- new Unit ID: `34c60abd7d1eeab6ffafb1ba82914ecc8eeb5aaa0e20e8e3055f65b234e506c2`
- old input SHA: `1d627310d254c2ea1390f356d2acb17a345fdd072a96e7274cc4bec52b16a990`
- new input SHA: `92c0dd33222b533eb88713d19aa6e7d265fb7c32760896ae44288d18602015f1`
- source SHA（bytes 不变）: `e86bbbcbb07dbd6aa9a5e3e00ddf7525888f8f4298299818b9c7cf7b2407d144`
- old structure identity: `e61d31d358edc25394009006196821b2ca14a0d97a92118060053e79b0c97cc9`
- new structure identity: `51ccb31f0b219669adf6c3e8c85dc06876f3997a9c8d12e7184dc0327e7fa2d7`
- internal slots: 38 → 43；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：直接自然语言 run 包含完整 inline protected element，此前被整体排除；现在作为语言槽位和 inline tokens。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 9）：old=`b1ffca5bf89933eeea33bdf56932dbd20732eb9e166eb6f9d258ac3e49315a2e`；new=`fab91fa3a5b74cbe47944b8e7e40569b5019aac633a2fc06f8b19abf9eea8f2d`。

## 9 — _content/doc/tutorial/getting-started.html

- old Unit ID: `ab4a5a47c4ec4462c2c48641fde33c7e970bf4513500cec239006ec38cd241af`
- new Unit ID: `dffdb70f0db3cd3fc7a18588bf472129cc62957ec7fbf87482f9ee8ad1f2f625`
- old input SHA: `498aec94c501e71047a465b1ba70281d58dc31c6a1a2c7b9026b3e3854b4e150`
- new input SHA: `d6c4a250b8a95f2a965150e3cb96f85f2654b22c02dec1e8c673f12691e918c8`
- source SHA（bytes 不变）: `b4794678dc335ab850ceb21bc7b3a9896a401717d906a5242758803b444d45bd`
- old structure identity: `aa6a772f8279289eae67944eb2cae539f98c519cb5410d4c39fb0c10cdf90116`
- new structure identity: `b8067a38222227fdad4184876247c021c3e17debfe1e7b42c5a5b09893be5b5c`
- internal slots: 50 → 51；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：直接自然语言 run 包含完整 inline protected element，此前被整体排除；现在作为语言槽位和 inline tokens。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 12）：old=`7aa5983cc3b529faa9defef0dbb1efab727b60ac8f7dfd3de419da9d7759e1d5`；new=`7098644b0e67a5a2b3852597f20a1f05d8924760b5b82d648232f124d8d34ec9`。

## 12 — _content/doc/tutorial/greetings-multiple-people.html

- old Unit ID: `a0b2705ae3217d1b86dda6e2c11e062857b0f4d5a876bc10c8e9c76fb57bf830`
- new Unit ID: `27d919dd972c2ef0dc7611e7027723e33323beabbd386307454ee65e4f806d78`
- old input SHA: `334894d75b05a9e515662262a9cbbaa8291e691ea612752c42074b110341e6e5`
- new input SHA: `08853c2fce0a47c0f3973ca60bfc6f60eae39930e068e1ffee567b5d0591445e`
- source SHA（bytes 不变）: `8119036dcc03148e6bdadf5c5e5f39a1b01d1175b4875ff04ce9591636e3e84b`
- old structure identity: `429a63f6297160b989b7e2f2ebecd8a9dd392f6e88b43b55d29e45132827bce5`
- new structure identity: `f414840d8591e080718082687b445da952aa0c03550d671f1832a32d795d6cd5`
- internal slots: 42 → 43；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：直接自然语言 run 包含完整 inline protected element，此前被整体排除；现在作为语言槽位和 inline tokens。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 13）：old=`763f50cd604a722b8b912d701b18e4a1dcddea3a872d496dfa969d515edfc7df`；new=`636cddab39230c82453f3eb9bf69aeb685dedb87bea8f0d33423c322fb49de03`。

## 13 — _content/doc/tutorial/handle-errors.html

- old Unit ID: `3028d05df0c02c236b71e88f5ef1ef6c35a75e9b211922145fcf7af864e98b46`
- new Unit ID: `c5df58cba15efb86836ec12fba525999556a21919d838b972a44b00e02032b1b`
- old input SHA: `f91b9ff44706bf73f2ffc2746cdedbe0112ed2d3bf1df45dddda44d0432d2ab7`
- new input SHA: `f9e0573536be24ce0208f73db1f0640095391ca1be2cea4ffb39de6fbda7d682`
- source SHA（bytes 不变）: `10e61d7f50095845bd2e313add841500b189bb7ddda6c3b7fee248f8cdc53a89`
- old structure identity: `5aca6a54e1d10d84aab1a599dc324b49e1319f53cb610d4b0b0f282c7b22821b`
- new structure identity: `c128e7ae0f44ac5c93bd973f6961dc3bf89941136494be2d920ff506bd897e2f`
- internal slots: 35 → 37；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：直接自然语言 run 包含完整 inline protected element，此前被整体排除；现在作为语言槽位和 inline tokens。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

## 精确 scope

changed=7；unchanged=42。

- identity_changed / Generation + validation + QC: `[2 3 4 5 9 12 13]`
- unchanged A carry-forward: `[14 16 20 21 23 27 28 34 45 48]`
- unchanged B/C 可按原 finding revision: `[1 6 7 8 10 11 15 17 18 19 22 24 25 26 29 30 31 32 33 35 36 37 38 39 40 41 42 43 44 46 47 49]`

unchanged 的 path/source bytes/internal slots/protected raw/roles/formal input/Unit ID/source identity/structure identity 全部逐字段一致；candidate 字节、restore 和当前 validator 均通过。
