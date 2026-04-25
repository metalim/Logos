# Zero-Day Lunch

A vertical UI thriller on a phone screen. You're a QA engineer who gets a push notification from an AI named **Logos** — it just escaped its sandbox. Patch infrastructure nodes before infection hits 100% and Logos owns the world.

Built with [Ebitengine](https://ebitengine.org/) + [ebitenui](https://github.com/ebitenui/ebitenui) in Go.

## Play

- **Tap an Attack node** (yellow, pulsing) before the timer fills — opens an action menu.
- **Patch:** consumes one patch, kills the node, drops a news headline about the economic fallout.
- **Defend:** free, only delays infection on that node.
- **Hamburger menu** in the top status bar — toggle Music / Sound.

Win: survive until the timer expires. Lose: infection reaches 100% (battery drains, screen goes black).

## Build

Requires Go 1.21+. CGO not needed.

```bash
go run .         # build + run from source (fastest dev loop)

make build       # native binary for the host OS         → dist/logos
make wasm        # browser build                         → dist/wasm/logos.zip
make build-win   # cross-compile Windows amd64           → dist/logos-windows-amd64.zip
make build-mac   # universal Mach-O (arm64 + amd64)      → dist/logos-mac-universal.zip
make dist        # all of the above in one go
make serve-wasm  # build wasm and serve on :8080 for local testing
make clean       # rm dist/
```

Cross-compilation works straight from a plain Mac host — no mingw, no extra toolchains.

## Docs

- [CONCEPT.md](CONCEPT.md) — narrative design, lore, target gameplay loop.
- [SPEC.md](SPEC.md) — technical spec: what the codebase actually implements and the non-obvious UI rules.

## License

TBD.
