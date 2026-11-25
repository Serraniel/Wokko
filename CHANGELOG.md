# Changelog

## [0.5.1](https://github.com/Serraniel/Wokko/compare/v0.5.0...v0.5.1) (2026-11-25)


### Dependencies

* bump @lucide/svelte from 1.70.0 to 1.76.0 in /frontend ([#162](https://github.com/Serraniel/Wokko/issues/162)) ([252e316](https://github.com/Serraniel/Wokko/commit/252e316fb96a308a28e391c93672bc01a0f6a072))
* bump @lucide/svelte from 1.76.0 to 1.77.0 in /frontend ([#165](https://github.com/Serraniel/Wokko/issues/165)) ([4462cde](https://github.com/Serraniel/Wokko/commit/4462cdef5a1108563708421ac0068983ba130fe7))
* bump golang.org/x/net from 0.62.0 to 0.63.0 in /backend ([#160](https://github.com/Serraniel/Wokko/issues/160)) ([252e316](https://github.com/Serraniel/Wokko/commit/252e316fb96a308a28e391c93672bc01a0f6a072))
* bump modernc.org/sqlite from 1.62.0 to 1.63.0 in /backend ([#164](https://github.com/Serraniel/Wokko/issues/164)) ([c2fb44d](https://github.com/Serraniel/Wokko/commit/c2fb44dc2f980cf5c605d6a9cd7e1b1f23fcafa7))
* bump svelte from 5.57.3 to 5.58.0 in /frontend ([#161](https://github.com/Serraniel/Wokko/issues/161)) ([252e316](https://github.com/Serraniel/Wokko/commit/252e316fb96a308a28e391c93672bc01a0f6a072))

## [0.5.0](https://github.com/Serraniel/Wokko/compare/v0.4.0...v0.5.0) (2026-11-20)


### Features

* **auth:** admins create accounts with a one-time setup link ([#152](https://github.com/Serraniel/Wokko/issues/152)) ([fd78070](https://github.com/Serraniel/Wokko/commit/fd7807030294ffa0c9516b02db2a7fe4aa667ed1)), closes [#150](https://github.com/Serraniel/Wokko/issues/150)
* **auth:** optional mail for invites, setup links and forgotten passwords ([#153](https://github.com/Serraniel/Wokko/issues/153)) ([3746a87](https://github.com/Serraniel/Wokko/commit/3746a87f750b8d094d3fd306d5bacda4a94356ba)), closes [#151](https://github.com/Serraniel/Wokko/issues/151)
* **i18n:** switch language and theme without an account, and the theme in one click ([#134](https://github.com/Serraniel/Wokko/issues/134)) ([4ab1727](https://github.com/Serraniel/Wokko/commit/4ab1727ef8bd46c6178c3ad23228eeddaf543568)), closes [#131](https://github.com/Serraniel/Wokko/issues/131)
* **ingredients:** add everyday staples to the starter catalog ([#140](https://github.com/Serraniel/Wokko/issues/140)) ([c381992](https://github.com/Serraniel/Wokko/commit/c381992b45a5bfdca30a7eb2a67e6748cf040604)), closes [#126](https://github.com/Serraniel/Wokko/issues/126)
* **ingredients:** more names for starter foods from Wikidata ([#115](https://github.com/Serraniel/Wokko/issues/115)) ([67d38d8](https://github.com/Serraniel/Wokko/commit/67d38d805e7125eb466459c536c3fc902d9a7e80))
* **mcp:** let assistants fill in missing nutrition values ([#139](https://github.com/Serraniel/Wokko/issues/139)) ([31a1f99](https://github.com/Serraniel/Wokko/commit/31a1f992781cc62ce18d24db05a09e652d1b6354)), closes [#125](https://github.com/Serraniel/Wokko/issues/125)
* **mcp:** set recipe photos without a public address ([#141](https://github.com/Serraniel/Wokko/issues/141)) ([9408cc5](https://github.com/Serraniel/Wokko/commit/9408cc5fa7e66a12416779f0c1f38c7fba4269ad)), closes [#128](https://github.com/Serraniel/Wokko/issues/128)
* **plan:** cook a linked recipe instead of buying it ([#148](https://github.com/Serraniel/Wokko/issues/148)) ([ae6cbdf](https://github.com/Serraniel/Wokko/commit/ae6cbdf9f60216ca1e634936cf3af274ce7deb55))
* **plan:** look up a food's nutrition values offline ([#138](https://github.com/Serraniel/Wokko/issues/138)) ([2f58aa5](https://github.com/Serraniel/Wokko/commit/2f58aa5bad5f27a394fc6b0ac1623d422a8e658f)), closes [#124](https://github.com/Serraniel/Wokko/issues/124)
* **recipes:** count linked recipes for nutrition, allergens, kitshn and imports ([#149](https://github.com/Serraniel/Wokko/issues/149)) ([b390cfe](https://github.com/Serraniel/Wokko/commit/b390cfe7b8e6fb64ebddfb4964983d7eaa3114e9)), closes [#136](https://github.com/Serraniel/Wokko/issues/136)
* **recipes:** import a recipe file from New recipe ([#143](https://github.com/Serraniel/Wokko/issues/143)) ([c9b9348](https://github.com/Serraniel/Wokko/commit/c9b93489ed4926c328d35b6e1d3fee1647d5ba68)), closes [#121](https://github.com/Serraniel/Wokko/issues/121)
* **recipes:** link an ingredient to the recipe it's cooked from ([#147](https://github.com/Serraniel/Wokko/issues/147)) ([be628bc](https://github.com/Serraniel/Wokko/commit/be628bc7a1be17a89adb00720daf17294a509c06))
* **sharing:** link previews for shared recipes and every other page ([#146](https://github.com/Serraniel/Wokko/issues/146)) ([2cc6df8](https://github.com/Serraniel/Wokko/commit/2cc6df85bbefdfd8e3baec18c47d409ee07cdcb8)), closes [#130](https://github.com/Serraniel/Wokko/issues/130)
* **sharing:** make clear which links need a login ([#135](https://github.com/Serraniel/Wokko/issues/135)) ([aca858c](https://github.com/Serraniel/Wokko/commit/aca858c593943ff90806f477605c7164912d811d)), closes [#129](https://github.com/Serraniel/Wokko/issues/129)
* **ui:** blur the page slightly behind sheets and dialogs ([#133](https://github.com/Serraniel/Wokko/issues/133)) ([2796402](https://github.com/Serraniel/Wokko/commit/27964026e78391330e1a206b53d668226ebb116b)), closes [#119](https://github.com/Serraniel/Wokko/issues/119)
* **ui:** integrations as an overview of cards with a dialog each ([#145](https://github.com/Serraniel/Wokko/issues/145)) ([6d3c705](https://github.com/Serraniel/Wokko/commit/6d3c7050f117036f562d061970ff373130111ff3))


### Fixes

* **ingredients:** find known foods inside longer names ([#137](https://github.com/Serraniel/Wokko/issues/137)) ([c57a0d5](https://github.com/Serraniel/Wokko/commit/c57a0d5b945c8ed9e5607899f6c143dccc97297c)), closes [#127](https://github.com/Serraniel/Wokko/issues/127)
* **mcp:** keep idle assistant apps listed so they can be disconnected ([#144](https://github.com/Serraniel/Wokko/issues/144)) ([a8c3fd3](https://github.com/Serraniel/Wokko/commit/a8c3fd3b8c6fd649b87bfde72659fa9c6fff0f28))
* **mcp:** show Wokko's logo for the connector in assistants ([#123](https://github.com/Serraniel/Wokko/issues/123)) ([bf90bbd](https://github.com/Serraniel/Wokko/commit/bf90bbdfdfaa5acf23e10a539e4872b492711ead))
* **recipes:** stop concurrent requests from crashing search ([#142](https://github.com/Serraniel/Wokko/issues/142)) ([ceee9cf](https://github.com/Serraniel/Wokko/commit/ceee9cf619aa530145e31e154adad24d2633dd09))
* **ui:** show the pointer cursor on everything clickable ([#132](https://github.com/Serraniel/Wokko/issues/132)) ([80a7775](https://github.com/Serraniel/Wokko/commit/80a77750658197a53b01e5f1a6ca7852ec1e3867)), closes [#118](https://github.com/Serraniel/Wokko/issues/118)

## [0.4.0](https://github.com/Serraniel/Wokko/compare/v0.3.0...v0.4.0) (2026-11-14)


### Features

* **ingredients:** starter catalog updates reach existing databases ([#110](https://github.com/Serraniel/Wokko/issues/110)) ([a97b89e](https://github.com/Serraniel/Wokko/commit/a97b89eda647d16cf779850773fdd5caf9a9b285)), closes [#107](https://github.com/Serraniel/Wokko/issues/107)
* **knowledge:** built-in kitchen knowledge for every recipe ([#108](https://github.com/Serraniel/Wokko/issues/108)) ([57cd0a5](https://github.com/Serraniel/Wokko/commit/57cd0a5a91036d4874a2d8efa7bcb2f19a9467c5)), closes [#105](https://github.com/Serraniel/Wokko/issues/105)
* **knowledge:** hints while editing a recipe ([#113](https://github.com/Serraniel/Wokko/issues/113)) ([025d415](https://github.com/Serraniel/Wokko/commit/025d415deff54d1dc73ad31f984d0f5c5b8532c9)), closes [#105](https://github.com/Serraniel/Wokko/issues/105)
* **knowledge:** set the kind of a food Wokko doesn't recognise ([#111](https://github.com/Serraniel/Wokko/issues/111)) ([6712fba](https://github.com/Serraniel/Wokko/commit/6712fba9821af1e8e9c0620d15f27d32ca3f90a2)), closes [#105](https://github.com/Serraniel/Wokko/issues/105)
* **knowledge:** substitutes for ingredients in a recipe ([#112](https://github.com/Serraniel/Wokko/issues/112)) ([2c135a1](https://github.com/Serraniel/Wokko/commit/2c135a1b2adc4df8f64efe438d6104b0b19e88b4)), closes [#105](https://github.com/Serraniel/Wokko/issues/105)
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
