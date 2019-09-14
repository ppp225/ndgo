package ndgo

import (
	"encoding/json"
	"time"

	"github.com/dgraph-io/dgo/protos/api"
	"github.com/ppp225/go-common"
)

// Do executes a query followed by one or more mutations.
// Possible to run query without mutations, or vice versa
func (v *Txn) Do(req *api.Request) (resp *api.Response, err error) {
	t := time.Now()
	common.Log(true, "Do Req: %+v\n", req)
	resp, err = v.txn.Do(v.ctx, req)
	v.diag.addNW(t)
	if err != nil {
		return nil, err
	}
	v.diag.addDB(resp.Latency)
	common.Log(true, "Do Resp: %+v\n", resp)
	return
}

// DoSetb is equivalent to Do using mutation with SetJson
func (v *Txn) DoSetb(query string, json []byte) (resp *api.Response, err error) {
	mutations := []*api.Mutation{
		&api.Mutation{
			SetJson: json,
		},
	}
	return v.Do(&api.Request{
		Query:     query,
		Mutations: mutations,
	})
}

// DoSetbi is equivalent to DoSeti, but it uses single api.Mutation,
// as it marshalls structs into one slice of mutations
func (v *Txn) DoSetbi(query string, jsonMutations ...interface{}) (resp *api.Response, err error) {
	allBytes := make([][]byte, len(jsonMutations))
	for i := 0; i < len(jsonMutations); i++ {
		jm := jsonMutations[i]
		jsonBytes, err := json.Marshal(jm)
		if err != nil {
			return nil, err
		}
		allBytes[i] = jsonBytes
	}
	return v.DoSetb(query, byteJoinByCommaAndPutInBrackets(allBytes...))
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

func byteJoinByCommaAndPutInBrackets(vars ...[]byte) []byte {
	items := len(vars)
	varsLen := 0
	for _, v := range vars {
		varsLen += len(v)
	}

	current := 0
	res := make([]byte, varsLen+items+1)
	res[current] = '['
	current++

	i := 0
	copy(res[current:], vars[i])
	current += len(vars[i])
	i++
	for ; i < items; i++ {
		res[current] = ','
		current++
		copy(res[current:], vars[i])
		current += len(vars[i])
	}

	res[current] = ']'
	return res
}
