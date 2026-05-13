# Asset Planning

Use this when a deck needs screenshots, photos, diagrams, logos, icons, or chart data.

## Asset Plan

For each asset, record:

- Slide number and purpose.
- Asset type: screenshot, product image, chart, diagram, logo, icon, photo.
- Source: provided file, generated file, downloaded file, or chart from data.
- Local path under the current working directory.
- Intended placement and dimensions.

## Rules

- Slides XML cannot use HTTP(S) image URLs directly.
- For a new deck using `+create --slides`, local image placeholders can use `src="@./path.png"`.
- For existing decks or raw slide APIs, upload first with `slides +media-upload`, then use the returned `file_token`.
- Keep source files inside the current working directory or a safe project subdirectory.
- Check image dimensions and file size before upload; slides media upload limit is 20 MB.
