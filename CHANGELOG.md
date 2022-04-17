# Changelog

## [v0.40.1](https://github.com/k1LoW/octocov/compare/v0.40.0...v0.40.1) (2022-04-18)

* Fix artifact name [#155](https://github.com/k1LoW/octocov/pull/155) ([k1LoW](https://github.com/k1LoW))

## [v0.40.0](https://github.com/k1LoW/octocov/compare/v0.39.2...v0.40.0) (2022-04-17)

* Fix the name of storing report data in the Artifact in the case of changing the `repository:` section. [#154](https://github.com/k1LoW/octocov/pull/154) ([k1LoW](https://github.com/k1LoW))
* Separate comment for each repository name [#153](https://github.com/k1LoW/octocov/pull/153) ([k1LoW](https://github.com/k1LoW))

## [v0.39.2](https://github.com/k1LoW/octocov/compare/v0.39.1...v0.39.2) (2022-04-11)

* Fix handling of branch with slash in name [#151](https://github.com/k1LoW/octocov/pull/151) ([k1LoW](https://github.com/k1LoW))

## [v0.39.1](https://github.com/k1LoW/octocov/compare/v0.39.0...v0.39.1) (2022-04-03)

* Fix FuzzyFindByFile [#150](https://github.com/k1LoW/octocov/pull/150) ([k1LoW](https://github.com/k1LoW))

## [v0.39.0](https://github.com/k1LoW/octocov/compare/v0.38.2...v0.39.0) (2022-04-03)

* Support JaCoCo format [#149](https://github.com/k1LoW/octocov/pull/149) ([k1LoW](https://github.com/k1LoW))

## [v0.38.2](https://github.com/k1LoW/octocov/compare/v0.38.1...v0.38.2) (2022-03-16)

* Revert build environment [#147](https://github.com/k1LoW/octocov/pull/147) ([k1LoW](https://github.com/k1LoW))

## [v0.38.2](https://github.com/k1LoW/octocov/compare/v0.38.1...v0.38.2) (2022-03-16)

* Revert build environment [#147](https://github.com/k1LoW/octocov/pull/147) ([k1LoW](https://github.com/k1LoW))

## [v0.38.1](https://github.com/k1LoW/octocov/compare/v0.38.0...v0.38.1) (2022-03-13)

* When detecting prefix, only files under the working directory are targeted. [#145](https://github.com/k1LoW/octocov/pull/145) ([k1LoW](https://github.com/k1LoW))
* Fix: panic when targeting a file with no coverage data. [#144](https://github.com/k1LoW/octocov/pull/144) ([k1LoW](https://github.com/k1LoW))

## [v0.38.0](https://github.com/k1LoW/octocov/compare/v0.37.1...v0.38.0) (2022-02-19)

* [BREAKING] Remove all `enable:` section [#143](https://github.com/k1LoW/octocov/pull/143) ([k1LoW](https://github.com/k1LoW))
* When merging coverage reports, if any one of them is not a TypeLOC, it should be TypeMerged. [#142](https://github.com/k1LoW/octocov/pull/142) ([k1LoW](https://github.com/k1LoW))

## [v0.37.1](https://github.com/k1LoW/octocov/compare/v0.37.0...v0.37.1) (2022-02-11)

* [BREAKING] Revert "coverage.Gocover return LOC coverage (not statement count)" [#141](https://github.com/k1LoW/octocov/pull/141) ([k1LoW](https://github.com/k1LoW))

## [v0.37.0](https://github.com/k1LoW/octocov/compare/v0.36.0...v0.37.0) (2022-02-11)

* [BREAKING] coverage.Gocover return LOC coverage (not statement count) [#140](https://github.com/k1LoW/octocov/pull/140) ([k1LoW](https://github.com/k1LoW))

## [v0.36.0](https://github.com/k1LoW/octocov/compare/v0.35.0...v0.36.0) (2022-02-10)

* Add `octocov init` for generating .octocov.yml [#139](https://github.com/k1LoW/octocov/pull/139) ([k1LoW](https://github.com/k1LoW))
* Not shrinking report data in some datastores [#138](https://github.com/k1LoW/octocov/pull/138) ([k1LoW](https://github.com/k1LoW))

## [v0.35.0](https://github.com/k1LoW/octocov/compare/v0.34.0...v0.35.0) (2022-02-07)

* Support datastore.artifact [#137](https://github.com/k1LoW/octocov/pull/137) ([k1LoW](https://github.com/k1LoW))
* Add log [#136](https://github.com/k1LoW/octocov/pull/136) ([k1LoW](https://github.com/k1LoW))

## [v0.34.0](https://github.com/k1LoW/octocov/compare/v0.33.3...v0.34.0) (2022-02-03)

* [BREAKING] Minimize previous coverage report comments instead of deleting them [#135](https://github.com/k1LoW/octocov/pull/135) ([k1LoW](https://github.com/k1LoW))
* Add test for pkg/badge [#134](https://github.com/k1LoW/octocov/pull/134) ([k1LoW](https://github.com/k1LoW))

## [v0.33.3](https://github.com/k1LoW/octocov/compare/v0.33.2...v0.33.3) (2022-01-21)

* Fix the counting non codes [#133](https://github.com/k1LoW/octocov/pull/133) ([k1LoW](https://github.com/k1LoW))
* Fix the counting of metrics when merging. [#132](https://github.com/k1LoW/octocov/pull/132) ([k1LoW](https://github.com/k1LoW))

## [v0.33.2](https://github.com/k1LoW/octocov/compare/v0.33.1...v0.33.2) (2022-01-21)

* Fix SimpleCov parser [#131](https://github.com/k1LoW/octocov/pull/131) ([k1LoW](https://github.com/k1LoW))

## [v0.33.1](https://github.com/k1LoW/octocov/compare/v0.33.0...v0.33.1) (2022-01-16)

* Fix parallel test [#130](https://github.com/k1LoW/octocov/pull/130) ([k1LoW](https://github.com/k1LoW))

## [v0.33.0](https://github.com/k1LoW/octocov/compare/v0.32.0...v0.33.0) (2022-01-15)

* Replace io/ioutil [#129](https://github.com/k1LoW/octocov/pull/129) ([k1LoW](https://github.com/k1LoW))
* Additional commits to #127 [#128](https://github.com/k1LoW/octocov/pull/128) ([k1LoW](https://github.com/k1LoW))
* [BREAKING] Add `--report` to `octocov` command. If `--report` is specified, only that report file is loaded. [#127](https://github.com/k1LoW/octocov/pull/127) ([k1LoW](https://github.com/k1LoW))
* Update pkgs [#126](https://github.com/k1LoW/octocov/pull/126) ([k1LoW](https://github.com/k1LoW))
* Support another SimpleCov format [#125](https://github.com/k1LoW/octocov/pull/125) ([k1LoW](https://github.com/k1LoW))
* Add log for debug [#124](https://github.com/k1LoW/octocov/pull/124) ([k1LoW](https://github.com/k1LoW))

## [v0.32.0](https://github.com/k1LoW/octocov/compare/v0.31.0...v0.32.0) (2022-01-12)

* [BREAKING] If env CI is not set, `octocov` command only displays metrics [#123](https://github.com/k1LoW/octocov/pull/123) ([k1LoW](https://github.com/k1LoW))
* [BREAKING] Add `octocov migrate-bq-table` and remove option `--create-bq-table` [#122](https://github.com/k1LoW/octocov/pull/122) ([k1LoW](https://github.com/k1LoW))
* [BREAKING] Add `octocov badge` and remove `--*-badge` [#121](https://github.com/k1LoW/octocov/pull/121) ([k1LoW](https://github.com/k1LoW))

## [v0.31.0](https://github.com/k1LoW/octocov/compare/v0.30.0...v0.31.0) (2021-12-29)

* Measuring test execution time by identifying steps of GitHub Actions from timestamp of multiple coverage report files [#120](https://github.com/k1LoW/octocov/pull/120) ([k1LoW](https://github.com/k1LoW))
* [BREAKING] Support multiple coverage report files [#119](https://github.com/k1LoW/octocov/pull/119) ([k1LoW](https://github.com/k1LoW))
* Support for merging ratios [#118](https://github.com/k1LoW/octocov/pull/118) ([k1LoW](https://github.com/k1LoW))
* Support for merging coverages [#117](https://github.com/k1LoW/octocov/pull/117) ([k1LoW](https://github.com/k1LoW))
* Fix coverage count [#116](https://github.com/k1LoW/octocov/pull/116) ([k1LoW](https://github.com/k1LoW))
* Fix coverage NumStmt (Cobertura, LCOV, SimpleCov) [#115](https://github.com/k1LoW/octocov/pull/115) ([k1LoW](https://github.com/k1LoW))
* [BREAKING] Change ratio.Ratio struct [#114](https://github.com/k1LoW/octocov/pull/114) ([k1LoW](https://github.com/k1LoW))
* Fix coverage count [#113](https://github.com/k1LoW/octocov/pull/113) ([k1LoW](https://github.com/k1LoW))
* Update pkgs [#112](https://github.com/k1LoW/octocov/pull/112) ([k1LoW](https://github.com/k1LoW))

## [v0.30.0](https://github.com/k1LoW/octocov/compare/v0.29.0...v0.30.0) (2021-12-08)

* If the condition in the `*.acceptable:` section is not met, add an error message to the comment of the pull request. [#111](https://github.com/k1LoW/octocov/pull/111) ([k1LoW](https://github.com/k1LoW))
* Use os.DirFS instead of k1LoW/osfs [#110](https://github.com/k1LoW/octocov/pull/110) ([k1LoW](https://github.com/k1LoW))
* Update packages and Go [#109](https://github.com/k1LoW/octocov/pull/109) ([k1LoW](https://github.com/k1LoW))

## [v0.29.0](https://github.com/k1LoW/octocov/compare/v0.28.3...v0.29.0) (2021-11-18)

* Update `acceptable` section logic [#108](https://github.com/k1LoW/octocov/pull/108) ([k1LoW](https://github.com/k1LoW))
* Add value `is_default_branch` in the `if:` section [#107](https://github.com/k1LoW/octocov/pull/107) ([k1LoW](https://github.com/k1LoW))
* Fully implemented the github datastore. [#106](https://github.com/k1LoW/octocov/pull/106) ([k1LoW](https://github.com/k1LoW))
* Use k1LoW/go-github-client [#105](https://github.com/k1LoW/octocov/pull/105) ([k1LoW](https://github.com/k1LoW))

## [v0.28.3](https://github.com/k1LoW/octocov/compare/v0.28.2...v0.28.3) (2021-11-01)

* Fix handle of loc [#104](https://github.com/k1LoW/octocov/pull/104) ([k1LoW](https://github.com/k1LoW))

## [v0.28.2](https://github.com/k1LoW/octocov/compare/v0.28.1...v0.28.2) (2021-10-30)

* Fix the bug of getting reports for diff [#103](https://github.com/k1LoW/octocov/pull/103) ([k1LoW](https://github.com/k1LoW))

## [v0.28.1](https://github.com/k1LoW/octocov/compare/v0.28.0...v0.28.1) (2021-10-30)

* Fix root dir of Code to Test Ratio [#102](https://github.com/k1LoW/octocov/pull/102) ([k1LoW](https://github.com/k1LoW))

## [v0.28.0](https://github.com/k1LoW/octocov/compare/v0.27.1...v0.28.0) (2021-10-30)

* [BREAKING] Support code metrics for each application in the monorepo [#101](https://github.com/k1LoW/octocov/pull/101) ([k1LoW](https://github.com/k1LoW))
* Fix report comment [#100](https://github.com/k1LoW/octocov/pull/100) ([k1LoW](https://github.com/k1LoW))

## [v0.27.1](https://github.com/k1LoW/octocov/compare/v0.27.0...v0.27.1) (2021-10-28)

* Fix handle coverage.yml of Clover format [#99](https://github.com/k1LoW/octocov/pull/99) ([k1LoW](https://github.com/k1LoW))

## [v0.27.0](https://github.com/k1LoW/octocov/compare/v0.26.1...v0.27.0) (2021-10-28)

* [BREAKING] Add `central.badges.datastores:` section instead of `central.badges:` [#98](https://github.com/k1LoW/octocov/pull/98) ([k1LoW](https://github.com/k1LoW))
* Add octocov logo [#97](https://github.com/k1LoW/octocov/pull/97) ([k1LoW](https://github.com/k1LoW))

## [v0.26.1](https://github.com/k1LoW/octocov/compare/v0.26.0...v0.26.1) (2021-10-26)

* Check if a job is related to a opened pull request in CommentConfigReady() [#96](https://github.com/k1LoW/octocov/pull/96) ([k1LoW](https://github.com/k1LoW))

## [v0.26.0](https://github.com/k1LoW/octocov/compare/v0.25.0...v0.26.0) (2021-10-26)

* Add value `is_pull_request` in the `if:` section  [#95](https://github.com/k1LoW/octocov/pull/95) ([k1LoW](https://github.com/k1LoW))
* Add central.if: section [#94](https://github.com/k1LoW/octocov/pull/94) ([k1LoW](https://github.com/k1LoW))

## [v0.25.0](https://github.com/k1LoW/octocov/compare/v0.24.0...v0.25.0) (2021-10-22)

* Add comment.if: section [#93](https://github.com/k1LoW/octocov/pull/93) ([k1LoW](https://github.com/k1LoW))
* [BREAKING] `enable: true` can be omitted if any other parameters are set. [#92](https://github.com/k1LoW/octocov/pull/92) ([k1LoW](https://github.com/k1LoW))
* fix(error): improve execution time violation message [#91](https://github.com/k1LoW/octocov/pull/91) ([rizalgowandy](https://github.com/rizalgowandy))
* Add diff.if: section [#90](https://github.com/k1LoW/octocov/pull/90) ([k1LoW](https://github.com/k1LoW))

## [v0.24.0](https://github.com/k1LoW/octocov/compare/v0.23.3...v0.24.0) (2021-10-18)

* Detect file encoding [#89](https://github.com/k1LoW/octocov/pull/89) ([k1LoW](https://github.com/k1LoW))

## [v0.23.3](https://github.com/k1LoW/octocov/compare/v0.23.2...v0.23.3) (2021-10-17)

* Fix Cloud Storage (gcs) client creation using default application credentials [#88](https://github.com/k1LoW/octocov/pull/88) ([dragon3](https://github.com/dragon3))

## [v0.23.2](https://github.com/k1LoW/octocov/compare/v0.23.1...v0.23.2) (2021-10-15)

* Fix simplecov block line [#87](https://github.com/k1LoW/octocov/pull/87) ([k1LoW](https://github.com/k1LoW))

## [v0.23.1](https://github.com/k1LoW/octocov/compare/v0.23.0...v0.23.1) (2021-10-14)

* If the prefix is ".", convert it to "". [#86](https://github.com/k1LoW/octocov/pull/86) ([k1LoW](https://github.com/k1LoW))

## [v0.23.0](https://github.com/k1LoW/octocov/compare/v0.22.2...v0.23.0) (2021-10-14)

* [BREAKING] Fix handle coverage file path [#85](https://github.com/k1LoW/octocov/pull/85) ([k1LoW](https://github.com/k1LoW))
* Fix handle filepath of cobertura  [#84](https://github.com/k1LoW/octocov/pull/84) ([k1LoW](https://github.com/k1LoW))

## [v0.22.2](https://github.com/k1LoW/octocov/compare/v0.22.1...v0.22.2) (2021-10-13)

* Fix ls-files path detection [#83](https://github.com/k1LoW/octocov/pull/83) ([k1LoW](https://github.com/k1LoW))

## [v0.22.1](https://github.com/k1LoW/octocov/compare/v0.22.0...v0.22.1) (2021-10-12)

* Fix file path relativization process of coverage [#82](https://github.com/k1LoW/octocov/pull/82) ([k1LoW](https://github.com/k1LoW))

## [v0.22.0](https://github.com/k1LoW/octocov/compare/v0.21.1...v0.22.0) (2021-10-12)

* Fix markdown table when long branch name [#81](https://github.com/k1LoW/octocov/pull/81) ([k1LoW](https://github.com/k1LoW))
* Detect root path using env `GITHUB_WORKSPACE` [#80](https://github.com/k1LoW/octocov/pull/80) ([k1LoW](https://github.com/k1LoW))

## [v0.21.1](https://github.com/k1LoW/octocov/compare/v0.21.0...v0.21.1) (2021-10-12)

* Fix nil pointer dereference when no code coverage metrics [#79](https://github.com/k1LoW/octocov/pull/79) ([k1LoW](https://github.com/k1LoW))

## [v0.21.0](https://github.com/k1LoW/octocov/compare/v0.20.1...v0.21.0) (2021-10-11)

* Add Getting Started [#78](https://github.com/k1LoW/octocov/pull/78) ([k1LoW](https://github.com/k1LoW))
* [BREAKING] Fix config [#77](https://github.com/k1LoW/octocov/pull/77) ([k1LoW](https://github.com/k1LoW))
* Fix ls-files file detection [#76](https://github.com/k1LoW/octocov/pull/76) ([k1LoW](https://github.com/k1LoW))
* [BREAKING] Change each file path in the coverage report to be relative to git root [#75](https://github.com/k1LoW/octocov/pull/75) ([k1LoW](https://github.com/k1LoW))

## [v0.20.1](https://github.com/k1LoW/octocov/compare/v0.20.0...v0.20.1) (2021-09-27)

* Fix: panic: assignment to entry in nil map [#74](https://github.com/k1LoW/octocov/pull/74) ([k1LoW](https://github.com/k1LoW))
* If `report.path:`, save the full report data [#73](https://github.com/k1LoW/octocov/pull/73) ([k1LoW](https://github.com/k1LoW))
* Add `report.path:` to save the report local path. [#72](https://github.com/k1LoW/octocov/pull/72) ([k1LoW](https://github.com/k1LoW))

## [v0.20.0](https://github.com/k1LoW/octocov/compare/v0.19.0...v0.20.0) (2021-09-22)

* Comment report using diff [#71](https://github.com/k1LoW/octocov/pull/71) ([k1LoW](https://github.com/k1LoW))
* Output code metrics report to STDOUT when octocov command is executed. [#70](https://github.com/k1LoW/octocov/pull/70) ([k1LoW](https://github.com/k1LoW))
* Add `octocov diff` [#69](https://github.com/k1LoW/octocov/pull/69) ([k1LoW](https://github.com/k1LoW))

## [v0.19.0](https://github.com/k1LoW/octocov/compare/v0.18.1...v0.19.0) (2021-09-17)

* Fix title [#68](https://github.com/k1LoW/octocov/pull/68) ([k1LoW](https://github.com/k1LoW))
* Add code coverage report of changed files to comment on pull request. [#67](https://github.com/k1LoW/octocov/pull/67) ([k1LoW](https://github.com/k1LoW))

## [v0.18.1](https://github.com/k1LoW/octocov/compare/v0.18.0...v0.18.1) (2021-09-15)

* Fix `NaN%` coverage [#65](https://github.com/k1LoW/octocov/pull/65) ([k1LoW](https://github.com/k1LoW))

## [v0.18.0](https://github.com/k1LoW/octocov/compare/v0.17.2...v0.18.0) (2021-09-15)

* Flush the block coverages from the report to handle `Error 413 (Request Entity Too Large)!!1` error. [#64](https://github.com/k1LoW/octocov/pull/64) ([k1LoW](https://github.com/k1LoW))
* Add `octocov ls-files` [#63](https://github.com/k1LoW/octocov/pull/63) ([k1LoW](https://github.com/k1LoW))
* [BREAKING] Add `octocov dump` instead of `--dump` [#62](https://github.com/k1LoW/octocov/pull/62) ([k1LoW](https://github.com/k1LoW))

## [v0.17.2](https://github.com/k1LoW/octocov/compare/v0.17.1...v0.17.2) (2021-09-13)

* Fix paintLine when TypeStmt [#61](https://github.com/k1LoW/octocov/pull/61) ([k1LoW](https://github.com/k1LoW))

## [v0.17.1](https://github.com/k1LoW/octocov/compare/v0.17.0...v0.17.1) (2021-09-13)

* Add `--report` option [#60](https://github.com/k1LoW/octocov/pull/60) ([k1LoW](https://github.com/k1LoW))
* Fix completion installation [#59](https://github.com/k1LoW/octocov/pull/59) ([k1LoW](https://github.com/k1LoW))

## [v0.17.0](https://github.com/k1LoW/octocov/compare/v0.16.0...v0.17.0) (2021-09-13)

* Added `octocov cat` to check line-by-line coverage of source code. [#58](https://github.com/k1LoW/octocov/pull/58) ([k1LoW](https://github.com/k1LoW))
* Change TypeStatement to TypeStmt [#57](https://github.com/k1LoW/octocov/pull/57) ([k1LoW](https://github.com/k1LoW))
* Include coverage per block in the report [#56](https://github.com/k1LoW/octocov/pull/56) ([k1LoW](https://github.com/k1LoW))
* Use cobra default completion [#55](https://github.com/k1LoW/octocov/pull/55) ([k1LoW](https://github.com/k1LoW))
* Fix testdata dir [#54](https://github.com/k1LoW/octocov/pull/54) ([k1LoW](https://github.com/k1LoW))
* Use github.com/k1LoW/osfs [#53](https://github.com/k1LoW/octocov/pull/53) ([k1LoW](https://github.com/k1LoW))

## [v0.16.0](https://github.com/k1LoW/octocov/compare/v0.15.2...v0.16.0) (2021-08-24)

* Support `OCTOCOV_` prefix environment variables [#52](https://github.com/k1LoW/octocov/pull/52) ([k1LoW](https://github.com/k1LoW))

## [v0.15.2](https://github.com/k1LoW/octocov/compare/v0.15.1...v0.15.2) (2021-08-24)

* Fix --create-bq-table [#51](https://github.com/k1LoW/octocov/pull/51) ([k1LoW](https://github.com/k1LoW))

## [v0.15.1](https://github.com/k1LoW/octocov/compare/v0.15.0...v0.15.1) (2021-08-24)

* Fix --create-bq-table [#50](https://github.com/k1LoW/octocov/pull/50) ([k1LoW](https://github.com/k1LoW))

## [v0.15.0](https://github.com/k1LoW/octocov/compare/v0.14.0...v0.15.0) (2021-08-18)

* Support GOOGLE_APPLICATION_CREDENTIALS_JSON [#49](https://github.com/k1LoW/octocov/pull/49) ([k1LoW](https://github.com/k1LoW))
* Add trivy-action [#48](https://github.com/k1LoW/octocov/pull/48) ([k1LoW](https://github.com/k1LoW))

## [v0.14.0](https://github.com/k1LoW/octocov/compare/v0.13.0...v0.14.0) (2021-08-09)

* Fix BigQuery query error [#47](https://github.com/k1LoW/octocov/pull/47) ([k1LoW](https://github.com/k1LoW))
* [BREAKING] Support multi datastores on central mode and change config format [#46](https://github.com/k1LoW/octocov/pull/46) ([k1LoW](https://github.com/k1LoW))
* [BREAKING] Support multi datastores and change config format. [#45](https://github.com/k1LoW/octocov/pull/45) ([k1LoW](https://github.com/k1LoW))

## [v0.13.0](https://github.com/k1LoW/octocov/compare/v0.12.1...v0.13.0) (2021-08-05)

* Support BigQuery for central.reports [#44](https://github.com/k1LoW/octocov/pull/44) ([k1LoW](https://github.com/k1LoW))
* Support datastore.bq [#43](https://github.com/k1LoW/octocov/pull/43) ([k1LoW](https://github.com/k1LoW))
* Support GCS for central.reports [#42](https://github.com/k1LoW/octocov/pull/42) ([k1LoW](https://github.com/k1LoW))
* Change datasource.Datastore interface [#41](https://github.com/k1LoW/octocov/pull/41) ([k1LoW](https://github.com/k1LoW))
* Support datastore.gcs [#40](https://github.com/k1LoW/octocov/pull/40) ([k1LoW](https://github.com/k1LoW))
* Support S3 for central.reports [#39](https://github.com/k1LoW/octocov/pull/39) ([k1LoW](https://github.com/k1LoW))
* Refactor central [#38](https://github.com/k1LoW/octocov/pull/38) ([k1LoW](https://github.com/k1LoW))
* Change datastore interface [#37](https://github.com/k1LoW/octocov/pull/37) ([k1LoW](https://github.com/k1LoW))
* Support datastore.s3 [#36](https://github.com/k1LoW/octocov/pull/36) ([k1LoW](https://github.com/k1LoW))
* Remove ghdag pkg [#35](https://github.com/k1LoW/octocov/pull/35) ([k1LoW](https://github.com/k1LoW))

## [v0.12.1](https://github.com/k1LoW/octocov/compare/v0.12.0...v0.12.1) (2021-07-02)

* Avoid incorrectly parsing other report formats. [#34](https://github.com/k1LoW/octocov/pull/34) ([k1LoW](https://github.com/k1LoW))

## [v0.12.0](https://github.com/k1LoW/octocov/compare/v0.11.0...v0.12.0) (2021-07-02)

* Support Cobertura XML format [#33](https://github.com/k1LoW/octocov/pull/33) ([k1LoW](https://github.com/k1LoW))
* Allow no code coverage report [#32](https://github.com/k1LoW/octocov/pull/32) ([k1LoW](https://github.com/k1LoW))
* Add comment.hideFooterLink section [#31](https://github.com/k1LoW/octocov/pull/31) ([k1LoW](https://github.com/k1LoW))

## [v0.11.0](https://github.com/k1LoW/octocov/compare/v0.10.0...v0.11.0) (2021-06-13)

* Update central mode report format [#30](https://github.com/k1LoW/octocov/pull/30) ([k1LoW](https://github.com/k1LoW))
* Add comment: for commenting report to pull request [#29](https://github.com/k1LoW/octocov/pull/29) ([k1LoW](https://github.com/k1LoW))

## [v0.10.0](https://github.com/k1LoW/octocov/compare/v0.9.0...v0.10.0) (2021-06-02)

* Support measure parallel/multi tests execution time [#28](https://github.com/k1LoW/octocov/pull/28) ([k1LoW](https://github.com/k1LoW))

## [v0.9.0](https://github.com/k1LoW/octocov/compare/v0.8.0...v0.9.0) (2021-05-30)

* Add push.if: section and central.push.if: section [#27](https://github.com/k1LoW/octocov/pull/27) ([k1LoW](https://github.com/k1LoW))
* Add push: for support self push badges [#26](https://github.com/k1LoW/octocov/pull/26) ([k1LoW](https://github.com/k1LoW))

## [v0.8.0](https://github.com/k1LoW/octocov/compare/v0.7.3...v0.8.0) (2021-05-26)

* Add testExecutionTime.acceptable: ( and fix typo ) [#25](https://github.com/k1LoW/octocov/pull/25) ([k1LoW](https://github.com/k1LoW))

## [v0.7.3](https://github.com/k1LoW/octocov/compare/v0.7.2...v0.7.3) (2021-05-24)

* Fix logic of detect step [#24](https://github.com/k1LoW/octocov/pull/24) ([k1LoW](https://github.com/k1LoW))

## [v0.7.2](https://github.com/k1LoW/octocov/compare/v0.7.1...v0.7.2) (2021-05-24)

* Fix log output [#23](https://github.com/k1LoW/octocov/pull/23) ([k1LoW](https://github.com/k1LoW))

## [v0.7.1](https://github.com/k1LoW/octocov/compare/v0.7.0...v0.7.1) (2021-05-24)

* Skip measuring test execution time when fail to detect test time [#22](https://github.com/k1LoW/octocov/pull/22) ([k1LoW](https://github.com/k1LoW))
* Add backoff logic to GetStepExecutionTimeByTime [#21](https://github.com/k1LoW/octocov/pull/21) ([k1LoW](https://github.com/k1LoW))

## [v0.7.0](https://github.com/k1LoW/octocov/compare/v0.6.1...v0.7.0) (2021-05-23)

* Support test execution time [#20](https://github.com/k1LoW/octocov/pull/20) ([k1LoW](https://github.com/k1LoW))
* Add gh.Gh and inject gh.Gh to datastore.Github [#19](https://github.com/k1LoW/octocov/pull/19) ([k1LoW](https://github.com/k1LoW))
* Support self git push in central mode [#18](https://github.com/k1LoW/octocov/pull/18) ([k1LoW](https://github.com/k1LoW))
* Fix option name [#17](https://github.com/k1LoW/octocov/pull/17) ([k1LoW](https://github.com/k1LoW))

## [v0.6.1](https://github.com/k1LoW/octocov/compare/v0.6.0...v0.6.1) (2021-05-12)

* Fix template of central mode [#15](https://github.com/k1LoW/octocov/pull/15) ([k1LoW](https://github.com/k1LoW))
* Fix build option [#16](https://github.com/k1LoW/octocov/pull/16) ([k1LoW](https://github.com/k1LoW))

## [v0.6.0](https://github.com/k1LoW/octocov/compare/v0.5.0...v0.6.0) (2021-05-12)

* Fix pkg/badge field names [#14](https://github.com/k1LoW/octocov/pull/14) ([k1LoW](https://github.com/k1LoW))
* [BREAKING]Support code to test ratio [#13](https://github.com/k1LoW/octocov/pull/13) ([k1LoW](https://github.com/k1LoW))

## [v0.5.0](https://github.com/k1LoW/octocov/compare/v0.4.0...v0.5.0) (2021-05-11)

* Show badge markdown link [#12](https://github.com/k1LoW/octocov/pull/12) ([k1LoW](https://github.com/k1LoW))

## [v0.4.0](https://github.com/k1LoW/octocov/compare/v0.3.1...v0.4.0) (2021-05-10)

* Support `datastore.if:` section [#11](https://github.com/k1LoW/octocov/pull/11) ([k1LoW](https://github.com/k1LoW))

## [v0.3.1](https://github.com/k1LoW/octocov/compare/v0.3.0...v0.3.1) (2021-05-10)

* Fix badge path rel [#10](https://github.com/k1LoW/octocov/pull/10) ([k1LoW](https://github.com/k1LoW))

## [v0.3.0](https://github.com/k1LoW/octocov/compare/v0.2.0...v0.3.0) (2021-05-08)

* [BREAKING] Update coverage.badge [#9](https://github.com/k1LoW/octocov/pull/9) ([k1LoW](https://github.com/k1LoW))

## [v0.2.0](https://github.com/k1LoW/octocov/compare/v0.1.1...v0.2.0) (2021-05-07)

* Add central mode [#8](https://github.com/k1LoW/octocov/pull/8) ([k1LoW](https://github.com/k1LoW))
* Enable Clover parser [#7](https://github.com/k1LoW/octocov/pull/7) ([k1LoW](https://github.com/k1LoW))
* Fix lcov does not set file name [#6](https://github.com/k1LoW/octocov/pull/6) ([k1LoW](https://github.com/k1LoW))
* Change default report path [#5](https://github.com/k1LoW/octocov/pull/5) ([k1LoW](https://github.com/k1LoW))
* Fix default datastore.github.path: [#4](https://github.com/k1LoW/octocov/pull/4) ([k1LoW](https://github.com/k1LoW))
* Use k1LoW/octocov-action [#3](https://github.com/k1LoW/octocov/pull/3) ([k1LoW](https://github.com/k1LoW))

## [v0.1.1](https://github.com/k1LoW/octocov/compare/v0.1.0...v0.1.1) (2021-05-05)

* Resolve permission error when creating a directory. [#2](https://github.com/k1LoW/octocov/pull/2) ([k1LoW](https://github.com/k1LoW))
* Fix `octocov completion` interface [#1](https://github.com/k1LoW/octocov/pull/1) ([k1LoW](https://github.com/k1LoW))

## [v0.1.0](https://github.com/k1LoW/octocov/compare/88314da64080...v0.1.0) (2021-05-04)

