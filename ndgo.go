// Package ndgo <read iNDiGO> provides dgo abstractions and helpers - github.com/ppp225/ndgo
package ndgo

import (
	"context"
	"time"

	"github.com/dgraph-io/dgo/v2"
	"github.com/dgraph-io/dgo/v2/protos/api"
	log "github.com/ppp225/lvlog"
)

// --------------------------------------- debug ---------------------------------------

// Debug enables logging of all requests and responses
// Uses the default std logger
func Debug() {
	log.SetLevel(log.ALL)
}

// --------------------------------------- core ---------------------------------------

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

// Do executes a query followed by one or more mutations.
// Possible to run query without mutations, or vice versa
func (v *Txn) Do(req *api.Request) (resp *api.Response, err error) {
	t := time.Now()
	log.Tracef("Req: %s \n", req.String())
	resp, err = v.txn.Do(v.ctx, req)
	v.diag.addNW(t)
	if err != nil {
		return nil, err
	}
	v.diag.addDB(resp.Latency)
	log.Tracef("Resp: %s\n---\n", resp.String())
	return
}

// Mutate performs dgraph mutation
func (v *Txn) Mutate(mu *api.Mutation) (resp *api.Response, err error) {
	t := time.Now()
	log.Tracef("Mutate JSON: %s %s\n", string(mu.DeleteJson), string(mu.SetJson))
	resp, err = v.txn.Mutate(v.ctx, mu)
	v.diag.addNW(t)
	if err != nil {
		return nil, err
	}
	v.diag.addDB(resp.Latency)
	log.Tracef("Mutate Resp: %s\n---\n", resp.String())
	return
}

// Query performs dgraph query
func (v *Txn) Query(q string) (resp *api.Response, err error) {
	t := time.Now()
	log.Tracef("Query JSON: %s\n", q)
	resp, err = v.txn.Query(v.ctx, q)
	v.diag.addNW(t)
	if err != nil {
		return nil, err
	}
	v.diag.addDB(resp.Latency)
	log.Tracef("Query Resp: %s\n---\n", resp.String())
	return
}

// QueryWithVars performs dgraph query with vars
func (v *Txn) QueryWithVars(q string, vars map[string]string) (resp *api.Response, err error) {
	t := time.Now()
	log.Tracef("QueryWithVars JSON: %s %s\n", q, vars)
	resp, err = v.txn.QueryWithVars(v.ctx, q, vars)
	v.diag.addNW(t)
	if err != nil {
		return nil, err
	}
	v.diag.addDB(resp.Latency)
	log.Tracef("QueryWithVars Resp: %s\n---\n", resp.String())
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
	v.nwms += (float64)(time.Now().Sub(start).Nanoseconds()) / 1e6
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

// --------------------------------------- set ---------------------------------------

// Setb is equivalent to Mutate using SetJson
func (v *Txn) Setb(json []byte) (resp *api.Response, err error) {
	return v.Mutate(&api.Mutation{
		SetJson: json,
	})
}

// Set is equivalent to Mutate using SetJson
func (v *Txn) Set(json string) (resp *api.Response, err error) {
	return v.Setb([]byte(json))
}

// Seti is equivalent to Setb, but it marshalls structs into one slice of mutations
func (v *Txn) Seti(jsonMutations ...interface{}) (resp *api.Response, err error) {
	return v.DoSeti("", jsonMutations...)
}

// --------------------------------------- delete ---------------------------------------

// Deleteb is equivalent to Mutate using DeleteJson
func (v *Txn) Deleteb(json []byte) (resp *api.Response, err error) {
	return v.Mutate(&api.Mutation{
		DeleteJson: json,
	})
}

// Delete is equivalent to Mutate using DeleteJson
func (v *Txn) Delete(json string) (resp *api.Response, err error) {
	return v.Deleteb([]byte(json))
}

// Deletei is equivalent to Deleteb, but it marshalls structs into one slice of mutations
func (v *Txn) Deletei(jsonMutations ...interface{}) (resp *api.Response, err error) {
	return v.Deleteb(interfaces2Bytes(jsonMutations...))
}

// --------------------------------------- do set ---------------------------------------

// DoSetb is equivalent to Do using mutation with SetJson
func (v *Txn) DoSetb(query string, json []byte) (resp *api.Response, err error) {
	mutations := []*api.Mutation{
		{
			SetJson: json,
		},
	}
	return v.Do(&api.Request{
		Query:     query,
		Mutations: mutations,
	})
}

// DoSet is equivalent to Do using mutation with SetJson
func (v *Txn) DoSet(query string, json string) (resp *api.Response, err error) {
	return v.DoSetb(query, []byte(json))
}

// DoSetbi is equivalent to DoSeti, but it uses single api.Mutation,
// as it marshalls structs into one slice of mutations
func (v *Txn) DoSetbi(query string, jsonMutations ...interface{}) (resp *api.Response, err error) {
	return v.DoSetb(query, interfaces2Bytes(jsonMutations...))
}

// DoSeti is equivalent to Do, but it marshalls structs into mutations
func (v *Txn) DoSeti(query string, jsonMutations ...interface{}) (resp *api.Response, err error) {
	return v.DoSetbi(query, jsonMutations...)
	// TODO: uncomment when dgraph will support multiple mutations.
	// mutations := []*api.Mutation{}
	// for _, jm := range jsonMutations {
	// 	jsonBytes, err := json.Marshal(jm)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	mu := &api.Mutation{
	// 		SetJson: jsonBytes,
	// 	}
	// 	mutations = append(mutations, mu)
	// }
	// return v.Do(&api.Request{
	// 	Query:     query,
	// 	Mutations: mutations,
	// })
}

// DoSetnq is equivalent to Do using mutation with SetNquads
func (v *Txn) DoSetnq(query string, nquads string) (resp *api.Response, err error) {
	mutations := []*api.Mutation{
		{
			SetNquads: []byte(nquads),
		},
	}
	return v.Do(&api.Request{
		Query:     query,
		Mutations: mutations,
	})
}

// --------------------------------------- do delete ---------------------------------------

// TODO:
