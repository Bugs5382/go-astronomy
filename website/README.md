# go-astronomy documentation site

The [Docusaurus](https://docusaurus.io) source for the go-astronomy guides,
deployed to GitHub Pages by `.github/workflows/job-docs-pages.yaml`.

## Local development

```sh
npm ci        # install dependencies (uses package-lock.json)
npm start     # dev server with hot reload at http://localhost:3000/go-astronomy/
npm run build # production build into website/build
npm run serve # serve the production build locally
```

From the repository root, `task docs` runs the production build.

## Layout

- `docs/` — the guide and reference Markdown pages.
- `sidebars.ts` — the manual sidebar ordering.
- `docusaurus.config.ts` — site config, including the GitHub Pages project-site
  `baseUrl` (`/go-astronomy/`).
- `src/` — the homepage and theme customization.

The full generated API reference lives on
[pkg.go.dev](https://pkg.go.dev/github.com/Bugs5382/go-astronomy); this site
covers concepts and a task-oriented tour of the public packages.
