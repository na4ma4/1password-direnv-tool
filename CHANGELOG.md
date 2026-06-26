# Changelog

## [1.3.0](https://github.com/na4ma4/1password-direnv-tool/compare/v1.2.1...v1.3.0) (2026-06-26)


### Features

* Add support for direnv watch files in op-direnv ([72d1f23](https://github.com/na4ma4/1password-direnv-tool/commit/72d1f23f25e4e4eed4c21019282d694dafa03f16))


### Bug Fixes

* remove commented-out legacy cache methods and return copy in GetModifiers ([dec83d4](https://github.com/na4ma4/1password-direnv-tool/commit/dec83d47316ebc355b19d3d9fdf5f64ffe6cc1ea))

## [1.2.1](https://github.com/na4ma4/1password-direnv-tool/compare/v1.2.0...v1.2.1) (2026-06-26)


### Bug Fixes

* Handle non-ID vault references in 1Password provider ([cbaacf4](https://github.com/na4ma4/1password-direnv-tool/commit/cbaacf4e9b65665a19b3b2b5c885eef7f91fe276))

## [1.2.0](https://github.com/na4ma4/1password-direnv-tool/compare/v1.1.1...v1.2.0) (2026-06-23)


### Features

* add support for alternative password providers and k8s exec-plugin ([f4414eb](https://github.com/na4ma4/1password-direnv-tool/commit/f4414ebcf3f4714a734e59d130973f40b3271995))

## [1.1.1](https://github.com/na4ma4/1password-direnv-tool/compare/v1.1.0...v1.1.1) (2026-03-27)


### Bug Fixes

* gosec linter warning ([78c2660](https://github.com/na4ma4/1password-direnv-tool/commit/78c26609881993bb28e7c49548dbe0858d26a322))


### Miscellaneous Chores

* release 1.1.1 ([8784fea](https://github.com/na4ma4/1password-direnv-tool/commit/8784feabe788064ce20d95cb2aa56c175d15f606))

## [1.1.0](https://github.com/na4ma4/1password-direnv-tool/compare/v1.0.3...v1.1.0) (2026-03-19)


### Features

* wrapped 1password to ensure timeout is enforced ([bdeba22](https://github.com/na4ma4/1password-direnv-tool/commit/bdeba225d59708fd4bfb610b29125c85b18d5084))


### Bug Fixes

* timeout on onepassword client initialization ([#8](https://github.com/na4ma4/1password-direnv-tool/issues/8)) ([88845c3](https://github.com/na4ma4/1password-direnv-tool/commit/88845c323b3baad65139355d8802efd1464d84a0))

## [1.0.3](https://github.com/na4ma4/1password-direnv-tool/compare/v1.0.2...v1.0.3) (2026-02-26)


### Bug Fixes

* enabled encryption on cache ([73095b1](https://github.com/na4ma4/1password-direnv-tool/commit/73095b14591f340c23c00405d14391e1199b1540))
* homebrew deployment script ([ecc1c75](https://github.com/na4ma4/1password-direnv-tool/commit/ecc1c758ca1f85b9c6aa01a212e44f68306845f8))

## [1.0.2](https://github.com/na4ma4/1password-direnv-tool/compare/v1.0.1...v1.0.2) (2026-02-26)


### Bug Fixes

* release builds on macos (default golang is 1.25) ([89a7e46](https://github.com/na4ma4/1password-direnv-tool/commit/89a7e46f52fa8495a39cc5320f0904aba520e632))

## [1.0.1](https://github.com/na4ma4/1password-direnv-tool/compare/v1.0.0...v1.0.1) (2026-02-26)


### Bug Fixes

* release builds on macos ([964909e](https://github.com/na4ma4/1password-direnv-tool/commit/964909e50a404b610807347a4f74343f4cca5827))

## 1.0.0 (2026-02-26)


### Features

* Add Go CLI tool outline replacing shell script with onepassword-sdk-go ([#1](https://github.com/na4ma4/1password-direnv-tool/issues/1)) ([8714019](https://github.com/na4ma4/1password-direnv-tool/commit/8714019c9d0374eeac185f09bceeb9f1c336e611))


### Bug Fixes

* dependabot auto-merge and go workspace files ([312c263](https://github.com/na4ma4/1password-direnv-tool/commit/312c263770601080bdd3a3b7b898b53dcb52b710))
* linter warning in non-darwin code ([6ccd707](https://github.com/na4ma4/1password-direnv-tool/commit/6ccd707f92fc671592e69d1b4ed76ce02858be35))
* update github workflow ([83c8425](https://github.com/na4ma4/1password-direnv-tool/commit/83c8425d3b500f2caac5efd4cdc9d3ee52c23301))
* update missing go-module ([13d2691](https://github.com/na4ma4/1password-direnv-tool/commit/13d26912902d16dbbf71454c183b7cd3e79cf88a))
