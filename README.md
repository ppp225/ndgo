# ndgo [![Build Status](https://travis-ci.org/ppp225/ndgo.svg?branch=master)](https://travis-ci.org/ppp225/ndgo) [![codecov](https://codecov.io/gh/ppp225/ndgo/branch/master/graph/badge.svg)](https://codecov.io/gh/ppp225/ndgo) [![Go Report Card](https://goreportcard.com/badge/github.com/ppp225/ndgo)](https://goreportcard.com/report/github.com/ppp225/ndgo)
ndgo provides [dgraph](https://github.com/dgraph-io) [dgo](https://github.com/dgraph-io/dgo) txn abstractions and helper func's.

# Why

* Reduce txn related boilerplate, thus making code more readable,
* Make using ratel like queries easier,
* Get query execution times the easy way,
* Don't do magic, keep dgo things exposed enough for them to be usable, if necessary.

# Install

```bash
go get github.com/ppp225/ndgo
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

response, err := q.Run(txn)
```

Code marshalled transactions are simplified as well:

```go
jsonBytes, err := json.Marshal(myObj)
if err != nil {
  return nil, err
}
assigned, err := txn.Setb(jsonBytes)
```

See `TestBasic` and `TestComplex` in `ndgo_test.go` for a complete example.

# ndgo.Txn

### Create a transaction:

##### ndgo
```go
dg := NewDgraphClient()
txn := ndgo.NewTxn(dg.NewTxn()) // or dg.NewReadOnlyTxn(), you can use any dgo.txn options you like. You can also use ndgo.NewTxnWithContext(ctx, txn)
defer txn.Discard()
...
err = txn.Commit()
```

##### dgo (for comparison)
```go
dg := NewDgraphClient()
ctx := context.Background()
txn := dg.NewTxn()
defer txn.Discard(ctx)
...
err = txn.Commit(v.ctx)
```

### Do mutations and queries:

```go
resp, err := txn.Set(setString)
resp, err := txn.Setb(setBytes)
resp, err := txn.Delete(deleteString)
resp, err := txn.Deleteb(deleteBytes)
resp, err := txn.Mutate(&api.Mutation{SetJson: jsonBytes})
resp, err := txn.Query(queryString)
resp, err := txn.QueryWithVars(queryWithVarsString, vars...)
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

### Flatten

Sometimes, when querying dgraph, results are nested too much, which can be de-nested one level with ndgo.Flatten:

```go
var result interface{}
resp, err := query{}.myQuery().Run(txn)
if err != nil {
	log.Fatal(err)
}
if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
	log.Fatal(err)
}
flattened := ndgo.Flatten(result)
```

# Plan

dgraph 1.1 transaction and upsert block helper.

# Note

Everything may or may not change ¯\\\_(ツ)\_/¯
