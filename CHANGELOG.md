# Changelog
All **breaking** changes to this project will be documented in this file.

The format is based loosely on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---
### [Unreleased]
* Changed Txn and Query{} API, reduced api noise, removed funcs that should not be used now after upserts and types got introduced
* Added SetRDF and DeleteRDF mutation helpers, along with Query{} helpers
* Renamed `FlattenJSON` -> `FlattenRespToObject` to better describe what it does
* Renamed `QueryJSON` -> `QueryDQL`
---

## v3.1.0 - 2020-11-23

- Added Query{}.GetPredExpandType helper
- Added Unsafe{}.FlattenRespToArray helper
- Changed Query{}.DeleteEdge() to support wildcard delete

## v3.0.0 - 2020-04-04

- Update for dgo v200.03.0 / dgraph 20.03.0
- Renamed `NewTxn()` to `NewTxnWithoutContext()`
- Renamed `NewTxnWithContext()` to `NewTxn()`
- `ndgo.Flatten()` has been removed. Replaced with `ndgo.Unsafe{}.FlattenJSON()`

## v2.2.0 - 2020-04-04

- Final version for dgo 2.2.0 / dgraph 1.2.2

## v2.0.1 - 2019-10-31

- Update for dgo 2.1.0 / dgraph 1.1.0

## v1.0.0 - 2019-09-13

- Initial release for dgo 1.0.0 / dgraph 1.0.0
