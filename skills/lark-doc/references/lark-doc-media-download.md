
# docs +media-download（下载文档素材/画板缩略图）

> **前置条件：** 先阅读 [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和安全规则。

下载文档中的图片/文件素材（`file_token`），或下载画板缩略图（`whiteboard_id`）。当 `--output` 不带扩展名时，会根据响应的 `Content-Type` 自动补全扩展名。

## 选择规则

- 用户明确说“下载素材”时，使用 `docs +media-download`
- 用户只是想查看、预览图片或文件素材时，优先使用 [`docs +media-preview`](lark-doc-media-preview.md)
- 如果素材来自**开启了高级权限**的多维表格附件，必须额外传 `--extra`
- 如果目标明确是画板 / whiteboard / 画板缩略图，继续使用 `docs +media-download --type whiteboard`；`+media-preview` 不支持画板

## 命令

```bash
# 下载图片/文件素材（默认 type=media）
lark-cli docs +media-download --token "Z1Fjxxxxxxxx" --output ./asset

# 指定输出文件名（带扩展名则不会自动补全）
lark-cli docs +media-download --token "Z1Fjxxxxxxxx" --output ./asset.png

# 下载开启高级权限的多维表格附件
lark-cli docs +media-download \
  --token "box_masked_file_token" \
  --output ./asset \
  --extra '{"bitablePerm":{"tableId":"tbl_masked_table_id","attachments":{"fld_masked_field_id":{"rec_masked_record_id":["box_masked_file_token"]}}}}'

# 下载画板缩略图（whiteboard token）
lark-cli docs +media-download --type whiteboard --token "wbcnxxxxxxxx" --output ./whiteboard
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--token <token>` | 是 | 资源 token：素材为 `file_token`，画板为 `whiteboard_id` |
| `--output <path>` | 是 | 本地保存路径；不带扩展名会自动补全 |
| `--type <type>` | 否 | `media`（默认）或 `whiteboard` |
| `--extra <json>` | 否 | 仅 `--type media` 使用。开启了高级权限的多维表格附件下载时必填；传**未 URL 编码**的 JSON 字符串，CLI 会负责放入 query 参数 |

## token 从哪里来

- 若你是从文档内容里提取：`lark-doc-fetch` 返回的 Markdown 里可能包含：
  - 图片：`<image token="..." .../>`
  - 文件：`<file token="..." name="..."/>`
  - 画板：`<whiteboard token="..."/>`

## 高级权限多维表格附件的 `--extra`

当附件来自**开启了高级权限**的多维表格时，下载接口必须追加 `extra` 鉴权；否则通常会返回 `HTTP 400`。

### 传参规则

- 传给 CLI 的值必须是**原始 JSON 字符串**，不要自己做 URL 编码
- 如果你拿到的是记录接口返回的附件 `url` / `tmp_url`，其中 query 里的 `extra` 往往已经被编码过；先 URL decode 一次，再把解码后的 JSON 字符串传给 `--extra`
- `attachments` 建议始终带上；官方说明里这是构造高级权限附件下载鉴权时的必填部分

### 推荐构造格式

```json
{"bitablePerm":{"tableId":"tbl_masked_table_id","attachments":{"fld_masked_field_id":{"rec_masked_record_id":["box_masked_file_token"]}}}}
```

字段含义：

- `extra`：下载接口的 query 参数，类型是字符串；值内容是一个 JSON 对象序列化后的字符串
- `extra.bitablePerm`：多维表格高级权限附件下载使用的鉴权对象
- `extra.bitablePerm.tableId`：附件所在数据表的 `tableId`
- `extra.bitablePerm.attachments`：附件定位信息对象；key 是附件字段的 `field_id`
- `extra.bitablePerm.attachments.<field_id>`：某个附件字段下的记录映射对象；key 是 `record_id`
- `extra.bitablePerm.attachments.<field_id>.<record_id>`：文件 token 数组，表示这条记录在该附件字段下要下载的文件集合
- `extra.bitablePerm.attachments.<field_id>.<record_id>[i]`：单个附件的 `file_token`

按上面的脱敏示例展开后，对应关系如下：

- `tbl_masked_table_id`：占位符，表示真实 `tableId`
- `fld_masked_field_id`：占位符，表示真实附件字段 `field_id`
- `rec_masked_record_id`：占位符，表示真实 `record_id`
- `box_masked_file_token`：占位符，表示真实附件 `file_token`

### 典型来源

- `base +record-get` / `base +record-search` 返回的附件字段
- 附件对象中的 `url` 或 `tmp_url`

如果附件对象已经返回完整下载链接，优先复用该链接里的 `extra` 信息，不要自行猜 `tableId / field_id / record_id`。

## 排障

- 如果是多维表格附件并返回 `HTTP 400`，先检查是否漏传 `--extra`，或是否把已经编码过的 `extra` 又编码了一次
- 如果报错返回的信息包含 `HTTP 403`，且目标是图片/文件素材，可以改成调用 [`docs +media-preview`](lark-doc-media-preview.md) 看是否能先预览内容

## 参考

- [lark-doc-fetch](lark-doc-fetch.md) — 获取文档内容（用于提取 token）
- [lark-doc-media-preview](lark-doc-media-preview.md) — 预览素材
- [lark-shared](../../lark-shared/SKILL.md) — 认证和全局参数
