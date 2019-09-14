// Package ndgo <read iNDiGO> provides dgo abstractions and helper func's - github.com/ppp225/ndgo
package ndgo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"github.com/ppp225/go-common"
)

const debug = false

// Txn is a dgo.Txn wrapper with additional diagnostic data
// Helps with Queries, by providing abstractions for dgraph Query and Mutation
type Txn struct {
	diag diag
	ctx  context.Context
	txn  *dgo.Txn
}

// NewTxn creates new Txn
func NewTxn(txn *dgo.Txn) *Txn {
	return &Txn{
		ctx: context.Background(),
		txn: txn,
	}
}

// NewTxnWithContext creates new Txn (with ctx)
func NewTxnWithContext(ctx context.Context, txn *dgo.Txn) *Txn {
	return &Txn{
		ctx: ctx,
		txn: txn,
	}
}

// Discard cleans up dgo.Txn resources. Always defer this on creation.
func (v *Txn) Discard() {
	v.txn.Discard(v.ctx)
}

// Commit commits dgo.Txn
func (v *Txn) Commit() (err error) {
	t := time.Now()
	err = v.txn.Commit(v.ctx)
	v.diag.addNW(t)
	return
}

// Set is equivalent to Mutate using SetJson
func (v *Txn) Set(json string) (resp *api.Response, err error) {
	return v.Mutate(&api.Mutation{
		SetJson: []byte(json),
	})
}

// Setb is equivalent to Mutate using SetJson
func (v *Txn) Setb(json []byte) (resp *api.Response, err error) {
	return v.Mutate(&api.Mutation{
		SetJson: json,
	})
}

// Seti is equivalent to Setb, but it marshalls structs into one slice of mutations
func (v *Txn) Seti(jsonMutations ...interface{}) (resp *api.Response, err error) {
	allBytes := make([][]byte, len(jsonMutations))
	for i := 0; i < len(jsonMutations); i++ {
		jsonBytes, err := json.Marshal(jsonMutations[i])
		if err != nil {
			return nil, err
		}
		allBytes[i] = jsonBytes
	}
	return v.Setb(byteJoinByCommaAndPutInBrackets(allBytes...))
}

// Delete is equivalent to Mutate using DeleteJson
func (v *Txn) Delete(json string) (resp *api.Response, err error) {
	return v.Mutate(&api.Mutation{
		DeleteJson: []byte(json),
	})
}

// Deleteb is equivalent to Mutate using DeleteJson
func (v *Txn) Deleteb(json []byte) (resp *api.Response, err error) {
	return v.Mutate(&api.Mutation{
		DeleteJson: json,
	})
}

// Deletei is equivalent to Deleteb, but it marshalls structs into one slice of mutations
func (v *Txn) Deletei(jsonMutations ...interface{}) (resp *api.Response, err error) {
	allBytes := make([][]byte, len(jsonMutations))
	for i := 0; i < len(jsonMutations); i++ {
		jsonBytes, err := json.Marshal(jsonMutations[i])
		if err != nil {
			return nil, err
		}
		allBytes[i] = jsonBytes
	}
	return v.Deleteb(byteJoinByCommaAndPutInBrackets(allBytes...))
}

// Mutate performs dgraph mutation
func (v *Txn) Mutate(mu *api.Mutation) (resp *api.Response, err error) {
	t := time.Now()
	common.Log(debug, "Mutate JSON: %+v %+v\n", string(mu.DeleteJson), string(mu.SetJson))
	resp, err = v.txn.Mutate(v.ctx, mu)
	v.diag.addNW(t)
	if err != nil {
		return nil, err
	}
	v.diag.addDB(resp.Latency)
	common.Log(debug, "Mutate Resp: %+v\n", resp)
	return
}

// Query performs dgraph query
func (v *Txn) Query(q string) (resp *api.Response, err error) {
	t := time.Now()
	common.Log(debug, "Query JSON: %+v\n", q)
	resp, err = v.txn.Query(v.ctx, q)
	v.diag.addNW(t)
	if err != nil {
		return nil, err
	}
	v.diag.addDB(resp.Latency)
	common.Log(debug, "Query Resp: %+v\n", resp)
	return
}

// QueryWithVars performs dgraph query with vars
func (v *Txn) QueryWithVars(q string, vars map[string]string) (resp *api.Response, err error) {
	t := time.Now()
	common.Log(debug, "QueryWithVars JSON: %+v\n", q)
	resp, err = v.txn.QueryWithVars(v.ctx, q, vars)
	v.diag.addNW(t)
	if err != nil {
		return nil, err
	}
	v.diag.addDB(resp.Latency)
	common.Log(debug, "QueryWithVars Resp: %+v\n", resp)
	return
}

// --------------------------------------- diag ---------------------------------------

// diag contains diagnostic data for timing the transaction
// dbms - database total time - which sums all dgraph resp.Latency and
// nwms - newtwork total time - which is the total time until response
type diag struct {
	dbms, nwms float64
}

func (v *diag) addDB(latency *api.Latency) {
	v.dbms += v.getQueryLatency(latency)
}

func (v *diag) addNW(start time.Time) {
	v.nwms += common.ElapsedMs(start)
}

func (v *diag) getQueryLatency(latency *api.Latency) float64 {
	return (float64)((latency.EncodingNs+latency.ParsingNs+latency.ProcessingNs)/1e3) / 1e3
}

// GetDatabaseTime gets time txn spend in db
func (v *Txn) GetDatabaseTime() float64 {
	return v.diag.dbms
}

// GetNetworkTime gets total time until response
func (v *Txn) GetNetworkTime() float64 {
	return v.diag.nwms
}

// --------------------------------------- transaction type definitions ---------------------------------------

// DeleteJSON represents a dgraph delete mutation string with some methods defined
type DeleteJSON string

// Run makes a dgraph db delete mutation (need to be in an array for Join to work)
func (v DeleteJSON) Run(t *Txn) (resp *api.Response, err error) {
	res := make([]byte, len(v)+2)
	res[0] = '['
	copy(res[1:], v)
	res[len(res)-1] = ']'
	return t.Deleteb(res)
}

// Join allows to join multiple json Query of same type
func (v DeleteJSON) Join(json DeleteJSON) DeleteJSON {
	return v + "," + json
}

// SetJSON represents a dgraph set mutation string with some methods defined
type SetJSON string

// Run makes a dgraph db set mutation (need to be in an array for Join to work)
func (v SetJSON) Run(t *Txn) (resp *api.Response, err error) {
	res := make([]byte, len(v)+2)
	res[0] = '['
	copy(res[1:], v)
	res[len(res)-1] = ']'
	return t.Setb(res)
}

// Join allows to join multiple json Query of same type
func (v SetJSON) Join(json SetJSON) SetJSON {
	return v + "," + json
}

// QueryJSON represents a dgraph query string with some methods defined
type QueryJSON string

// Run makes a dgraph db query
func (v QueryJSON) Run(t *Txn) (resp *api.Response, err error) {
	res := make([]byte, len(v)+2)
	res[0] = '['
	copy(res[1:], v)
	res[len(res)-1] = ']'
	return t.Query(string(res))
	// return t.Query(string(v))
}

// Join allows to join multiple json Query of same type
func (v QueryJSON) Join(json QueryJSON) QueryJSON {
	return v + "," + json
}

// --------------------------------------- common Query ---------------------------------------

// Query groups
type Query struct{}

// GetUIDExpandAll Usage: ndgo.Query{}.GetUIDExpandAll("q", assigned.Uids["blank-0"]).Run(txn)
func (Query) GetUIDExpandAll(queryID, uid string) QueryJSON {
	return QueryJSON(fmt.Sprintf(`
  {
    %s(func: uid(%s)) {
      expand(_all_)
    }
  }
  `, queryID, uid))
}

// GetPredExpandAll Usage: ndgo.Query{}.GetPredExpandAll("q1", "userName", decode.Name).Run(txn)
func (Query) GetPredExpandAll(queryID, pred, val string) QueryJSON {
	return QueryJSON(fmt.Sprintf(`
  {
    %s(func: eq(%s, "%s")) {
      expand(_all_)
    }
  }
  `, queryID, pred, val))
}

// GetPredExpandAllLevel2 expands subnodes as well Usage: ndgo.Query{}.GetPredExpandAll("q1", "userName", decode.Name).Run(txn)
func (Query) GetPredExpandAllLevel2(queryID, pred, val string) QueryJSON {
	return QueryJSON(fmt.Sprintf(`
  {
    %s(func: eq(%s, "%s")) {
			expand(_all_) {
				expand(_all_)
			}
		}
  }
  `, queryID, pred, val))
}

// GetPredUID Usage: Query{}.GetPredUID("q1", "userID", decode.UserID).Run(txn)
func (Query) GetPredUID(queryID, pred, val string) QueryJSON {
	return QueryJSON(fmt.Sprintf(`
  {
    %s(func: eq(%s, "%s")) {
      uid: uid
    }
  }
  `, queryID, pred, val))
}

// HasPredExpandAll Usage: Query{}.HasPredExpandAll("q1", "userID", decode.UserID).Run(txn)
func (Query) HasPredExpandAll(queryID, pred string) QueryJSON {
	return QueryJSON(fmt.Sprintf(`
  {
    %s(func: has(%s)) {
      expand(_all_)
    }
  }
  `, queryID, pred))
}

// DeleteNode Usage: _, err = ndgo.Query{}.DeleteNode(UID).Run(txn)
func (Query) DeleteNode(uid string) DeleteJSON {
	return DeleteJSON(fmt.Sprintf(`
  {
    "uid": "%s"
  }
  `, uid))
}

// DeleteEdge Usage:	_, err = ndgo.Query{}.DeleteEdge(parentUID, "edgeName", childUID).Run(txn)
func (Query) DeleteEdge(from, predicate, to string) DeleteJSON {
	return DeleteJSON(fmt.Sprintf(`
  {
    "uid": "%s",
    "%s": [{
      "uid": "%s"
    }]
  }
  `, from, predicate, to))
}

// --------------------------------------- helper fx ---------------------------------------

// Flatten flattens json/struct by 1 level. Root array should only 1 child/item
func Flatten(toFlatten interface{}) (result interface{}) {
	temp := toFlatten.(map[string]interface{})
	if len(temp) > 1 {
		panic("ndgo.Flatten:: flattened json has more than 1 item, operation not supported")
	}
	for _, item := range temp {
		return item
	}
	return nil
}
