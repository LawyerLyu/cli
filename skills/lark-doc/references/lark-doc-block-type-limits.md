# Block type limits (read-only blocks)

> **前置条件：** 先阅读 [`../SKILL.md`](../SKILL.md) 了解 `docs +create` / `docs +update` 的调用方式，必要时再读 [`lark-doc-xml.md`](lark-doc-xml.md)（资源块语法的权威定义）。

## 概念区分（避免误读）

`docs +create` / `docs +update` 并非支持全部 block 类型。**这些 block 在文档里 _可以存在_（通过 fetch 读出、通过 `block_move_after` 在文档之间搬动），但 _不能创建/更新_** —— 也就是说写入路径会静默跳过它们，不会抛错也不会出现在新文档中。两条路径行为如下：

- **写入路径（`docs +create` / `docs +update --command append/overwrite/...`）**：如果输入里出现这些 block 标签，写入阶段会被静默跳过，API 返回 `code=0 / success=true`，但 block 根本没进入新文档。
- **读取路径（`docs +fetch`）**：原文档里**已经存在**的同类 block 仍会被序列化输出（保留为对应 XML 标签）；少数飞书内部块如果无法稳定序列化，**v1 fetch（MCP `fetch-doc`）+ markdown 输出**会以 `<!-- Unsupported block type: N -->` 形式占位呈现。v2 fetch（`POST /open-apis/docs_ai/v1/documents/<token>/fetch`）默认 XML 输出，走结构化 block 表示，无此 markdown 注释占位符。
- **跨文档移动（`docs +update --command block_move_after`）**：所有 block 都支持移动，因此这些 read-only block 可以从一篇文档搬到另一篇。`block_copy_insert_after` 也可对部分类型生效，详见 [`lark-doc-xml.md` §四 块级复制与移动](lark-doc-xml.md#四块级复制与移动)。

简单说：**这些 block 不是"完全只读"，只是"create / update 时无法新建"**。

## 已知不支持 create / update 的 block

权威定义见 [`lark-doc-xml.md` §三 资源块](lark-doc-xml.md#三资源块)（『不可创建，仅支持移动』）。下表汇总命名、典型 XML 形式与处理建议：

| 块类型 | XML 标签（snake_case）| 典型形态 | 推荐处理 |
|---|---|---|---|
| 引用同步块 | `<synced_reference>` | `<synced_reference src-token="DOC_TOKEN" src-block-id="BLK_ID"/>` | 写入路径无效，需通过 UI 手动绑定；fetch 读出后可用 `block_move_after` 搬到目标文档 |
| 源同步块 | `<synced_source>` | `<synced_source>...内容...</synced_source>` | 同上 |
| 多维表格 | `<bitable>` | `<bitable token="APP_TOKEN" table-id="TBL_ID"/>` | 在另一篇文档里嵌入已有 base 时，先 fetch 拿到 `<bitable>` 标签，再 `block_move_after` 搬过去 |
| Base 引用 | `<base_ref>` | `<base_ref ...>` | 同上 |
| OKR | `<okr>` | `<okr ...>` | 同上 |

> **复制限制（`block_copy_insert_after`）**：上述五类（外加 `task`）都**不支持复制**；img / source / whiteboard / sheet / chat_card 才支持复制。

## 会产生 `<!-- Unsupported block type: N -->` 占位符的块（v1 fetch / markdown 输出）

> 本节专指 **v1 MCP `fetch-doc` + markdown 输出** 的兜底行为。v2 fetch 默认 XML 输出走结构化 block 表示，不会出现此 markdown 注释形式；v2 markdown 输出的兜底未单独验证，可能与 v1 不同。

v1 `fetch-doc` 导出 markdown 时遇到无法稳定序列化的原生 block，会以 `<!-- Unsupported block type: <N> -->` 形式占位（例如 block type 53）。典型触发者包括：

- 部分 **文档小组件（AddOns）** — `<add-ons component-type-id="..." record='{...}'/>` 子集
- **Wiki SubPageList** — `<sub-page-list wiki="..."/>` 在 wiki 节点以外的上下文
- **议程（Agenda）** 的部分子块

如果 round-trip 回灌时看到这些注释，**不要直接把注释当成 markdown 源再 create / update** —— v1 / v2 写入路径都不会把 HTML 注释解析成 block。需要人工在 UI 中重建，或寻找对应的 OpenAPI 专用接口。

## 定位"该块没进去"的信号

1. `docs +create` / `docs +update` 响应 `code=0`、`success=true`，UI 上却找不到预期 block（v1 / v2 通用）
2. 随后 `docs +fetch` 拿回来的内容里该块消失；若用 **v1 fetch + markdown** 还可能出现 `<!-- Unsupported block type: ... -->` 注释
3. round-trip diff 多出一段 `+<synced_reference ...>` / `-<synced_reference ...>` 或类似标签

出现上述任一信号时，优先怀疑 block 类型在上面表格中。

## 对 AI Agent 的影响

- **周报 / 文档模板场景**：首行 `synced_reference` 团队介绍块必须通过 UI 手动补上；skill 里应显式记录"绑定同步块"是手工步骤，不要在 markdown 里伪造。
- **文档迁移 / round-trip**：跨文档迁移同步块、bitable、okr 等优先用 `block_move_after`（搬不复制）；权限或环境变化导致引用断链时，需要人工修复。
- **生成式内容**：Agent 从零生成时应避免主动插入 `<synced_reference>`、`<bitable>`、`<base_ref>`、`<okr>` 等标签（写不进去）；如果是从一篇旧文档迁内容到新文档，先 fetch 这些 block 再用 `block_move_after`。
