// sveltekit-superforms's public "sveltekit-superforms/adapters" barrel
// eagerly re-exports every optional schema-library adapter it supports,
// including one for a package named "typebox". The npm package that name
// actually resolves to in this project's dependency tree is an unrelated,
// incompatible library (no `Type.Base`), which throws at import time — even
// though nothing in this project ever uses the typebox adapter, only zod.
//
// A Vite `resolve.alias` on the bare "sveltekit-superforms/adapters"
// specifier was tried first, but this project's rolldown-based Vite dev
// server pre-bundles the barrel's dynamic `import('typebox/compile')` /
// `import('typebox/format')` calls in a way that ignores the alias for
// those subpaths. A plain relative import to the adapter's own file
// (bypassing the package's exports map and the barrel entirely) sidesteps
// the whole problem.
export { zod, zodClient } from '../../../node_modules/sveltekit-superforms/dist/adapters/zod4.js';
