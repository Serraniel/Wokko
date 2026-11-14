# Changelog

## [0.4.0](https://github.com/Serraniel/Wokko/compare/v0.3.0...v0.4.0) (2026-11-14)


### Features

* **ingredients:** starter catalog updates reach existing databases ([#110](https://github.com/Serraniel/Wokko/issues/110)) ([a97b89e](https://github.com/Serraniel/Wokko/commit/a97b89eda647d16cf779850773fdd5caf9a9b285)), closes [#107](https://github.com/Serraniel/Wokko/issues/107)
* **knowledge:** built-in kitchen knowledge for every recipe ([#108](https://github.com/Serraniel/Wokko/issues/108)) ([57cd0a5](https://github.com/Serraniel/Wokko/commit/57cd0a5a91036d4874a2d8efa7bcb2f19a9467c5)), closes [#105](https://github.com/Serraniel/Wokko/issues/105)
* **knowledge:** set the kind of a food Wokko doesn't recognise ([#111](https://github.com/Serraniel/Wokko/issues/111)) ([6712fba](https://github.com/Serraniel/Wokko/commit/6712fba9821af1e8e9c0620d15f27d32ca3f90a2)), closes [#105](https://github.com/Serraniel/Wokko/issues/105)
* **mcp:** assistants look up the catalog, and you decide on new ingredients ([#104](https://github.com/Serraniel/Wokko/issues/104)) ([e89998f](https://github.com/Serraniel/Wokko/commit/e89998f6d2b396e1d8469ccdeb689d13b0add236)), closes [#102](https://github.com/Serraniel/Wokko/issues/102)
* **recipes:** export a single recipe for another Wokko or other apps ([#97](https://github.com/Serraniel/Wokko/issues/97)) ([03e09e9](https://github.com/Serraniel/Wokko/commit/03e09e94ee1f30cc40a411ae3290c5596f97c801)), closes [#93](https://github.com/Serraniel/Wokko/issues/93)
* **recipes:** export my recipes or all I can see, for Wokko or other apps ([#98](https://github.com/Serraniel/Wokko/issues/98)) ([22bde9d](https://github.com/Serraniel/Wokko/commit/22bde9deaf43dff61d19fca593b21dd7f027b686)), closes [#94](https://github.com/Serraniel/Wokko/issues/94)
* **recipes:** import files exported from another Wokko ([#95](https://github.com/Serraniel/Wokko/issues/95)) ([3a80b4b](https://github.com/Serraniel/Wokko/commit/3a80b4bf9a2cbe910c3b63f60157d5a960a0801a)), closes [#92](https://github.com/Serraniel/Wokko/issues/92)


### Fixes

* **recipes:** ingredients written with an alias use the known food or unit ([#103](https://github.com/Serraniel/Wokko/issues/103)) ([3980952](https://github.com/Serraniel/Wokko/commit/39809521ef992788488e0228bbbe63d9f7231ddc)), closes [#102](https://github.com/Serraniel/Wokko/issues/102)

## [0.3.0](https://github.com/Serraniel/Wokko/compare/v0.2.0...v0.3.0) (2026-11-08)


### Features

* **ui:** add Integrations with setup for each AI app and a choice of access ([#82](https://github.com/Serraniel/Wokko/issues/82)) ([287a3d5](https://github.com/Serraniel/Wokko/commit/287a3d585a1bff770a41cf760be1377dba186e3c))


### Fixes

* **ui:** line up the AI assistants setting on the admin page ([#81](https://github.com/Serraniel/Wokko/issues/81)) ([658d997](https://github.com/Serraniel/Wokko/commit/658d9978bf3f26ee7a6a1df636f065282c529f3a)), closes [#80](https://github.com/Serraniel/Wokko/issues/80)

## [0.2.0](https://github.com/Serraniel/Wokko/compare/v0.1.0...v0.2.0) (2026-11-07)


### Features

* **kitshn:** support writing recipes, meal plans, shopping, books and imports ([c50e30e](https://github.com/Serraniel/Wokko/commit/c50e30ec9421549635ebb5f3ea65eda501e6e685))
* **mcp:** add passkeys and an MCP server with tokens and OAuth ([6d24364](https://github.com/Serraniel/Wokko/commit/6d243644cc329455c904b3e2cc04efbbd883a61c))
* **recipes:** add books, all-recipes links, meal plans, shopping and nutrition APIs ([e6f9661](https://github.com/Serraniel/Wokko/commit/e6f966179512d257f8b8f6506e05abc990aee1f4))
* **recipes:** import recipe files from Tandoor, Mealie and Paprika ([7428694](https://github.com/Serraniel/Wokko/commit/7428694d78337325d3a32ea37bb6e40abdccf931))
* **recipes:** import recipes from links and pasted text ([2cf8892](https://github.com/Serraniel/Wokko/commit/2cf88925cf45c341b7d3bbb5233a83dc6015b6d7))
* **recipes:** rewrite the backend on SQLite with data safety first ([b5a98c8](https://github.com/Serraniel/Wokko/commit/b5a98c8572a0e1909853a8e6facc2fe24c70192e))
* **ui:** add meal plan, shopping, books, cooking mode and history pages ([01a989b](https://github.com/Serraniel/Wokko/commit/01a989ba6cd87e26a93d71cd76c0e1916383d4bf))
* **ui:** add printing, live cooking on shared links, a footer and tags on cards ([ddb2d4a](https://github.com/Serraniel/Wokko/commit/ddb2d4a73877ddf7475ee40084e2917078cdea06))
* **ui:** overhaul the look and speed of the app ([cedfc9b](https://github.com/Serraniel/Wokko/commit/cedfc9b87dbd2944e0f87bb6832ba5026e047c01))
* **ui:** rebuild the frontend as a mobile-first static app ([364213d](https://github.com/Serraniel/Wokko/commit/364213d1484f078f8ee938186b46f0b729849c2b))
* **ui:** work offline and accept pages shared from other apps ([5d9fe04](https://github.com/Serraniel/Wokko/commit/5d9fe041047ba92bfe381480f6e5ed028e249378))


### Fixes

* **kitshn:** answer the version check before login ([473e696](https://github.com/Serraniel/Wokko/commit/473e696cb4a1cfb8a26093189d04a9026f51d97a))
* **kitshn:** repair the test broken by the version check change ([b9f88c2](https://github.com/Serraniel/Wokko/commit/b9f88c2d8c296e412973b65a4d2d4ae7a099c951))
* **ui:** fix issues from a full browser and accessibility audit ([8bbd08d](https://github.com/Serraniel/Wokko/commit/8bbd08db535a468bfafe74ed3fbf302c8c81b697))

## 0.1.0

The first foundation, later replaced by a full rewrite. It was never used in production.
