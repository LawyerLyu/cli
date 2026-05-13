# Visual Planning

Use this to define the deck's visual system before generating slide XML.

## Decisions

- Palette: background, primary accent, secondary accent, text, muted text, border.
- Typography: title, section heading, body, caption, metric number.
- Layout rhythm: margins, grid, recurring title position, footer treatment.
- Components: cards, callouts, timelines, charts, tables, quote blocks, section dividers.

## Guidance

- Business reports should be quiet, readable, and scannable.
- Product or technology decks can use stronger contrast, but keep hierarchy clear.
- Use repeated structure across related slides.
- Keep text inside predictable bounds; leave enough whitespace for rendering variance.
- Do not rely on external image URLs in XML. Images must become `file_token` values through the execution workflow.

## XML Note

Before writing XML, read `../lark-slides/references/xml-schema-quick-ref.md`. Gradient fills must use `rgba()` stops with percentages.
