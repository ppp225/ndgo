# ndgo [![Build Status](https://travis-ci.org/ppp225/ndgo.svg?branch=master)](https://travis-ci.org/ppp225/ndgo)    [![codecov](https://codecov.io/gh/ppp225/ndgo/branch/master/graph/badge.svg)](https://codecov.io/gh/ppp225/ndgo)    [![Go Report Card](https://goreportcard.com/badge/github.com/ppp225/ndgo)](https://goreportcard.com/report/github.com/ppp225/ndgo)    [![Maintainability](https://api.codeclimate.com/v1/badges/7954fe4d199f0426bb5d/maintainability)](https://codeclimate.com/github/ppp225/ndgo/maintainability)   [![GoDoc](https://godoc.org/github.com/ppp225/ndgo?status.svg)](https://godoc.org/github.com/ppp225/ndgo)   
ndgo provides [dgraph](https://github.com/dgraph-io) [dgo](https://github.com/dgraph-io/dgo) txn abstractions and helpers.

# Why

* Reduce txn related boilerplate, thus making code more readable,
* Make using ratel like queries easier,
* Get query execution times the easy way, even for multi-query txns,
* Be low level - don't do magic, keep dgo things exposed for them to be usable, if necessary,
* No performance overhead compared to dgo, if possible.

# Install

```bash
go get -u github.com/ppp225/ndgo
```

```bash
import (
  "github.com/ppp225/ndgo/v5"
)
```

# Common uses

Run transactions the same way one would do in ratel:

```go
q := QueryDQL(fmt.Sprintf(`
  {
    %s(func: eq(%s, "%s")) {
      uid
      name
    }
  }
  `, "favActor", "name", "Keanu Reeves"))

resp, err := q.Run(txn)
```

Or marshal structs:

```go
jsonBytes, err := json.Marshal(myObj)
if err != nil {
  return nil, err
}
resp, err := txn.Setb(jsonBytes, nil)
```

Or don't:

```go
resp, err := txn.Seti(myObj, myObj2, myObj3)
```

Do Upserts:

```go
q := `{ a as var(func: eq(name, "Keanu")) }`
myObj := personStruct{
	UID:  "uid(a)",
	Type: "Person",
	Name: "Keanu",
}
resp, err := txn.DoSeti(q, myObj)
```

See `TestBasic` and `TestComplex` and `TestTxnUpsert` in `ndgo_test.go` for a complete example.

# ndgo.Txn

### Create a transaction:

```go
dg := NewDgraphClient()
txn := ndgo.NewTxn(ctx, dg.NewTxn()) // or dg.NewReadOnlyTxn(), you can use any available dgo.txn options. You can also use ndgo.NewTxnWithoutContext(txn)
defer txn.Discard()
...
err = txn.Commit()
```

### Do mutations and queries:

```go
resp, err := txn.Mutate(*api.Mutation)
resp, err := txn.Setb(jsonBytes, rdfBytes)
resp, err := txn.Seti(myObjs...)
resp, err := txn.Setnq(nquads)
resp, err := txn.Deleteb(jsonBytes, deleteRdfBytes)
resp, err := txn.Deletei(myObjs...)
resp, err := txn.Deletenq(nquads)

resp, err := txn.Query(queryString)
resp, err := txn.QueryWithVars(queryWithVarsString, vars...)

resp, err := txn.Do(req *api.Request)
resp, err := txn.DoSetb(queryString, jsonBytes)
resp, err := txn.DoSeti(queryString, myObjs...)
resp, err := txn.DoSetnq(queryString, nquads)
```

### Get diagnostics:

```go
dbms := txn.GetDatabaseTime()
nwms := txn.GetNetworkTime()
```

# ndgo.Set/Delete JSON/RDF

Define and run txns through json, rdf or predefined helpers

### Set:

```go
set := ndgo.SetJSON(fmt.Sprintf(`
  {
    "name": "%s",
    "age": "%s"
  }`, "L", "25"))
// or
set := ndgo.SetRDF(`<_:new> <name> "L" .
                    <_:new> <age> "25" .`)
// or
set := ndgo.Query{}.SetPred("_:new", name, "L") + 
       ndgo.Query{}.SetPred("_:new", age, "25")

resp, err := set.Run(txn)
```

### Delete:

```go
del := ndgo.DeleteJSON(fmt.Sprintf(`
  {
    "uid": "%s"
  }
  `, uid))
// or
del := ndgo.Query{}.DeleteNode("0x420") + 
       ndgo.Query{}.DeleteEdge("0x420", "edgeName", "0x42") +
       ndgo.Query{}.DeletePred("0x42", "name")

resp, err := del.Run(txn)
```

### Query:

```go
q := ndgo.QueryDQL(fmt.Sprintf(`
  {
    %s(func: eq(%s, "%s")) {
      uid
    }
  }
  `, "favActor", "name", "Keanu Reeves"))

response, err := q.Run(txn)
```

### Join:

You can chain queries and JSON mutations with Join, RDF mutations with + operator, assuming they are the same type:

```go
q1 := ndgo.QueryDQL(`{q1(func:eq(name,"a")){uid}})`
q2 := ndgo.QueryDQL(`{q2(func:eq(name,"b")){uid}})`
resp, err := q1.Join(q2).Run(txn)
```

Note that query blocks have to be named uniquely.

# Other helpers

### FlattenResp

Sometimes, when querying dgraph, we want just 1 resulting element, instead of a nested array.

```go
// Instead of having this:
type Queries struct {
  Q []struct {
    UID string `json:"uid"`
  } `json:"q"`
}
// We can have:
type MyObj struct {
  UID string `json:"uid"`
}
```
```go
resp, _ := ndgo.QueryDQL(`{q(func:uid(0x123)){uid}}`).Run(txn)
// query block name -------^ must be one letter and only one query block must be in the query for this helper to work!
var result MyObj
if err := json.Unmarshal(ndgo.Unsafe{}.FlattenRespToObject(resp.GetJson()), &result); err != nil {
	panic(err) // will fail, if result has more than 1 element!
}
var resultSlice []MyObj
if err := json.Unmarshal(ndgo.Unsafe{}.FlattenRespToArray(resp.GetJson()), &resultSlice); err != nil {
	panic(err) // will fail, if multiple query blocks are used in Query, but can have multiple results!
}
log.Print(result)
log.Print(resultSlice)
```

# Future plans

* add more upsert things

# Note

This project uses semantic versioning, see the [changelog](https://github.com/ppp225/ndgo/blob/master/CHANGELOG.md) for details.
