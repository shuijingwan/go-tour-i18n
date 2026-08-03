# 页面持久身份

## `page_id` 的用途

`page_id` 是课程页面在本项目中的持久身份，也是语言状态、候选文件和未来发布记录的永久关联键。当前 [`data/tour-pages.tsv`](data/tour-pages.tsv) 中的 103 个正式发布页面 `page_id` 已冻结。

`route` 是页面当前的课程访问路径，`article` 和 `section_number` 是页面在当前上游 article 中的位置，`source_title` 用于诊断，`source_sha256` 标识当前完整英文页面源版本。这些字段都可能在上游更新时变化，但变化本身不得导致 `page_id` 自动变化。

当前 ID（如 `welcome/1`、`basics/17`）看起来与 route 和位置编号相同，只是因为首次分配采用了当时的位置。不能据此假定未来 `page_id` 永远等于 route，也不能从新的 `section_number` 重新生成 ID。

`welcome/4` 与 `welcome/5` 是从上游 `#appengine:` 条件源投影出的正式页面，当前 route 分别为 `/welcome/3` 与 `/welcome/4`；原有 `welcome/3` 的 route 因发布顺序插入而移动到 `/welcome/5`，其持久 ID 与源哈希均保持不变。条件源本身仍在 `data/tour-conditional-pages.tsv` 中独立留档。

## 上游变化规则

- 上游插入页面时，后续旧页面通过保守匹配继续使用原 `page_id`，不得按新位置重编号。
- 上游删除页面时，原 ID 和语言状态不得自动删除；该 ID 永不复用。
- 上游重排或跨 article 移动页面时，只有唯一的完整源哈希匹配才能自动保留身份并更新当前位置。
- 当前 route 仍存在但内容变化时，只有 protected structure 唯一兼容才可标为 `content_changed`。
- 证据不足、重复特征、移动同时发生结构变化等情况均为 `ambiguous`，必须人工决定。

新增页面不会由普通 `catalog write` 隐式获得 ID。未来同步流程必须显式分配新 ID，推荐使用所属 article 内下一个从未使用的数字，不插入编号、不重排旧 ID。例如 `basics` 已使用 1 至 17 时，在第 5 个位置插入的新页面可获得 `basics/18`，同时 route 为 `/basics/5`。

## 语言文件路径

语言页面路径使用持久 `page_id`：

```text
locales/zh-CN/pages/<page_id>.article
```

例如 `locales/zh-CN/pages/welcome/1.article`。即使该页未来 route、article 或 `section_number` 改变，语言文件路径也不应自动重命名。

## 源版本与变化预览

`source_sha256` 是当前完整英文源页面的 SHA-256，用于判断候选译文对应哪个英文源版本，而不是页面身份本身。

在导入新的上游版本前运行只读预览：

```bash
go run -mod=readonly ./cmd/tour-i18n upstream preview \
  --source-root /path/to/website
```

预览将页面分类为 `unchanged`、`content_changed`、`moved`、`added`、`removed` 或 `ambiguous`。`ambiguous` 必须人工映射；预览不会修改 catalog、`status.tsv` 或任何语言文件。

`catalog write` 只在现有页面能够安全一一对应时更新允许变化的目录字段，不负责自动迁移语言状态，不创建新 ID，也不删除旧状态。出现 `added`、`removed` 或 `ambiguous` 时会停止并提示先查看预览。

当前尚未发生第二次 upstream 同步，固定上游 commit 仍未改变。
