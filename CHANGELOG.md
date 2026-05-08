# Changelog

## [0.42.1](https://github.com/grafana/plugin-validator/compare/plugin-validator/v0.42.0...plugin-validator/v0.42.1) (2026-05-08)


### 🐛 Bug Fixes

* **ci:** handle release-please prefixed tags in goreleaser publish ([#575](https://github.com/grafana/plugin-validator/issues/575)) ([e33c65a](https://github.com/grafana/plugin-validator/commit/e33c65a023b9fba37f347b9b32b5b5bb692057f9))

## [0.42.0](https://github.com/grafana/plugin-validator/compare/plugin-validator/v0.41.0...plugin-validator/v0.42.0) (2026-05-08)


### 🎉 Features

* Add flag to select specific analyzer to run ([#264](https://github.com/grafana/plugin-validator/issues/264)) ([882de9a](https://github.com/grafana/plugin-validator/commit/882de9a14a39bc67e1948022d0e6ba75343ba303))
* add ghaOutput and jsonOutput CLI flags ([#455](https://github.com/grafana/plugin-validator/issues/455)) ([b4c9a33](https://github.com/grafana/plugin-validator/commit/b4c9a33bb69233ed88cb3bc7c7aaab5e35484772))
* add output in GitHub Actions workflow commands format ([#452](https://github.com/grafana/plugin-validator/issues/452)) ([7a8dcf3](https://github.com/grafana/plugin-validator/commit/7a8dcf36fe36b5ec05dca91907af9e6b9b49ae23))
* Change metadatavalid default severity ([#271](https://github.com/grafana/plugin-validator/issues/271)) ([32dae9d](https://github.com/grafana/plugin-validator/commit/32dae9d754fe3b23e581a1251f7c5240f0516686))
* check screenshot image mimetype and extension ([#491](https://github.com/grafana/plugin-validator/issues/491)) ([45b1c83](https://github.com/grafana/plugin-validator/commit/45b1c83634c09dae49ace4eec2a0b030f20b18a4))
* check screenshot image type ([#278](https://github.com/grafana/plugin-validator/issues/278)) ([aeea69a](https://github.com/grafana/plugin-validator/commit/aeea69a2cca76d2f5dd5adec7e1adc93123609be))
* mcp server for validate plugin ([#518](https://github.com/grafana/plugin-validator/issues/518)) ([4de19a1](https://github.com/grafana/plugin-validator/commit/4de19a19c09adbc3a73d2da5e7715b334e91302c))
* support Anthropic and OpenAI as LLM providers ([#541](https://github.com/grafana/plugin-validator/issues/541)) ([06099b1](https://github.com/grafana/plugin-validator/commit/06099b164d8e36fa6e62bbab7d9e1fec4f16eb5b))
* switch to Anthropic Sonnet 4.6 with prompt caching ([#548](https://github.com/grafana/plugin-validator/issues/548)) ([9d7e161](https://github.com/grafana/plugin-validator/commit/9d7e16182094a93d33adb029b65f834f1da9407d))
* validate grafanaDep with semver ([#515](https://github.com/grafana/plugin-validator/issues/515)) ([b0f0069](https://github.com/grafana/plugin-validator/commit/b0f0069229d8f4366f3c9f6dfff98b5275ac59d7))
* warn if readme plugin contains a comment. ([#253](https://github.com/grafana/plugin-validator/issues/253)) ([8c479a1](https://github.com/grafana/plugin-validator/commit/8c479a10e88cf6b8374360dfa86734496c7365c3))


### 🐛 Bug Fixes

* **ci:** use github hosted runners for npm publish ([#462](https://github.com/grafana/plugin-validator/issues/462)) ([e8aab49](https://github.com/grafana/plugin-validator/commit/e8aab4920ddc1a0cbc923994923e69cd8a8e39cf))
* **deps:** update dependency tar to v7.5.10 [security] ([#531](https://github.com/grafana/plugin-validator/issues/531)) ([df396d5](https://github.com/grafana/plugin-validator/commit/df396d5bc86c1a06b04c8868b1fe9f68ac55bcb4))
* **deps:** update go dependencies ([#414](https://github.com/grafana/plugin-validator/issues/414)) ([72734bf](https://github.com/grafana/plugin-validator/commit/72734bfb45ee06e2ae28d4955c83e0932dcadf33))
* **deps:** update go dependencies ([#426](https://github.com/grafana/plugin-validator/issues/426)) ([e275f16](https://github.com/grafana/plugin-validator/commit/e275f16a824f54e1ad804c9b1261dbbd9ca652dd))
* **deps:** update go dependencies ([#468](https://github.com/grafana/plugin-validator/issues/468)) ([9322b24](https://github.com/grafana/plugin-validator/commit/9322b24b97ae54126217e9c858d85f02072fb23a))
* **deps:** update go dependencies ([#476](https://github.com/grafana/plugin-validator/issues/476)) ([cbcb1e5](https://github.com/grafana/plugin-validator/commit/cbcb1e527bc9974ca0905d57266873828de48f67))
* **deps:** update go dependencies ([#482](https://github.com/grafana/plugin-validator/issues/482)) ([8f27d4a](https://github.com/grafana/plugin-validator/commit/8f27d4a5a306402be8fb0be7a92addafd36b11d8))
* **deps:** update go dependencies (major) ([#415](https://github.com/grafana/plugin-validator/issues/415)) ([6c73dba](https://github.com/grafana/plugin-validator/commit/6c73dba9d6a77bd6e43ad313a8e5a7432b843993))
* **deps:** update module github.com/modelcontextprotocol/go-sdk to v1.3.1 [security] ([#528](https://github.com/grafana/plugin-validator/issues/528)) ([5a442bb](https://github.com/grafana/plugin-validator/commit/5a442bbd1d9911c9d3a3769865fb3aff783c2725))
* **deps:** update module github.com/modelcontextprotocol/go-sdk to v1.4.1 [security] ([#545](https://github.com/grafana/plugin-validator/issues/545)) ([144c6bb](https://github.com/grafana/plugin-validator/commit/144c6bbad7d06b2e53b52658a00cf116a73d5540))
* **deps:** update module golang.org/x/crypto to v0.45.0 [security] ([#470](https://github.com/grafana/plugin-validator/issues/470)) ([25e1974](https://github.com/grafana/plugin-validator/commit/25e19744794669201b2c2ead2c6d07942233418f))
* **deps:** update module google.golang.org/api to v0.252.0 ([#424](https://github.com/grafana/plugin-validator/issues/424)) ([e2549a6](https://github.com/grafana/plugin-validator/commit/e2549a635341a8242c9b688a0c4eeab9b1a6ddc3))
* **deps:** update module google.golang.org/api to v0.253.0 ([#459](https://github.com/grafana/plugin-validator/issues/459)) ([54fcc74](https://github.com/grafana/plugin-validator/commit/54fcc744d7d73b2d9ac559763187cb91524f465a))
* go-manifest false positives from node_modules in sourceCodeUri scans ([#556](https://github.com/grafana/plugin-validator/issues/556)) ([820579f](https://github.com/grafana/plugin-validator/commit/820579fe80f2bd7028fa36c9a4a530476783a5a5))
* make LLM review resilient to individual question failures ([#550](https://github.com/grafana/plugin-validator/issues/550)) ([c653140](https://github.com/grafana/plugin-validator/commit/c653140234a2b550081649fde1eb833531d44336))
* needs to check all the links ([#351](https://github.com/grafana/plugin-validator/issues/351)) ([3a1c2aa](https://github.com/grafana/plugin-validator/commit/3a1c2aa247243b174beba5a3ba6a03488eef9001))
* return ok when screenshots don't exist ([#108](https://github.com/grafana/plugin-validator/issues/108)) ([090d08b](https://github.com/grafana/plugin-validator/commit/090d08b6ee400ca013fdbabb8434c78c92708ae3))
* **screenshots:** handle malformed screenshots format gracefully ([#346](https://github.com/grafana/plugin-validator/issues/346)) ([5d6d266](https://github.com/grafana/plugin-validator/commit/5d6d2665cfea3bbb3acc863d3aef85efe7d1cfbe))
* sdkusage now checking plugin's version age for warnings and errors ([#486](https://github.com/grafana/plugin-validator/issues/486)) ([31c45a4](https://github.com/grafana/plugin-validator/commit/31c45a453f69cf1c3b6e4daecc5bcff994c27bc4))
* use the new schema url ([#196](https://github.com/grafana/plugin-validator/issues/196)) ([f66c53d](https://github.com/grafana/plugin-validator/commit/f66c53d966b57fb6a507866655cf988ace6b49e9))


### 📝 Documentation

* improve release documentation ([#562](https://github.com/grafana/plugin-validator/issues/562)) ([ebc624c](https://github.com/grafana/plugin-validator/commit/ebc624c1e63fdb7f152fafe7cd768bf8419cb284))
* Validate multi-page docs ([#560](https://github.com/grafana/plugin-validator/issues/560)) ([d49fb5f](https://github.com/grafana/plugin-validator/commit/d49fb5f2b15a8e9384869346a684e45a3a942b15))


### ♻️ Code Refactoring

* add output package and marshaler interface ([#451](https://github.com/grafana/plugin-validator/issues/451)) ([1ff3cec](https://github.com/grafana/plugin-validator/commit/1ff3cec4778e328361a555e5142bb7279b0a712a))


### ✅ Tests

* Fix broken readme for old plugin ([#234](https://github.com/grafana/plugin-validator/issues/234)) ([ac28776](https://github.com/grafana/plugin-validator/commit/ac287766d19dd88569c7adbf2e678dcfe3423bc5))


### 🏗️ Builds

* allow building Docker image on Mac OS ARM ([#538](https://github.com/grafana/plugin-validator/issues/538)) ([5d514d0](https://github.com/grafana/plugin-validator/commit/5d514d0036b732e31e13562ba158556ad9120939))
* Install semgrep and gosec in build stage for CI tests ([#180](https://github.com/grafana/plugin-validator/issues/180)) ([12d52dc](https://github.com/grafana/plugin-validator/commit/12d52dc7cdcc9c8976e9b54ee53742f76b08990b))


### 🤖 Continuous Integrations

* migrate releases to release-please ([#571](https://github.com/grafana/plugin-validator/issues/571)) ([dcd8302](https://github.com/grafana/plugin-validator/commit/dcd83024f3007f7bd740ff1c47e857e034d01549))
* **npm:** publish using oidc instead of tokens ([#383](https://github.com/grafana/plugin-validator/issues/383)) ([56f7588](https://github.com/grafana/plugin-validator/commit/56f75882df3b996c55fea13c18d51ddd10b625f8))
* switch to self-hosted runners ([#456](https://github.com/grafana/plugin-validator/issues/456)) ([7be5deb](https://github.com/grafana/plugin-validator/commit/7be5debe7d9e2061c382b443ab1b149c59b87d8b))
* Update GitHub Action to build Docker container ([#181](https://github.com/grafana/plugin-validator/issues/181)) ([cb55801](https://github.com/grafana/plugin-validator/commit/cb5580130dcb6e0e4f2b1ad4a1a2ccb8076e09a3))


### 🔧 Chores

* address goreleaser deprecation notices ([#461](https://github.com/grafana/plugin-validator/issues/461)) ([b3cea63](https://github.com/grafana/plugin-validator/commit/b3cea63ceeed2ec6bf28c71576db467bc0ae03bb))
* Build: Use Go 1.20.X ([#118](https://github.com/grafana/plugin-validator/issues/118)) ([cfd81d9](https://github.com/grafana/plugin-validator/commit/cfd81d9bb1b4f66541351729a3aa6a50950a3d38))
* Bump to Go 1.24 ([#313](https://github.com/grafana/plugin-validator/issues/313)) ([182ad2e](https://github.com/grafana/plugin-validator/commit/182ad2e5954d984a9e80b93a658216d7dc6ce77a))
* change prev tag finding step for mcp ([#523](https://github.com/grafana/plugin-validator/issues/523)) ([7b6ddf9](https://github.com/grafana/plugin-validator/commit/7b6ddf9a1fd80fc8ae2b8ccb1d778e1ec79d1963))
* **ci:** migrate to docker-build-push-image action ([#558](https://github.com/grafana/plugin-validator/issues/558)) ([7185226](https://github.com/grafana/plugin-validator/commit/7185226ebaa11c5b86296a2aff98cc7d76638d63))
* clean up unnecessary report all (part 2) ([#373](https://github.com/grafana/plugin-validator/issues/373)) ([cdca30e](https://github.com/grafana/plugin-validator/commit/cdca30e11af8e2053eaa8d3bbbcf4f5723c61176))
* clean up unnecessary report all (part 3) ([#378](https://github.com/grafana/plugin-validator/issues/378)) ([7a692c9](https://github.com/grafana/plugin-validator/commit/7a692c9c964340d6bae355f18e55ed761eee2fa0))
* **deps:** Bump actions/checkout from 4.2.2 to 5.0.0 ([#366](https://github.com/grafana/plugin-validator/issues/366)) ([60531b1](https://github.com/grafana/plugin-validator/commit/60531b1c5c8827ac44ce8d2cea367b52ff5b5a33))
* **deps:** Bump actions/create-github-app-token from 2.0.6 to 2.1.1 ([#364](https://github.com/grafana/plugin-validator/issues/364)) ([28f126d](https://github.com/grafana/plugin-validator/commit/28f126d6b6730c8d23814afbb2c2d150660af03e))
* **deps:** Bump actions/create-github-app-token from 2.1.1 to 2.1.4 ([#390](https://github.com/grafana/plugin-validator/issues/390)) ([7ea187b](https://github.com/grafana/plugin-validator/commit/7ea187bd38638ef5d79abca5d21dd452111bb9e0))
* **deps:** Bump actions/setup-go from 5.5.0 to 6.0.0 ([#392](https://github.com/grafana/plugin-validator/issues/392)) ([4918cac](https://github.com/grafana/plugin-validator/commit/4918cac3fc695f087e0619851ae3eb1807f9363f))
* **deps:** Bump actions/setup-node from 4.4.0 to 5.0.0 ([#391](https://github.com/grafana/plugin-validator/issues/391)) ([0929d47](https://github.com/grafana/plugin-validator/commit/0929d47962db381d522306bb4f893ffb34b27770))
* **deps:** bump build-push-to-dockerhub to v0.4.0, add comment for renovate ([#458](https://github.com/grafana/plugin-validator/issues/458)) ([ce85bde](https://github.com/grafana/plugin-validator/commit/ce85bde2e3012b7860d3f2f8597044becfb608e0))
* **deps:** Bump github.com/bmatcuk/doublestar/v4 from 4.8.1 to 4.9.1 ([#354](https://github.com/grafana/plugin-validator/issues/354)) ([957b522](https://github.com/grafana/plugin-validator/commit/957b52207bb4b04ae66eb86de042d2f1d3726c55))
* **deps:** bump github.com/docker/docker to v28.5.2 [security] ([#567](https://github.com/grafana/plugin-validator/issues/567)) ([175afff](https://github.com/grafana/plugin-validator/commit/175afff4d2879064edd74df87389670ba65ecef9))
* **deps:** Bump github.com/jarcoal/httpmock from 1.4.0 to 1.4.1 ([#361](https://github.com/grafana/plugin-validator/issues/361)) ([3883fab](https://github.com/grafana/plugin-validator/commit/3883fabc39d53b74955c0070cd04bdd78fe68514))
* **deps:** Bump github.com/stretchr/testify from 1.10.0 to 1.11.1 ([#359](https://github.com/grafana/plugin-validator/issues/359)) ([aa54b2f](https://github.com/grafana/plugin-validator/commit/aa54b2f1227e6b853fbdb9fcd78a2d92f3b0c228))
* **deps:** Bump golang.org/x/crypto from 0.40.0 to 0.41.0 ([#362](https://github.com/grafana/plugin-validator/issues/362)) ([9159fa6](https://github.com/grafana/plugin-validator/commit/9159fa6d75c43bfb7bb9a76a025da70f33002fa3))
* **deps:** Bump golang.org/x/crypto from 0.41.0 to 0.42.0 ([#388](https://github.com/grafana/plugin-validator/issues/388)) ([660b0b7](https://github.com/grafana/plugin-validator/commit/660b0b7ad912d075dba07c413b6affec04fc948d))
* **deps:** Bump golang.org/x/mod from 0.25.0 to 0.26.0 ([#353](https://github.com/grafana/plugin-validator/issues/353)) ([f720abb](https://github.com/grafana/plugin-validator/commit/f720abb05a69755a6856a955bb0a771ad16cb7a3))
* **deps:** Bump golang.org/x/mod from 0.26.0 to 0.27.0 ([#363](https://github.com/grafana/plugin-validator/issues/363)) ([606c8ae](https://github.com/grafana/plugin-validator/commit/606c8ae56f41e7c1d4d73c2031d45074f42a27ce))
* **deps:** Bump golang.org/x/mod from 0.27.0 to 0.28.0 ([#386](https://github.com/grafana/plugin-validator/issues/386)) ([b760259](https://github.com/grafana/plugin-validator/commit/b7602592803f35e260f37d9554d6ecfa32a93d7e))
* **deps:** Bump google.golang.org/api from 0.235.0 to 0.239.0 ([#347](https://github.com/grafana/plugin-validator/issues/347)) ([d73dea8](https://github.com/grafana/plugin-validator/commit/d73dea899fdae3ac143bfa1510a30188de36f7e4))
* **deps:** Bump google.golang.org/api from 0.239.0 to 0.244.0 ([#355](https://github.com/grafana/plugin-validator/issues/355)) ([9578824](https://github.com/grafana/plugin-validator/commit/9578824b21a14260022b783250d67e0e712148a4))
* **deps:** Bump google.golang.org/api from 0.244.0 to 0.248.0 ([#360](https://github.com/grafana/plugin-validator/issues/360)) ([5f79e6b](https://github.com/grafana/plugin-validator/commit/5f79e6b655ae1858db1026b520ae6b57f30eea89))
* **deps:** Bump google.golang.org/api from 0.248.0 to 0.251.0 ([#387](https://github.com/grafana/plugin-validator/issues/387)) ([2d69294](https://github.com/grafana/plugin-validator/commit/2d692945e260da620c519ac0717bf583234fbd6e))
* **deps:** Bump goreleaser/goreleaser-action from 6.3.0 to 6.4.0 ([#367](https://github.com/grafana/plugin-validator/issues/367)) ([2e91d7d](https://github.com/grafana/plugin-validator/commit/2e91d7d02fe5e509d26bb1163b981cacdedf7a63))
* **deps:** Bump grafana/shared-workflows from 1.2.1 to 1.3.0 ([#365](https://github.com/grafana/plugin-validator/issues/365)) ([965c9f1](https://github.com/grafana/plugin-validator/commit/965c9f117c34ef7ea34c220e659d868269b1aa1d))
* **deps:** Bump tar from 7.4.3 to 7.5.1 ([#389](https://github.com/grafana/plugin-validator/issues/389)) ([1da3aed](https://github.com/grafana/plugin-validator/commit/1da3aedbc518a3ee2135fe2166cf3a2049421acf))
* **deps:** set up renovate for Go version in go.mod and Dockerfile ([#536](https://github.com/grafana/plugin-validator/issues/536)) ([9a70fee](https://github.com/grafana/plugin-validator/commit/9a70fee073b02044c8a2fce42ce4deec1c531eff))
* **deps:** update actions/setup-node action to v6 ([#434](https://github.com/grafana/plugin-validator/issues/434)) ([b33dbea](https://github.com/grafana/plugin-validator/commit/b33dbeaa11bfe97f6453a5a985bae9fc80d1eb5d))
* **deps:** update alpine docker tag to v3.22 ([#427](https://github.com/grafana/plugin-validator/issues/427)) ([42e58ef](https://github.com/grafana/plugin-validator/commit/42e58efcf21bd63aaec0fed034f5657f8351fa80))
* **deps:** update dependency go to v1.25.2 ([#421](https://github.com/grafana/plugin-validator/issues/421)) ([d3f301f](https://github.com/grafana/plugin-validator/commit/d3f301f7a8712fd9bd5e9997166d2089e3b5b8d6))
* **deps:** update dependency go to v1.25.3 ([#432](https://github.com/grafana/plugin-validator/issues/432)) ([3cd348c](https://github.com/grafana/plugin-validator/commit/3cd348cd81dcb1a3d97df7b5fe49639f2b2addc1))
* **deps:** update dependency tar to v7.5.11 [security] ([#540](https://github.com/grafana/plugin-validator/issues/540)) ([822b22b](https://github.com/grafana/plugin-validator/commit/822b22b4ad3efd9ed4d9b818720e7d06d2e6a9f6))
* **deps:** update dependency tar to v7.5.2 [security] ([#444](https://github.com/grafana/plugin-validator/issues/444)) ([89f94c3](https://github.com/grafana/plugin-validator/commit/89f94c32f4acff2253f5e9dd17bbbe5e77070fd0))
* **deps:** update dependency tar to v7.5.3 [security] ([#488](https://github.com/grafana/plugin-validator/issues/488)) ([fb51260](https://github.com/grafana/plugin-validator/commit/fb51260d94e67d6b585d075e077ae53e51f98594))
* **deps:** update dependency tar to v7.5.4 [security] ([#492](https://github.com/grafana/plugin-validator/issues/492)) ([3da2dc0](https://github.com/grafana/plugin-validator/commit/3da2dc0468bb5f45d0cf51b89524c701b3995b18))
* **deps:** update dependency tar to v7.5.7 [security] ([#505](https://github.com/grafana/plugin-validator/issues/505)) ([1060d2c](https://github.com/grafana/plugin-validator/commit/1060d2c17ae999e04fc1dc15b91413873fa13a40))
* **deps:** update dependency tar to v7.5.8 [security] ([#521](https://github.com/grafana/plugin-validator/issues/521)) ([e968649](https://github.com/grafana/plugin-validator/commit/e968649dc9194cc974526ca8893d10c551573c5b))
* **deps:** update github actions ([#480](https://github.com/grafana/plugin-validator/issues/480)) ([2d7f847](https://github.com/grafana/plugin-validator/commit/2d7f847f21c069f99c03c6c4599cfa634ccf8ffe))
* **deps:** update github actions ([#484](https://github.com/grafana/plugin-validator/issues/484)) ([5967bb5](https://github.com/grafana/plugin-validator/commit/5967bb5de75027f8412f51e2dc8a41f2d861e8d2))
* **deps:** update github actions ([#525](https://github.com/grafana/plugin-validator/issues/525)) ([d9d6d53](https://github.com/grafana/plugin-validator/commit/d9d6d53cb688c14eaf1877a69db15e7391c3e887))
* **deps:** update golang docker tag to v1.25 ([#440](https://github.com/grafana/plugin-validator/issues/440)) ([e8021b8](https://github.com/grafana/plugin-validator/commit/e8021b8174608e67c913c4f46e0399f4e4744b8b))
* **deps:** update golang:1.25-alpine3.21 docker digest to 3289aac ([#463](https://github.com/grafana/plugin-validator/issues/463)) ([149657d](https://github.com/grafana/plugin-validator/commit/149657dd07871183839f5ff8007ce430a99b2416))
* **deps:** update golang:1.25-alpine3.21 docker digest to 8f507c4 ([#460](https://github.com/grafana/plugin-validator/issues/460)) ([e71d25b](https://github.com/grafana/plugin-validator/commit/e71d25b8c0e1cb4f7d946197973e7145486fa48b))
* **deps:** update golang:1.25-alpine3.22 docker digest to fa3380a ([#493](https://github.com/grafana/plugin-validator/issues/493)) ([f0ec653](https://github.com/grafana/plugin-validator/commit/f0ec653c673fffdbe2c04255ee80e21a871b2457))
* **deps:** update grafana/shared-workflows/ action to ([#413](https://github.com/grafana/plugin-validator/issues/413)) ([174f652](https://github.com/grafana/plugin-validator/commit/174f652929acd3d9c42853ff0e96300be86bacb1))
* **deps:** update grafana/shared-workflows/ action to ([#416](https://github.com/grafana/plugin-validator/issues/416)) ([984e298](https://github.com/grafana/plugin-validator/commit/984e2980693a7de243c79a8b7b3d5f934ff42558))
* **deps:** update grafana/shared-workflows/ action to ([#420](https://github.com/grafana/plugin-validator/issues/420)) ([0a3aa5f](https://github.com/grafana/plugin-validator/commit/0a3aa5f37bb83986ddcc17e32caaa8889d8da527))
* **deps:** update grafana/shared-workflows/ action to ([#422](https://github.com/grafana/plugin-validator/issues/422)) ([d9e6aad](https://github.com/grafana/plugin-validator/commit/d9e6aad6109beb352e2e56033b1f1eaceb166d3a))
* **deps:** update grafana/shared-workflows/ action to ([#423](https://github.com/grafana/plugin-validator/issues/423)) ([ff95689](https://github.com/grafana/plugin-validator/commit/ff95689fd8f3ab24844b66cd88c9ebf1d729f196))
* **deps:** update grafana/shared-workflows/ action to ([#425](https://github.com/grafana/plugin-validator/issues/425)) ([51c1c4a](https://github.com/grafana/plugin-validator/commit/51c1c4a911a4aeaf30e189a51abf28695093e776))
* **deps:** update grafana/shared-workflows/ action to ([#428](https://github.com/grafana/plugin-validator/issues/428)) ([2484218](https://github.com/grafana/plugin-validator/commit/2484218ce0048dfa8c95a4c59bdca1b11bf2ce5e))
* **deps:** update grafana/shared-workflows/ action to ([#429](https://github.com/grafana/plugin-validator/issues/429)) ([9fbfba0](https://github.com/grafana/plugin-validator/commit/9fbfba0063381b11a4a04e55228df335559da890))
* **deps:** update grafana/shared-workflows/ action to ([#430](https://github.com/grafana/plugin-validator/issues/430)) ([a49848f](https://github.com/grafana/plugin-validator/commit/a49848f0fcdc5fe7653c28697322519728fe3782))
* **deps:** update grafana/shared-workflows/ action to ([#433](https://github.com/grafana/plugin-validator/issues/433)) ([3c2b5c5](https://github.com/grafana/plugin-validator/commit/3c2b5c512d7d502657eba4625f7297f8a16bd8ff))
* **deps:** update grafana/shared-workflows/ action to ([#435](https://github.com/grafana/plugin-validator/issues/435)) ([44f0efb](https://github.com/grafana/plugin-validator/commit/44f0efbc39629887188ec54291863311eceff296))
* **deps:** update grafana/shared-workflows/ action to ([#439](https://github.com/grafana/plugin-validator/issues/439)) ([d8c80dd](https://github.com/grafana/plugin-validator/commit/d8c80ddb4c15fafed082f540846607eadffb18d7))
* **deps:** update grafana/shared-workflows/ action to ([#442](https://github.com/grafana/plugin-validator/issues/442)) ([d5aa1c4](https://github.com/grafana/plugin-validator/commit/d5aa1c469fb047a3ebd4f35e0c2f47135601bb8a))
* **deps:** update grafana/shared-workflows/ action to ([#443](https://github.com/grafana/plugin-validator/issues/443)) ([f25ba40](https://github.com/grafana/plugin-validator/commit/f25ba40203f13758ca475d768866ddc1d5bd1afd))
* **deps:** update grafana/shared-workflows/ action to ([#445](https://github.com/grafana/plugin-validator/issues/445)) ([79e5be4](https://github.com/grafana/plugin-validator/commit/79e5be42d2a4fd23f944dc83a2201c7930575736))
* **deps:** update grafana/shared-workflows/ action to ([#446](https://github.com/grafana/plugin-validator/issues/446)) ([440e970](https://github.com/grafana/plugin-validator/commit/440e9708d8ec3d167e8adeefb7f13fc71736025e))
* **deps:** update grafana/shared-workflows/ action to ([#447](https://github.com/grafana/plugin-validator/issues/447)) ([24f97ee](https://github.com/grafana/plugin-validator/commit/24f97eeaaa0a5e942974f13c5f424b5a4e4bc152))
* **deps:** update grafana/shared-workflows/ action to ([#448](https://github.com/grafana/plugin-validator/issues/448)) ([0859fab](https://github.com/grafana/plugin-validator/commit/0859fab4bfa26f8a84212807b9a9c3ee7fc58100))
* **deps:** update grafana/shared-workflows/ action to ([#449](https://github.com/grafana/plugin-validator/issues/449)) ([7357b6e](https://github.com/grafana/plugin-validator/commit/7357b6e0413f667b2e7743605eac57e77c2903b5))
* **deps:** update grafana/shared-workflows/build-push-to-dockerhub action to v0.4.1 ([#481](https://github.com/grafana/plugin-validator/issues/481)) ([89c590d](https://github.com/grafana/plugin-validator/commit/89c590d19da33ec18fef7cd7b4ac64f07cb204eb))
* **deps:** update module github.com/cloudflare/circl to v1.6.3 [security] ([#527](https://github.com/grafana/plugin-validator/issues/527)) ([c114da4](https://github.com/grafana/plugin-validator/commit/c114da456127d83cebf05aa3f26cdcf42621eec7))
* **deps:** update module github.com/containerd/containerd to v1.7.29 [security] ([#465](https://github.com/grafana/plugin-validator/issues/465)) ([73b3e6a](https://github.com/grafana/plugin-validator/commit/73b3e6ab4da4ca6607d543be4748525e2411eb88))
* **deps:** update module github.com/docker/cli to v29 [security] ([#532](https://github.com/grafana/plugin-validator/issues/532)) ([3b48852](https://github.com/grafana/plugin-validator/commit/3b488524084a77afb5a432e73ec7468a8f6b1586))
* **deps:** update module github.com/go-git/go-git/v5 to v5.16.5 [security] ([#506](https://github.com/grafana/plugin-validator/issues/506)) ([af2f3c2](https://github.com/grafana/plugin-validator/commit/af2f3c2ac8c965011e2328472323c4bc1a8c3aa3))
* **deps:** update module github.com/go-git/go-git/v5 to v5.17.1 [security] ([#554](https://github.com/grafana/plugin-validator/issues/554)) ([81c0331](https://github.com/grafana/plugin-validator/commit/81c0331dfd0bdabda3e600c5cda380ad1617d5f9))
* **deps:** update module github.com/go-git/go-git/v5 to v5.18.0 [security] ([#559](https://github.com/grafana/plugin-validator/issues/559)) ([731cf94](https://github.com/grafana/plugin-validator/commit/731cf94006454f94defaf2aca6463e5a4e16a556))
* **deps:** update module github.com/opencontainers/selinux to v1.13.0 [security] ([#466](https://github.com/grafana/plugin-validator/issues/466)) ([e985ef0](https://github.com/grafana/plugin-validator/commit/e985ef06cef56f260ed0af03fa08b94f495e952a))
* **deps:** update module github.com/prometheus/exporter-toolkit to v0.7.2 [security] ([#396](https://github.com/grafana/plugin-validator/issues/396)) ([ba3eefb](https://github.com/grafana/plugin-validator/commit/ba3eefbb53f5ad11f45d4a860d5cd871bc58e155))
* **deps:** update module go.opentelemetry.io/otel to v1.41.0 [security] ([#563](https://github.com/grafana/plugin-validator/issues/563)) ([48220f5](https://github.com/grafana/plugin-validator/commit/48220f5b16ebea984ebddcbcaf1b6c47221006eb))
* **deps:** update module golang.org/x/net to v0.53.0 [security] ([#568](https://github.com/grafana/plugin-validator/issues/568)) ([6a1440f](https://github.com/grafana/plugin-validator/commit/6a1440fe864090127577559715745291d72a8a19))
* **deps:** update module google.golang.org/grpc to v1.79.3 [security] ([#544](https://github.com/grafana/plugin-validator/issues/544)) ([6a8f1b9](https://github.com/grafana/plugin-validator/commit/6a8f1b9a260189068f06e12da7f82453c60056b1))
* exclude test files from validator ([#242](https://github.com/grafana/plugin-validator/issues/242)) ([a859319](https://github.com/grafana/plugin-validator/commit/a859319b2431f34d39bccc1037cd093f38a4dfe4))
* Fix bump and release action ([#329](https://github.com/grafana/plugin-validator/issues/329)) ([15f1b0c](https://github.com/grafana/plugin-validator/commit/15f1b0c8d7ee47ff23c8f50774cedc0dd0837629))
* Get npm token from vault for release ([#337](https://github.com/grafana/plugin-validator/issues/337)) ([0dad87c](https://github.com/grafana/plugin-validator/commit/0dad87c8d9af6702fc9e99948e6eb5aab98ea347))
* Harden github token permissions ([#344](https://github.com/grafana/plugin-validator/issues/344)) ([f9326a4](https://github.com/grafana/plugin-validator/commit/f9326a41140911f08e5b5ba0b804c34b1d786fbb))
* Improve validation message when source map differ from source code ([#139](https://github.com/grafana/plugin-validator/issues/139)) ([475749e](https://github.com/grafana/plugin-validator/commit/475749ed9b17373f3e5e5c25c91e3a98e6e4e9a5))
* remove binwrap dependency ([#248](https://github.com/grafana/plugin-validator/issues/248)) ([91ad559](https://github.com/grafana/plugin-validator/commit/91ad559edd911d6f8c15bdc4d6458008c0fd936e))
* remove drone and move to github action to release a docker image ([#255](https://github.com/grafana/plugin-validator/issues/255)) ([f7df6d5](https://github.com/grafana/plugin-validator/commit/f7df6d5d4cbfd87475bff2ee338caa9f964d4929))
* Update go dependencies and pin github action versions ([#316](https://github.com/grafana/plugin-validator/issues/316)) ([3283c73](https://github.com/grafana/plugin-validator/commit/3283c73b31911f5359bbfc70edba81368884eba8))
* update Grafana plugin schema ([#479](https://github.com/grafana/plugin-validator/issues/479)) ([da6d9c1](https://github.com/grafana/plugin-validator/commit/da6d9c1dd221992f0e28a10959174bf66a7fc13f))
* update Grafana plugin schema ([#489](https://github.com/grafana/plugin-validator/issues/489)) ([67e23f7](https://github.com/grafana/plugin-validator/commit/67e23f75bdc64a35c06dd3060377769ac620a00b))
* update Grafana plugin schema ([#514](https://github.com/grafana/plugin-validator/issues/514)) ([92521fc](https://github.com/grafana/plugin-validator/commit/92521fcbdcc907581bb5211c0e6a26f81864b305))
* update Grafana plugin schema ([#522](https://github.com/grafana/plugin-validator/issues/522)) ([daabb15](https://github.com/grafana/plugin-validator/commit/daabb15449df1adba1e0e6081971b885c02f528a))
* update Grafana plugin schema ([#543](https://github.com/grafana/plugin-validator/issues/543)) ([67ff0f4](https://github.com/grafana/plugin-validator/commit/67ff0f4f141be1bce7a536558be5660fca346de6))
* update Grafana plugin schema ([#557](https://github.com/grafana/plugin-validator/issues/557)) ([f896b9a](https://github.com/grafana/plugin-validator/commit/f896b9a8f016a3f7900f8ad9d847a1f7a3f00e0b))
* update Grafana plugin schema ([#566](https://github.com/grafana/plugin-validator/issues/566)) ([c9fa796](https://github.com/grafana/plugin-validator/commit/c9fa796e5b13662c8aafb05d84bd55535acab871))
* update Grafana plugin schema ([#569](https://github.com/grafana/plugin-validator/issues/569)) ([ecb9105](https://github.com/grafana/plugin-validator/commit/ecb91052f8f9877131abf061d5e93a5e2af19597))
* update packages and images ([#173](https://github.com/grafana/plugin-validator/issues/173)) ([39c038c](https://github.com/grafana/plugin-validator/commit/39c038cb4cf0edf8312ad2387f1a66bedd7f6639))
* Upgrade go releaser ([#247](https://github.com/grafana/plugin-validator/issues/247)) ([7febb5c](https://github.com/grafana/plugin-validator/commit/7febb5c04dd0a029168f97b5d1a85504e689730e))
* Use get-vault-secrets without exporting env variables ([#345](https://github.com/grafana/plugin-validator/issues/345)) ([a1a241b](https://github.com/grafana/plugin-validator/commit/a1a241b95781bf2ccd8fb1709651a8d000821ae8))
