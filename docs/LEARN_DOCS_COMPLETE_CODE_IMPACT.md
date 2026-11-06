# Learn/Docs complete-code teaching-comment parser 影响分析

历史 predecessor：`learn-docs-v1-qc-002`。本报告是诊断，不是 Generation、Snapshot 或 QC evidence。

正式 Unit source identity（index 2）：old=`40e961149f3934c6d9b544c3deb0c41f878d9f0ae88bfc5db58872ab2b3de07f`；new=`bce0c39ba78a281d00e4d434d5dbb8b1b12f1d5342d32fe43b142265bbb91bd7`。

## 2 — _content/doc/tutorial/add-a-test.html

- old Unit ID: `a58b321bd863b55022f34c8b3f5b8fb48b6fa77ec9e8b252a88095df48a2d66f`
- new Unit ID: `e993fec2454ba4fe82ff89cc33b82fba5c836acfeb9bbe23ce160fdc07e12ece`
- old input SHA: `736803d99cd5be72d58fbaaf184e695686e96a2c8dc58faad7e68a80cae98f5b`
- new input SHA: `6f7abdba1277d0871cd038182d2b556c9f51ce66705326ae5fa1894b2627e842`
- source SHA（bytes 不变）: `d5ea155f25736df5051fddc35df81af3600b57700972d8fde4c344345c9a0ca0`
- old structure identity: `028a5cf4fb99310328727814ec4138e39b12fbcf5f14a464f580ba2c48573c1a`
- new structure identity: `86600037382553b18434eab6e116ca7040da6ac130974141e76bb13f91c0b407`
- internal slots: 23 → 27；protected raw/roles 与 formal input 变化；原评级 B。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 3）：old=`0a780645e0cb0432144c801b95299466ca352573be208e104403e6e221ccb3b1`；new=`ac90bc6802bde74946397fec791993f4c22ac2e87a29e943c4450261208e726a`。

## 3 — _content/doc/tutorial/call-module-code.html

- old Unit ID: `c85ef7834d7badd0decefc11c5e655fe930e2d9c2059f72c8b3b4329514bc129`
- new Unit ID: `d6a0464cbe1d4843c82810dfac96d42c621065f81c56c1f63fb8af589abcc748`
- old input SHA: `0480db27d3558cb90dd8e8c19df91f6456a3158499c5be6dee16e904b6d92ca1`
- new input SHA: `bd05757cb65ccb6431423b301fdb942c61517bc8f470d4e39f6590d04a15704e`
- source SHA（bytes 不变）: `e15097929f143836d6b0ba6f7ed56b1a6f61fec64d8e4fbab78407af439a2460`
- old structure identity: `8008e161924ed6b278765300dec5bbc7eeb05375f10d79ccfe3fd0b9801e8d08`
- new structure identity: `1e059307a59e801f2711da38ba850ed16287d3c1c2ba10949608cd6fcef145f0`
- internal slots: 31 → 32；protected raw/roles 与 formal input 变化；原评级 B。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 5）：old=`c568ce226407de82fb9e64cde2501c0ad1dc207a11b169a37bb001a53f899bd5`；new=`9de624015ca4e03d4ffe7bf26c80592fb4bc94e84729dfba5e0f6bd610bfa98a`。

## 5 — _content/doc/tutorial/create-module.html

- old Unit ID: `34c60abd7d1eeab6ffafb1ba82914ecc8eeb5aaa0e20e8e3055f65b234e506c2`
- new Unit ID: `2cf2207a4cf54f3b5ef344d2c02198f70a6b513a4ccf323424dea35e2e338d8d`
- old input SHA: `92c0dd33222b533eb88713d19aa6e7d265fb7c32760896ae44288d18602015f1`
- new input SHA: `e9edb86f2c214a5326263327ad9193152136fe307f524218c7066ea8f3b71a47`
- source SHA（bytes 不变）: `e86bbbcbb07dbd6aa9a5e3e00ddf7525888f8f4298299818b9c7cf7b2407d144`
- old structure identity: `51ccb31f0b219669adf6c3e8c85dc06876f3997a9c8d12e7184dc0327e7fa2d7`
- new structure identity: `2eefa34592b01de1bea6d1f6f49a73f0406f8516f6f2cc7d06e8a0c74456778c`
- internal slots: 43 → 45；protected raw/roles 与 formal input 变化；原评级 B。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 6）：old=`7c6294f339beb1305110532625c54dccabbded56e55d2a32307983d371399536`；new=`aebbeb5fe399ca991987a3582f7115cf62a8035fe039b8cd9eefc854c71a5fca`。

## 6 — _content/doc/tutorial/database-access.md

- old Unit ID: `6ab8c22a6bd6dd7281d49d00deea62bce81f4673d9e385e4f316aded236d758a`
- new Unit ID: `51f3462b599aba4267be310f3954e51e7d0c4fd4230460ac0eb4eb64c9ee5628`
- old input SHA: `7ef904e8bba0ad9b8704b0976f19bb8314430e14021956ed87441fdf71eb3128`
- new input SHA: `9ffa3276911d07c0cb9b74e6fe09a3da6815e040dc59ea84570ce2120f02408d`
- source SHA（bytes 不变）: `006b82bedadb9fbf37ab24e3e7e09ae8f86abc8870d0b1df45e67b4d6ea337e1`
- old structure identity: `c17b7b82ff3b9fdaf7b61bb06e2218b249a3d7114d7d1a71d9a35fd101449094`
- new structure identity: `fa3fedc0efd8bc54eb777a12fa201d5b7700f6b438aefdb58c6800f2af162b8e`
- internal slots: 170 → 177；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 7）：old=`28612f9e1dad6b58803d83ef008618523d6430474434ba2ff947381ee0801ede`；new=`d84b36dca4546261bc9af3f5afa4d3d85a14e0325a945adae6f56f5dd2185378`。

## 7 — _content/doc/tutorial/fuzz.md

- old Unit ID: `7e2b101749d045df4cf9be3f50225d400bb41aa5e5b433d3c071fed8f8797451`
- new Unit ID: `483944fc286965a89e6977332a877957cb15e3ba1171361a517d862bdc50f858`
- old input SHA: `7433f54512dc242f3549a4277f9c7b432ee4028729c28ad4dded48541557265d`
- new input SHA: `22474512cc3b540cd1d66ad65c114487904cb88b502b58494fd54180d1a2a560`
- source SHA（bytes 不变）: `91f902e95484e8320b35801d1bb22c12b90f08983a3a269ca373cce96c8cc903`
- old structure identity: `e0b08b7a1bb25d362e0e3a17fe6918f4de3450766cce4559179efd48a0c83b2d`
- new structure identity: `a2784cc9cc5bc2b3a8ba5b78dc3136f40817a6fbd5d3017adadb527b8d24649f`
- internal slots: 153 → 152；protected raw/roles 与 formal input 变化；原评级 B。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 8）：old=`e8f32409376b6c7662e102643717f26a8c9a282b3e1c8c39e94c6fc445c0a22a`；new=`f7a7b4c3d594ff05f0ae4e9def3d038c2f99039561bd8f0598f66032fd56d0e9`。

## 8 — _content/doc/tutorial/generics.md

- old Unit ID: `e6404a75442381a01ddea7603ec5b9d60058392f9ad7cb8314e34fbf75742e2f`
- new Unit ID: `e00dd8869bf8155daea7791025b56a0b1c3ad8e691ac7e9ff3347d0b8db023cd`
- old input SHA: `791cf837cebc46f1bd96e271a25d3a28a4561335b66c6ce579e84e68b7ffe934`
- new input SHA: `c5f9b65f36925a1668b66b8cbaa369bb1daa4573c462ec9ac1bd4d9ea3941ba6`
- source SHA（bytes 不变）: `c15398a80cb69b1b0d486d14bb35ccb2a807a4fbcac6b7bd5319a997513aa67f`
- old structure identity: `a3d924879377f9a410c954d71ffc4cb6ecba0e4f4cb6bf329d92c3c127874de3`
- new structure identity: `65e69a85efa0712f45a4479432abbeff378a5e2b7bf2d7d0c5e7feb7695f58bc`
- internal slots: 121 → 126；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 10）：old=`33e09da539fdc5848366920795b9c5397fcec0854e095ae19d698c44eb9e310d`；new=`2656e66cec4ba1af642469b26b6ab1e5b350f12723672a24cf0eb58fabc9a2a3`。

## 10 — _content/doc/tutorial/govulncheck-ide.md

- old Unit ID: `447efb9972318d8da5dd586f4603750174791f5cbc5f48564d77b862160edeb3`
- new Unit ID: `d8f9a1931eb5cbf1f5eae5df9648f25b122d758a04472a107fbcf4579a0c60fb`
- old input SHA: `91da270bbda11110439a20caea3f07bd543397725ce87049623cca8e9a4a3e0b`
- new input SHA: `68543e61f3d4d1fe6e13aa1d01abef45799c428d6d59f0b2ca49273e51883d4a`
- source SHA（bytes 不变）: `b0173a57e407fe2937c85ed434dee7037eee0a9d91f495cb4d43a4e6f7dabd35`
- old structure identity: `1cd1f83f3e27fea926e6160c5506a9bf478f9b4ce76baa92360c7c093c1dc501`
- new structure identity: `6da828ed7abd81b2dd4f80a5a1cccb8133e447cf596bef81e95e1ccf345e2464`
- internal slots: 30 → 32；protected raw/roles 与 formal input 变化；原评级 B。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 12）：old=`7098644b0e67a5a2b3852597f20a1f05d8924760b5b82d648232f124d8d34ec9`；new=`5c7d7f6af9b257aae46639e035eaa5aa420026bce34f2fdc0228dddd4db79546`。

## 12 — _content/doc/tutorial/greetings-multiple-people.html

- old Unit ID: `27d919dd972c2ef0dc7611e7027723e33323beabbd386307454ee65e4f806d78`
- new Unit ID: `bceb0b950e9bf9ac6fd4187156327d95019229af6ec9d7bf00a8178edca3272e`
- old input SHA: `08853c2fce0a47c0f3973ca60bfc6f60eae39930e068e1ffee567b5d0591445e`
- new input SHA: `cc08457cc4ac8850e3770132fe45261f75a270348bba0e4153fad787d5967f2d`
- source SHA（bytes 不变）: `8119036dcc03148e6bdadf5c5e5f39a1b01d1175b4875ff04ce9591636e3e84b`
- old structure identity: `f414840d8591e080718082687b445da952aa0c03550d671f1832a32d795d6cd5`
- new structure identity: `0346b9e11101b367c5c575e5341b5e53e554f9e722febbd2b6e9ca48da8d3bad`
- internal slots: 43 → 39；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 13）：old=`636cddab39230c82453f3eb9bf69aeb685dedb87bea8f0d33423c322fb49de03`；new=`0e18c9eb6d8ac1fee2c970a14777ad482475d8735aa90dffd4617f578180f044`。

## 13 — _content/doc/tutorial/handle-errors.html

- old Unit ID: `c5df58cba15efb86836ec12fba525999556a21919d838b972a44b00e02032b1b`
- new Unit ID: `66264496c6d846fefd6419e37c58df366152af07523cdd1f81088c8839f0e5ec`
- old input SHA: `f9e0573536be24ce0208f73db1f0640095391ca1be2cea4ffb39de6fbda7d682`
- new input SHA: `811b1e940e44ef879cebb607ca190e9ae89a0afaf04e0ea034c59bc0ede7625c`
- source SHA（bytes 不变）: `10e61d7f50095845bd2e313add841500b189bb7ddda6c3b7fee248f8cdc53a89`
- old structure identity: `c128e7ae0f44ac5c93bd973f6961dc3bf89941136494be2d920ff506bd897e2f`
- new structure identity: `68d7aa7d4ab9f1639abdd055cce5dfd7851ccb249a4522a62409865eade90df6`
- internal slots: 37 → 34；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 15）：old=`f0608910c4829f9a6caf51c03166e39a8adacbd6104f6af563db5d8b550f7039`；new=`5d4a753cee6409cf577f8a39c4bc5da70681d3dd48953c25234d7160e38a80f0`。

## 15 — _content/doc/tutorial/json.md

- old Unit ID: `f5d71ccafb0457155eab86b0a06262d146276c2204859b5e635fdfe6648d331a`
- new Unit ID: `99824b13c26ffe219230ad0171e93d850003d16aa9f780c2f06c41bc46845cb3`
- old input SHA: `71b0fd09f0c257c611911e73e295b1666741cc81ebf97ece197452115321145c`
- new input SHA: `06554ff7462184a90b908d394519b5a5a5fc1a72d14177f61b9b583efcd15bab`
- source SHA（bytes 不变）: `9bb5bed860a71bdb4dc9776c9eff2e5f4a40f1ca9851851bfd1619c1793970be`
- old structure identity: `e992b1b5a52b5ccc6c78844bb62884c0bec478e1deb94b08a190b4e1a8d503b1`
- new structure identity: `48daca8ae73d5beeb74a4040662f4df703c7d21e848234ec85df993f253201b7`
- internal slots: 66 → 66；protected raw/roles 与 formal input 变化；原评级 B。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 17）：old=`a95942fced22321b308a86d5558ddf773a468b85bb8e66eb6574af7e221362a1`；new=`ffcd8f99c369457b1af659eec687f3fcde9eb1d88762f5624f8cf6ba421dd6ac`。

## 17 — _content/doc/tutorial/random-greeting.html

- old Unit ID: `164cff8e421c0f81385460d6097dd494a740c898314bf09358aa27a6f05f9119`
- new Unit ID: `0895c1f2bf020716e82386c738f1b63ff49194250cf4cf254f9cf346a453fc10`
- old input SHA: `3269753f1e9e29754f1671ae3a505bcff158056aeb3f565bd478c83966fafb4b`
- new input SHA: `ba08c8c4608c30b732839055043b90f66391f6bc74fd1d29647e791cafa196d3`
- source SHA（bytes 不变）: `9951ede236291e2db0beaa868159e101bc2983f9754c587c3ab3954fa26cdfbe`
- old structure identity: `b288317f5194156206706ccaa31f3a9748188d45caab213455154b05f9ad6dee`
- new structure identity: `12c4980d578eba20c9780bf38d5740a185d9fd41f98be53fd8ac42b9b320fae3`
- internal slots: 22 → 22；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 18）：old=`b0b53af6a15813286362835f762120f700cb749650af4fd57c8ffb3d8aa953d9`；new=`569dce0cb2ac2ad95c038c677aae0956427032641f2c8c4bd8217e9679a6d60d`。

## 18 — _content/doc/tutorial/web-service-gin.md

- old Unit ID: `2fecae0061d1f0a34632d79118a2bf1af10e8acba325bbde63e713e9c63eed47`
- new Unit ID: `da414f5ecd312a539f1ee2f38c60986db5fafccaf41699b454802234b2b5499f`
- old input SHA: `410547e9a1231bd630f30ca469bce6a492583210393dbbe0ee210a59c8608d77`
- new input SHA: `22ea938a3aa1a43f5220fa4547cb34a4e3d916421fcc822d272a57cae6fa0aa2`
- source SHA（bytes 不变）: `3342ff17a8b087b5d5a2a3c03d3fbcf4197eafd943c65a05c62c43df0cf33cf4`
- old structure identity: `12c004d4852cb6cd4ba8feb2f5b364b00d3ec67109825c033aa8bcf9220ba4fa`
- new structure identity: `42c7484cfef65ff7479ebb08feb3be11dfecd74faf239417b54844d72a185f71`
- internal slots: 155 → 156；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 19）：old=`38e81f96f8fd6c2fd8264ccf84d34ececfbbe820379ce07dff68fbbed1b39685`；new=`92684cde966219f90669edc5aae936fe68aaf3f06dcb8a6f59ce8e095e4367e6`。

## 19 — _content/doc/tutorial/workspaces.md

- old Unit ID: `34113d761416f892795b65fae1a768f78ab15224e9bb89825fb5762c91ffb8a9`
- new Unit ID: `a508d39fb57b2d122ca8536e038c6b090df53042c04d433b74d869bfbba2b9f1`
- old input SHA: `46cbaafa3a16d141e340ce342284be225d8d5ca491cb1d4a30fb1b7466ef6083`
- new input SHA: `5e80fbb54cd527bf205fa741ac6e3a3943d6dd2644d31cfab008d99029dddc9e`
- source SHA（bytes 不变）: `4991323de780ed74be6286cb07f2b1cfae1c03d27aa80ff864a151782bd293de`
- old structure identity: `64524ced362a884a3fdd858696f6e7435598c335c47da900d55c1ece07eeb547`
- new structure identity: `3e07b14cd672a9a1ff5544064dbd6dba12063b628228e5ad8efec6714699e805`
- internal slots: 63 → 64；protected raw/roles 与 formal input 变化；原评级 B。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 20）：old=`a8b1b04fa6b527ec091cea680bede747ce73ed502c1c32f55f6e8c465acd7a37`；new=`8fc2a773266b9dfeaba73565be50f81c849af1246d052c75de6ca7247ec27a70`。

## 20 — _content/doc/database/cancel-operations.md

- old Unit ID: `54f01004345e173d10052bd3a6b194af1292ba0d1d50fcb01b00125f2762eade`
- new Unit ID: `00bcbeef62b7348b1d93580ddcc4cccdff4c14d5cfdde94a6523f92c4926ac5e`
- old input SHA: `6f511371569c6e56e94bd0e4c92702d78882b4a8505f2b686758b4e645c30393`
- new input SHA: `d7c222d00db11dc038256fb895c8ad708ff7c9f912633e13bfafbe56d564c3b0`
- source SHA（bytes 不变）: `c71ef831003d93f55e7ffb503aadb883797e574b21d905caa42c783cf02ff683`
- old structure identity: `bcf4067076c209056a918d02df88983d57adaf0080f4686433ef3067d4e649a7`
- new structure identity: `a5283e1b941e40f7ccb2b20592bcc6726c8e32584ea1215b568c708f88956743`
- internal slots: 15 → 13；protected raw/roles 与 formal input 变化；原评级 A。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 22）：old=`b1527a3654c01a0c01a573c66ee1b1447f91f41511f6bccc235f2538f9e59919`；new=`59b79e0316f4acd06a977fce59c011c8d64b2e6b0661448b92ab7029a2556163`。

## 22 — _content/doc/database/execute-transactions.md

- old Unit ID: `bb64c7810e5c03eab18084368a46be54d2e2961643ca0e67487f0b1d443934d7`
- new Unit ID: `6aea6756a6e2e8960a6b3eb8cd1c35ce6dec89cf1b78558feac894d25bb0a767`
- old input SHA: `25ebc99b29227de04922df34ea1e583b04acb3ba4c8654e41538a5fb56075c56`
- new input SHA: `f7533c1813fa1ca4bd4380fbef43a51b0f5c29f6745954674036ab316f728868`
- source SHA（bytes 不变）: `3e8e4ee1071772b796cf0e467f4a2cc8065e866385604116a6ddd3d0b07ee59f`
- old structure identity: `37e2f9b36751a2c14cb0bd6047579c238aab469b5458b7c2cd84771b3cec70af`
- new structure identity: `2cdd047b1500341f1108a9c32bf53cbfb5aa7562aa76e26429bf22c34867b3cd`
- internal slots: 48 → 46；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 25）：old=`ffc2428b74c669c98f903dcf8459cdb8ba97f1b3e1570ebf549bcb464d027804`；new=`3012f2de98a635cad5fa1cc1bab68774ab02cd98fb3700a5cb58263449668cd4`。

## 25 — _content/doc/database/open-handle.md

- old Unit ID: `6052db6a0e2099fd25b7a55fec52169237a1ed59800e5bd195f46c75d2f28a66`
- new Unit ID: `e9374ce4c3a09a205ebe3c943ef6405286703f2d0ba2f92063f2b0abf3fbceb7`
- old input SHA: `a9ba67319d4990283b7049510dcd635d38fd2c5c31f2d6f3a597b2c36687ccbc`
- new input SHA: `9784f75daf39062e80e062b55e54fbaf4992a6509fddb8fe55e1808799963c91`
- source SHA（bytes 不变）: `544c86d9189672ad1008bed23999c977e1013042cb5d3d30a145b0efb6bcf58c`
- old structure identity: `50a4f651265e3dacb129eeec89830ff82402110ec79d11db3b9c180f0ffa0447`
- new structure identity: `fd23d54ce0b4e37c5b1102ebb5f027ca44d018290675b70e7712b65136dbbd4c`
- internal slots: 48 → 48；protected raw/roles 与 formal input 变化；原评级 B。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 26）：old=`2ce1945bc287a9fa7ec80a60243aba4ab1946a5dd37b91412956a26f11dc172f`；new=`fb42bf90d4474b6078ec514bd47670bfe84d8ec99595baf1efd53d984d4345cc`。

## 26 — _content/doc/database/prepared-statements.md

- old Unit ID: `770fe80b23e2deb4619d1ad7ef40f6cb691c349f5712dffd9d31fe0f70926086`
- new Unit ID: `3111096562c8eefbefc8af2f1b2fd5d56583aa990de823606b5d228d5ce4d428`
- old input SHA: `67f1e9455c66f7e59438af06cc2572c4649a403d7ccac4063079dc6975239531`
- new input SHA: `dff8579c4fe1b365f4eb0a2250f0ab4bebbc8b6ce72eb9a8ad9bdc1326d77185`
- source SHA（bytes 不变）: `37854cb2a8e7ef640129566162422472821faadcbab8dd2a7514513d40eff004`
- old structure identity: `ffde4bab1a0ef93b35daaf5f1de9ea7eacc2dc9dd51c382b7107c21c5c1b21c7`
- new structure identity: `ebbd5044a64f61137e0b653bc3a6229b98c1714014c57c8e846241792cf31549`
- internal slots: 29 → 28；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 27）：old=`63476bf922d1eed2a1af1deffc49fdd6a1fa8c6fc6c0c085415ed07ee126e5e8`；new=`f8e178de8852494ad8527158571c4d86cf2edeb5f8fda04def99e36cb6779851`。

## 27 — _content/doc/database/querying.md

- old Unit ID: `554eaa7b7d70b8b634e5bbc5d4309b72bd63abae2a18e1af103b357e5f53a8fb`
- new Unit ID: `6fd0d36e7073cb6ef85352612edcb9ecb1bb3b2e7e7482a2b91a54259f9d9e2f`
- old input SHA: `9b9654001b0df7095535f9f38fdac53ae980d70d9aab0d60d2b090ab1b109958`
- new input SHA: `5f82b83ffb636d308957b472aaf8ebf96a1251456d00e2042429f8810c37e12b`
- source SHA（bytes 不变）: `02e399c4e8b9895689676f976ce052da6d6efa6f6bed3c7368314a47e3314864`
- old structure identity: `e27b3d86c13f64201a195e68129421686089566a6256497a271b253171ce339b`
- new structure identity: `ee20f3721aedc2d8c690539fd9c97bece293de2e74bcd0347445c109fc8b6e49`
- internal slots: 61 → 59；protected raw/roles 与 formal input 变化；原评级 A。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

正式 Unit source identity（index 31）：old=`79fac4abdafaa7fb5d377e108b31e1bf28c3cd1081fdac3f2152365a6f2dbfd4`；new=`6e8a5133bd084566b787b58f977557dfedef5ee9ed710e2aa1754fcafc7aa274`。

## 31 — _content/doc/modules/layout.md

- old Unit ID: `2dc9a561d950be08d2963289b78129362fb12e4c57519637b3e1eea632057f46`
- new Unit ID: `b490dbaa9904c807b80d7df512293e4cfc00599d940f6a3dba415295ee15e820`
- old input SHA: `87ade25156c2e9c9b29df8d615b66c092e642425e0734aeab0e3e99d9d8713b1`
- new input SHA: `7b8ed20ee1ef45ef45b509b5796d03bcfca77880185f6e9544f1829009332da7`
- source SHA（bytes 不变）: `3cf064a34e84e48dc66ffdf9277fea2b2c9fdc52d0b5dd7d4de506fb8eeba4a5`
- old structure identity: `f2798a9e434e607e35b30e6b581942914572da9cea278f47bf5a7ff23c7b0730`
- new structure identity: `a5a2069d5eb65e71f594fcaabdc8dc3ccc9293d076e8cee7df4175d51f62fe02`
- internal slots: 39 → 40；protected raw/roles 与 formal input 变化；原评级 C。
- 原因：共享 analyzer 现在接受完整 Go 文件，完整 teaching body 保留为一个 slot，identifier 为 comment 内 atom；其余代码仍为 boundary。旧 candidate 不兼容，需 Generation/validation 和独立 QC。

## 精确 scope

changed=19；unchanged=30。

- identity_changed / Generation + validation + QC: `[2 3 5 6 7 8 10 12 13 15 17 18 19 20 22 25 26 27 31]`
- unchanged A carry-forward: `[4 14 16 21 23 28 34 45 48]`
- unchanged B/C 可按原 finding revision: `[1 9 11 24 29 30 32 33 35 36 37 38 39 40 41 42 43 44 46 47 49]`

unchanged 的 path/source bytes/internal slots/protected raw/roles/formal input/Unit ID/source identity/structure identity 全部逐字段一致；candidate 字节、restore 和当前 validator 均通过。

## 全部49 Page 的逐字段对照

全部 source bytes unchanged。teaching span 是 source coordinate 内部教学槽位；before / after 完整 Document（含 slots、protected raw/role/pair、正式 input）分别保存在 `/tmp/go-tour-teaching-before.json` 和 `/tmp/go-tour-teaching-after.json`。unchanged 使用完整 Document 的 deep equality 判定；不是仅比较 Unit ID。

| index | path | 状态 | slots old/new | teaching spans old/new | protected old/new |
| --- | --- | --- | --- | --- | --- |
| 1 | _content/learn/index.md | unchanged | 20/20 | 0/0 | 108/108 |
| 2 | _content/doc/tutorial/add-a-test.html | identity_changed | 23/27 | 0/4 | 122/133 |
| 3 | _content/doc/tutorial/call-module-code.html | identity_changed | 31/32 | 0/1 | 160/162 |
| 4 | _content/doc/tutorial/compile-install.html | unchanged | 26/26 | 0/0 | 109/109 |
| 5 | _content/doc/tutorial/create-module.html | identity_changed | 43/45 | 0/2 | 180/185 |
| 6 | _content/doc/tutorial/database-access.md | identity_changed | 170/177 | 13/20 | 519/550 |
| 7 | _content/doc/tutorial/fuzz.md | identity_changed | 153/152 | 4/3 | 449/454 |
| 8 | _content/doc/tutorial/generics.md | identity_changed | 121/126 | 11/16 | 318/343 |
| 9 | _content/doc/tutorial/getting-started.html | unchanged | 51/51 | 0/0 | 179/179 |
| 10 | _content/doc/tutorial/govulncheck-ide.md | identity_changed | 30/32 | 0/2 | 94/97 |
| 11 | _content/doc/tutorial/govulncheck.md | unchanged | 55/55 | 0/0 | 174/174 |
| 12 | _content/doc/tutorial/greetings-multiple-people.html | identity_changed | 43/39 | 22/18 | 123/141 |
| 13 | _content/doc/tutorial/handle-errors.html | identity_changed | 37/34 | 14/11 | 109/111 |
| 14 | _content/doc/tutorial/index.html | unchanged | 27/27 | 0/0 | 74/74 |
| 15 | _content/doc/tutorial/json.md | identity_changed | 66/66 | 2/2 | 277/279 |
| 16 | _content/doc/tutorial/module-conclusion.html | unchanged | 6/6 | 0/0 | 26/26 |
| 17 | _content/doc/tutorial/random-greeting.html | identity_changed | 22/22 | 5/5 | 75/78 |
| 18 | _content/doc/tutorial/web-service-gin.md | identity_changed | 155/156 | 19/20 | 448/488 |
| 19 | _content/doc/tutorial/workspaces.md | identity_changed | 63/64 | 0/1 | 193/196 |
| 20 | _content/doc/database/cancel-operations.md | identity_changed | 15/13 | 4/2 | 86/86 |
| 21 | _content/doc/database/change-data.md | unchanged | 18/18 | 2/2 | 100/100 |
| 22 | _content/doc/database/execute-transactions.md | identity_changed | 48/46 | 12/10 | 193/195 |
| 23 | _content/doc/database/index.md | unchanged | 29/29 | 0/0 | 162/162 |
| 24 | _content/doc/database/manage-connections.md | unchanged | 22/22 | 0/0 | 139/139 |
| 25 | _content/doc/database/open-handle.md | identity_changed | 48/48 | 5/5 | 212/214 |
| 26 | _content/doc/database/prepared-statements.md | identity_changed | 29/28 | 7/6 | 139/141 |
| 27 | _content/doc/database/querying.md | identity_changed | 61/59 | 11/9 | 279/280 |
| 28 | _content/doc/database/sql-injection.md | unchanged | 9/9 | 1/1 | 58/58 |
| 29 | _content/doc/modules/developing.md | unchanged | 36/36 | 0/0 | 150/150 |
| 30 | _content/doc/modules/gomod-ref.md | unchanged | 207/207 | 0/0 | 733/733 |
| 31 | _content/doc/modules/layout.md | identity_changed | 39/40 | 0/1 | 148/149 |
| 32 | _content/doc/modules/major-version.md | unchanged | 25/25 | 0/0 | 80/80 |
| 33 | _content/doc/modules/managing-dependencies.md | unchanged | 142/142 | 0/0 | 635/635 |
| 34 | _content/doc/modules/managing-source.md | unchanged | 44/44 | 0/0 | 135/135 |
| 35 | _content/doc/modules/publishing.md | unchanged | 18/18 | 0/0 | 64/64 |
| 36 | _content/doc/modules/release-workflow.md | unchanged | 83/83 | 0/0 | 332/332 |
| 37 | _content/doc/modules/version-numbers.md | unchanged | 67/67 | 0/0 | 209/209 |
| 38 | _content/doc/security/best-practices.md | unchanged | 19/19 | 0/0 | 127/127 |
| 39 | _content/doc/security/decisions.md | unchanged | 58/58 | 0/0 | 183/183 |
| 40 | _content/doc/security/fips140.md | unchanged | 54/54 | 0/0 | 264/264 |
| 41 | _content/doc/security/fuzz/index.md | unchanged | 90/90 | 0/0 | 392/392 |
| 42 | _content/doc/security/fuzz/technical.md | unchanged | 32/32 | 0/0 | 59/59 |
| 43 | _content/doc/security/index.md | unchanged | 33/33 | 0/0 | 127/127 |
| 44 | _content/doc/security/policy.md | unchanged | 59/59 | 0/0 | 284/284 |
| 45 | _content/doc/security/threat-model.md | unchanged | 17/17 | 0/0 | 36/36 |
| 46 | _content/doc/security/vuln/cna.md | unchanged | 12/12 | 0/0 | 69/69 |
| 47 | _content/doc/security/vuln/database.md | unchanged | 78/78 | 0/0 | 370/370 |
| 48 | _content/doc/security/vuln/editor.md | unchanged | 19/19 | 0/0 | 68/68 |
| 49 | _content/doc/security/vuln/index.md | unchanged | 47/47 | 0/0 | 238/238 |
