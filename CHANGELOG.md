# Changelog

## [0.2.0](https://github.com/Serraniel/Wokko/compare/v0.1.0...v0.2.0) (2026-11-07)


### Features

* **auth:** auto-redirect to /setup on first run ([#37](https://github.com/Serraniel/Wokko/issues/37)) ([e5668f3](https://github.com/Serraniel/Wokko/commit/e5668f376804d0fefc361374b347fe8b782f5c99))
* **kitshn:** support writing recipes, meal plans, shopping, books and imports ([7d7176c](https://github.com/Serraniel/Wokko/commit/7d7176c397ba0e5d4f6a400e92e9b09e688921c8))
* **mcp:** add passkeys and an MCP server with tokens and OAuth ([2ec063b](https://github.com/Serraniel/Wokko/commit/2ec063b60d8d0b91fd0c842cdd588fe57b212291))
* **recipes:** add books, all-recipes links, meal plans, shopping and nutrition APIs ([b9ff6a7](https://github.com/Serraniel/Wokko/commit/b9ff6a7e2fbdc93cb9c31334468fd7bf1139a4cb))
* **recipes:** import recipe files from Tandoor, Mealie and Paprika ([4f76e33](https://github.com/Serraniel/Wokko/commit/4f76e33acbaa95b1c734ae123a44e8b0cda84615))
* **recipes:** import recipes from links and pasted text ([3fc480f](https://github.com/Serraniel/Wokko/commit/3fc480fb0a144bcf03f4e8458e796c56318991c2))
* **recipes:** rewrite the backend on SQLite with data safety first ([384d0ff](https://github.com/Serraniel/Wokko/commit/384d0ff3cce55896a97db5375439f29a29c58a09))
* **ui:** add meal plan, shopping, books, cooking mode and history pages ([18e10e5](https://github.com/Serraniel/Wokko/commit/18e10e50e4a77dacc39ca852a303b61c4e08c826))
* **ui:** add printing, live cooking on shared links, a footer and tags on cards ([f8ca787](https://github.com/Serraniel/Wokko/commit/f8ca7875b5d8c725ac4d50025dea437517749e01))
* **ui:** overhaul the look and speed of the app ([c7fb84d](https://github.com/Serraniel/Wokko/commit/c7fb84dd64fef7af385648be6577bef190254ce7))
* **ui:** rebuild the frontend as a mobile-first static app ([58b3d47](https://github.com/Serraniel/Wokko/commit/58b3d47bfbd9435d523ea0e3b87c8ea882ef1e19))
* **ui:** work offline and accept pages shared from other apps ([1758bc6](https://github.com/Serraniel/Wokko/commit/1758bc66acdc0bfc020d86c20ab2300ac76dfc07))


### Fixes

* **auth:** auto-refresh expired access token on 401 and retry request ([#55](https://github.com/Serraniel/Wokko/issues/55)) ([d27fa7c](https://github.com/Serraniel/Wokko/commit/d27fa7c3ec2173a5d95219956447a107ecbaf035))
* **auth:** non-destructive session validation, fixes [#45](https://github.com/Serraniel/Wokko/issues/45) [#47](https://github.com/Serraniel/Wokko/issues/47) [#50](https://github.com/Serraniel/Wokko/issues/50) ([#52](https://github.com/Serraniel/Wokko/issues/52)) ([281abe1](https://github.com/Serraniel/Wokko/commit/281abe1fb7a2be4959e3f19334e2aaa9d553f5c2))
* **infra:** prevent storage.New panic on empty path, set UploadPath in test config ([818d16d](https://github.com/Serraniel/Wokko/commit/818d16d2a818e426185ab786c5a51aa86e31c673))
* **infra:** run migrations in test server setup to fix CI test failures ([8eabde2](https://github.com/Serraniel/Wokko/commit/8eabde2c45dcd6166e7ce0dea99294bff30c5add))
* **infra:** update Go version to 1.26 in Dockerfile and CI ([69ea7c8](https://github.com/Serraniel/Wokko/commit/69ea7c881aa29bfc8a3cefccc65c0838fadf6769))
* **infra:** use absolute migrations path in container ([e695677](https://github.com/Serraniel/Wokko/commit/e69567745dca0e41d0a21268b073f43e2ce0f5f9))
* **kitshn:** answer the version check before login ([2454804](https://github.com/Serraniel/Wokko/commit/2454804944ba4a769e46ce4dd3cbda727aa5503e))
* **kitshn:** repair the test broken by the version check change ([4bf81b4](https://github.com/Serraniel/Wokko/commit/4bf81b41e379026008958c784d37c508de9f98b5))
* **recipes:** fix empty ingredient names and step descriptions on recipe detail page ([#54](https://github.com/Serraniel/Wokko/issues/54)) ([f5e4ce9](https://github.com/Serraniel/Wokko/commit/f5e4ce97ad3135eae1bda9010751c6ae0dc91260))
* **ui:** always show invite code field on register page ([#35](https://github.com/Serraniel/Wokko/issues/35)) ([fb4a3fe](https://github.com/Serraniel/Wokko/commit/fb4a3fe1dc8270ca565e56c8f61996976409b08b))
* **ui:** autofocus first input field on login, register, and setup pages, closes [#38](https://github.com/Serraniel/Wokko/issues/38) ([#39](https://github.com/Serraniel/Wokko/issues/39)) ([20e5f81](https://github.com/Serraniel/Wokko/commit/20e5f81e9b3aeab3b6711b998298ee7d2a69eee2))
* **ui:** enable Tailwind v4 class-based dark mode, fix nav and icon colors, closes [#41](https://github.com/Serraniel/Wokko/issues/41) [#42](https://github.com/Serraniel/Wokko/issues/42) [#46](https://github.com/Serraniel/Wokko/issues/46) ([#53](https://github.com/Serraniel/Wokko/issues/53)) ([38e23d2](https://github.com/Serraniel/Wokko/commit/38e23d214f61a041abb484587b43e1217495594c))
* **ui:** fix issues from a full browser and accessibility audit ([5f6c35c](https://github.com/Serraniel/Wokko/commit/5f6c35cecd6db5276700fd328dbaac2e2796e745))
* **ui:** proxy /api/* through SvelteKit to backend ([cea83ae](https://github.com/Serraniel/Wokko/commit/cea83ae3a91c78100e32edd4ab614b7e956be9ea))
* **ui:** proxy /api/* through SvelteKit to backend, fixes 404 on all API calls ([a78fd12](https://github.com/Serraniel/Wokko/commit/a78fd12c03be781fe5ee1a92e0a12997684c41df))
* **ui:** wire ingredient and unit fields in recipe editor to Combobox ([#56](https://github.com/Serraniel/Wokko/issues/56)) ([b0f31ad](https://github.com/Serraniel/Wokko/commit/b0f31ad78af1b0d6245c3616b253f4ccdfe34140))

## 0.1.0

The first foundation, later replaced by a full rewrite. It was never used in production.
