# ndgo [![Build Status](https://travis-ci.org/ppp225/ndgo.svg?branch=master)](https://travis-ci.org/ppp225/ndgo)    [![codecov](https://codecov.io/gh/ppp225/ndgo/branch/master/graph/badge.svg)](https://codecov.io/gh/ppp225/ndgo)    [![Go Report Card](https://goreportcard.com/badge/github.com/ppp225/ndgo)](https://goreportcard.com/report/github.com/ppp225/ndgo)    [![Maintainability](https://api.codeclimate.com/v1/badges/7954fe4d199f0426bb5d/maintainability)](https://codeclimate.com/github/ppp225/ndgo/maintainability)   [![GoDoc](https://godoc.org/github.com/ppp225/ndgo?status.svg)](https://godoc.org/github.com/ppp225/ndgo)   
ndgo provides [dgraph](https://github.com/dgraph-io) [dgo](https://github.com/dgraph-io/dgo) txn abstractions and helpers.

> ⚠️ **Info**: master branch is updated for dgo v200.03.0 / dgraph 20.03.0

# Why

* Reduce txn related boilerplate, thus making code more readable,
* Make using ratel like queries easier,
* Get query execution times the easy way,
* Don't do magic, keep dgo things exposed enough for them to be usable, if necessary.

# Install

```bash
go get -u github.com/ppp225/ndgo
```

```bash
import (
  "github.com/ppp225/ndgo/v3"
)
```

# Common uses

Run transactions the same way one would do in ratel:

```go
q := QueryJSON(fmt.Sprintf(`
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
resp, err := txn.Setb(jsonBytes)
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
txn := ndgo.NewTxnWithoutContext(dg.NewTxn()) // or dg.NewReadOnlyTxn(), you can use any dgo.txn options you like. You can also use ndgo.NewTxn(ctx, txn)
defer txn.Discard()
...
err = txn.Commit()
```

### Do mutations and queries:

```go
resp, err := txn.Mutate(&api.Mutation{SetJson: jsonBytes})
resp, err := txn.Set(setString)
resp, err := txn.Setb(setBytes)
resp, err := txn.Seti(myObjs...)
resp, err := txn.Delete(deleteString)
resp, err := txn.Deleteb(deleteBytes)
resp, err := txn.Deletei(myObjs...)

resp, err := txn.Query(queryString)
resp, err := txn.QueryWithVars(queryWithVarsString, vars...)

resp, err := txn.Do(req *api.Request)
resp, err := txn.DoSet(queryString, jsonString)
resp, err := txn.DoSetb(queryString, jsonBytes)
resp, err := txn.DoSeti(queryString, myObjs...)
resp, err := txn.DoSetnq(queryString, nquads)
```

### Get diagnostics:

```go
dbms := txn.GetDatabaseTime()
nwms := txn.GetNetworkTime()
```

# ndgo.*JSON

Define and run txn's the same way as in ratel.

### Set:

```go
set := ndgo.SetJSON(fmt.Sprintf(`
  {
    "name": "%s",
    "age": "%s"
  }`, "L", "25"))

assigned, err := set.Run(txn)
```

### Delete:

```go
del := ndgo.DeleteJSON(fmt.Sprintf(`
  {
    "uid": "%s"
  }
  `, uid))

assigned, err := del.Run(txn)
```

### Query:

```go
q := ndgo.QueryJSON(fmt.Sprintf(`
  {
    %s(func: eq(%s, "%s")) {
      uid
    }
  }
  `, "favActor", "name", "Keanu Reeves"))

response, err := q.Run(txn)
```

### Join:

You can chain queries with Join, assuming they are the same type:

```go
response, err := ndgo.Query{}.
  GetPredUID("q1", "name", "Keanu").Join(ndgo.Query{}.
  GetPredUID("q2", "name", "L")).Join(ndgo.Query{}.
  GetPredUID("q3", "name", "Dio")).Run(txn)
```

Note, that query blocks have to be named uniquely.

## Predefined queries

There are some predefined queries. They can be Joined and Run just like shown above.

Predefined queries and mutations:

```go
func (Query) GetUIDExpandAll(queryID, uid string) QueryJSON {...}
func (Query) GetPredExpandAll(queryID, pred, val string) QueryJSON {...}
func (Query) GetPredExpandAllLevel2(queryID, pred, val string) QueryJSON {...}
func (Query) GetPredUID(queryID, pred, val string) QueryJSON {...}
func (Query) HasPredExpandAll(queryID, pred string) QueryJSON {...}

func (Query) DeleteNode(uid string) DeleteJSON {...}
func (Query) DeleteEdge(from, predicate, to string) DeleteJSON {...}
```

# Other helpers

### FlattenJSON

Sometimes, when querying dgraph, we want just 1 resulting element, instead of a nested array.

```go
// Instead of having this:
type Queries struct {
  Q []struct {
    UID string `json:"uid"`
  } `json:"q"`
}
// We can have:
type MyUID struct {
  UID string `json:"uid"`
}
```
```go
var result MyUID
resp, err := ndgo.QueryJSON(`{q(func:uid(0x123)){uid}}`).Run(txn)
// query block name (i.e. 'q' ^) must have length of 1 for this helper to work!
if err != nil {
	log.Fatal(err) 
}
if err := json.Unmarshal(ndgo.Unsafe{}.FlattenJSON(resp.GetJson()), &result); err != nil {
	log.Fatal(err) // will fail, if result has more than 1 element!
}
log.Print(result)
```

# Future plans

* add more upsert things

# Note

Everything may or may not change ¯\\\_(ツ)\_/¯
