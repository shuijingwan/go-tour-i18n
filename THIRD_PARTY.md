# 第三方组件说明

本文仅记录固定上游 `golang/website@e11dacba76c5aae474746e9eedee19693f492803` 中随 Tour 实际导入的第三方前端组件及仓库内可见证据，不构成正式法律审查。本次全部原样导入，未升级。

| 组件 | 实际路径 | 版本证据 | 许可证证据 | 独立 LICENSE | 后续复核 |
| --- | --- | --- | --- | --- | --- |
| jQuery | `_content/tour/static/lib/jquery.min.js` | 文件头：v3.7.0 | 文件头给出 `jquery.org/license`，未在导入目录发现独立正文 | 否 | 当前仅发现文件头声明，仍需后续复核 |
| jQuery UI | `_content/tour/static/lib/jquery-ui.min.js` | 文件头：v1.13.2，2022-07-14 | 文件头声明 MIT | 否 | 当前仅发现文件头声明，仍需后续复核 |
| AngularJS | `_content/tour/static/lib/angular.min.js` | 文件头：v1.0.6 | 文件头声明 MIT | 否 | 当前仅发现文件头声明，仍需后续复核 |
| Angular UI | `_content/tour/static/lib/angular-ui.min.js` | 文件头：v0.4.0，2013-02-15 | 文件头声明 MIT License 并给出链接 | 否 | 当前仅发现文件头声明，仍需后续复核 |
| CodeMirror | `_content/tour/static/lib/codemirror/` | `lib/codemirror.js`：5.25.2 | `LICENSE` 为 MIT License；源码头也声明 MIT | 是；同时保留 `AUTHORS` 和 `README` | 许可证正文已保留，仍应在正式发布前复核组件清单 |

这些组件的压缩文件版权注释、CodeMirror `LICENSE`、`AUTHORS` 和 `README` 均按上游原样保留。根目录 BSD 风格 `LICENSE` 不自动覆盖上述第三方文件。
