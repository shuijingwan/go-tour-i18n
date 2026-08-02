# zh-CN 公共 UI 资源边界

公共 UI 文案与课程正文分开维护。当前英文 UI 源仍位于 `_content/tour/static/js/values.js` 等上游文件，AngularJS 保持不变。

后续里程碑会把 UI 文案提取为声明式语言资源，支持键完整性校验、缺失键检测，以及构建时生成旧前端所需的 JavaScript。本次不决定最终采用 JSON 还是 YAML，不创建伪造的中文 UI 文案，也不复制英文内容冒充中文资源。
