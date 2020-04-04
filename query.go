package ndgo

import (
	"fmt"

	"github.com/dgraph-io/dgo/v200/protos/api"
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
}

// Join allows to join multiple json Query of same type
func (v QueryJSON) Join(json QueryJSON) QueryJSON {
	return v + "," + json
}

// --------------------------------------- predefined common queries ---------------------------------------

// Query groups. Usage: ndgo.Query{}...
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
