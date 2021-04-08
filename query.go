package ndgo

import (
	"fmt"

	"github.com/dgraph-io/dgo/v210/protos/api"
)

// --------------------------------------- operation type definitions ---------------------------------------

// DeleteJSON represents a dgraph delete mutation string with some methods defined
type DeleteJSON string

// Run makes a dgraph db delete mutation (need to be in an array for Join to work)
func (v DeleteJSON) Run(t *Txn) (resp *api.Response, err error) {
	res := make([]byte, len(v)+2)
	res[0] = '['
	copy(res[1:], v)
	res[len(res)-1] = ']'
	return t.Deleteb(res, nil)
}

// Join allows to join multiple json Query of same type
func (v DeleteJSON) Join(json DeleteJSON) DeleteJSON {
	return v + "," + json
}

// DeleteRDF represents a dgraph delete rdf mutation string with some methods defined
type DeleteRDF string

// Run makes a dgraph db delete mutation
func (v DeleteRDF) Run(t *Txn) (resp *api.Response, err error) {
	return t.Deletenq(string(v))
}

// SetJSON represents a dgraph set mutation string with some methods defined
type SetJSON string

// Run makes a dgraph db set mutation (need to be in an array for Join to work)
func (v SetJSON) Run(t *Txn) (resp *api.Response, err error) {
	res := make([]byte, len(v)+2)
	res[0] = '['
	copy(res[1:], v)
	res[len(res)-1] = ']'
	return t.Setb(res, nil)
}

// Join allows to join multiple json Query of same type
func (v SetJSON) Join(json SetJSON) SetJSON {
	return v + "," + json
}

// SetRDF represents a dgraph set rdf mutation string with some methods defined
type SetRDF string

// Run makes a dgraph db set mutation
func (v SetRDF) Run(t *Txn) (resp *api.Response, err error) {
	return t.Setnq(string(v))
}

// QueryDQL represents a dgraph query string with some methods defined
type QueryDQL string

// Run makes a dgraph db query
func (v QueryDQL) Run(t *Txn) (resp *api.Response, err error) {
	res := make([]byte, len(v)+2)
	res[0] = '['
	copy(res[1:], v)
	res[len(res)-1] = ']'
	return t.Query(string(res))
}

// Join allows to join multiple json Query of same type
func (v QueryDQL) Join(json QueryDQL) QueryDQL {
	return v + "," + json
}

// --------------------------------------- predefined common queries ---------------------------------------

// Query groups. Usage: ndgo.Query{}...
// It's recommended to create your own helpers, than to use the build in ones.
type Query struct{}

// GetPredExpandType constructs a complete query. It's for convenience, so formatting can be done only once. Also one liner!
// Usage: resp, err := ndgo.Query{}.GetPredExpandType("q", "eq", predicate, value, ",first:1", "", "uid dgraph.type", dgTypes).Run(txn)
func (Query) GetPredExpandType(blockID, fx, pred, val, funcParams, directives, dgPreds, dgTypes string) QueryDQL {
	if len(val) > 1 && (val[0] == '[' || val[0] == '"') {
		// special case for when val is:
		// slice i.e. ["val1" "val4" "some other val"]
		// quoted i.e. "val1" OR "val1", "val2"
		return QueryDQL(fmt.Sprintf(`
		{
		  %s(func: %s(%s, %s)%s) %s {
			%s expand(%s)
		  }
		}
		`, blockID, fx, pred, val, funcParams, directives, dgPreds, dgTypes))
	}
	return QueryDQL(fmt.Sprintf(`
	{
	  %s(func: %s(%s, "%s")%s) %s {
	    %s expand(%s)
	  }
	}
	`, blockID, fx, pred, val, funcParams, directives, dgPreds, dgTypes))
}

// GetUIDExpandType constructs a complete query. It's for convenience, so formatting can be done only once. Also one liner!
// Usage: resp, err := ndgo.Query{}.GetUIDExpandType("q", "uid", uid, "", "", "", "_all_").Run(txn)
func (Query) GetUIDExpandType(blockID, fx, uid, funcParams, directives, dgPreds, dgTypes string) QueryDQL {
	return QueryDQL(fmt.Sprintf(`
	{
	  %s(func: %s(%s)%s) %s {
	    %s expand(%s)
	  }
	}
  `, blockID, fx, uid, funcParams, directives, dgPreds, dgTypes))
}

// DeleteEdge Usage:	_, err = ndgo.Query{}.DeleteEdge(parentUID, "edgeName", childUID).Run(txn)
func (Query) DeleteEdge(from, predicate, to string) DeleteRDF {
	if to == "*" {
		return Query{}.DeletePred(from, predicate)
	}
	return DeleteRDF(fmt.Sprintf(`<%s> <%s> <%s> .`+"\n", from, predicate, to))
}

// DeleteNode Usage: _, err = ndgo.Query{}.DeleteNode(UID).Run(txn)
func (Query) DeleteNode(uid string) DeleteRDF {
	return DeleteRDF(fmt.Sprintf(`<%s> * * .`+"\n", uid))
}

// DeletePred Usage: ndgo.Query{}.DeletePred(from, predicate)
func (Query) DeletePred(uid, predicate string) DeleteRDF {
	return DeleteRDF(fmt.Sprintf(`<%s> <%s> * .`+"\n", uid, predicate))
}

// SetEdge Usage: ndgo.Query{}.SetEdge(from, predicate, to)
func (Query) SetEdge(from, predicate, to string) SetRDF {
	return SetRDF(fmt.Sprintf(`<%s> <%s> <%s> .`+"\n", from, predicate, to))
}

// SetPred Usage: ndgo.Query{}.SetPred(uid, predicate, value)
func (Query) SetPred(uid, predicate, value string) SetRDF {
	return SetRDF(fmt.Sprintf(`<%s> <%s> "%s" .`+"\n", uid, predicate, value))
}
