# Fuck Algo extension

Chrome extension, Manifest V3. React 19, TypeScript, Vite with crxjs.

## Run

```
cp .env.example .env
pnpm install
pnpm dev
```

Load `dist-dev` unpacked from `chrome://extensions`. `pnpm build` writes a production build to `dist`.

| Variable | Default | Meaning |
| --- | --- | --- |
| `VITE_API_URL` | `http://localhost:8787` | The Go server |
| `VITE_WEBSITE_URL` | `https://fuck-algo-website.vercel.app` | The website, used when checkout cannot start |

## Layout

See [../docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md). In short: `content/` filters pages, `background/` talks to the server and owns shared state, `offscreen/` holds the microphone, `popup/` and `options/` are the UI, `engine/` is the pure rule logic, `ui/` holds tokens, components and the voice panel.

## Checks

```
pnpm lint && pnpm typecheck && pnpm test && pnpm build
```

## Brand assets

`public/assets/fa-mark.png` and `public/icons/icon-*.png` are cut from the hooded eye mascot. The source art lives in the website repo under `brand/`.
