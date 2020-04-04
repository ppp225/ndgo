package ndgo_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/dgraph-io/dgo/v2"
	"github.com/dgraph-io/dgo/v2/protos/api"
	log "github.com/ppp225/lvlog"
	"github.com/ppp225/ndgo/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type testStruct struct {
	UID  string      `json:"uid,omitempty"`
	Type string      `json:"dgraph.type,omitempty"`
	Name string      `json:"testName,omitempty"`
	Attr string      `json:"testAttribute,omitempty"`
	Edge *testStruct `json:"testEdge,omitempty"`
}

const (
	predicateName = "testName"
	predicateAttr = "testAttribute"
	predicateEdge = "testEdge"
	firstName     = "first"
	secondName    = "second"
	thirdName     = "third"
	firstAttr     = "attribute"
	secondAttr    = "attributer"
	thirdAttr     = "attributest"
	dbIP          = "localhost:9080"
)

func TestDBConnection(t *testing.T) {
	dg := dgNewClient()
	setupTeardown(dg)
}

// dgNewClient creates new *dgo.Dgraph Client
func dgNewClient() *dgo.Dgraph {
	// read db ip address from db.cfg file, if it exists
	ip := dbIP
	dat, err := ioutil.ReadFile("db.cfg")
	if err == nil {
		ip = string(dat)
	}
	log.Tracef("db.cfg ip address: '%s' | Using address: '%s'\n", string(dat), ip)
	// make db connection
	conn, err := grpc.Dial(ip, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	// defer conn.Close()
	return dgo.NewDgraphClient(
		api.NewDgraphClient(conn),
	)
}

func dgAddSchema(dg *dgo.Dgraph) {
	ctx := context.Background()
	err := dg.Alter(ctx, &api.Operation{
		Schema: `
		<testName>: string @index(hash) @upsert .
		<testAttribute>: string .
		<testEdge>: [uid] .

		type TestObject {
			testName: string
			testAttribute: string
			testEdge: uid
		  }
		`,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func dgDropTestPredicates(dg *dgo.Dgraph) {
	ctx := context.Background()
	retries := 5
	for { // retry, as sometimes it races with txn.Discard. Err: "rpc error: code = Unknown desc = Pending transactions found. Please retry operation"
		err := dg.Alter(ctx, &api.Operation{
			DropAttr: predicateName,
		})
		if err != nil {
			retries--
			fmt.Printf("dgDropTestPredicates (retries left: %d) error: %+v \n", retries, err)
			time.Sleep(time.Millisecond * 101)
			if retries <= 0 {
				log.Fatalf("dgDropTestPredicates (has run out of retries) last error: %+v \n", err)
			}
			continue
		}
		break
	}
	err := dg.Alter(ctx, &api.Operation{
		DropAttr: predicateAttr,
	})
	if err != nil {
		log.Fatal(err)
	}
	err = dg.Alter(ctx, &api.Operation{
		DropAttr: predicateEdge,
	})
	if err != nil {
		log.Fatal(err)
	}
}

// Usage: defer setupTeardown(dg)()
func setupTeardown(dg *dgo.Dgraph) func() {
	// Setup
	dgAddSchema(dg)

	// Teardown
	return func() {
		dgDropTestPredicates(dg)
	}
}

// --------------------------------------------------------------------- Test Txn ---------------------------------------------------------------------

// TestTxn tests txn functions
func TestTxn(t *testing.T) {
	dg := dgNewClient()
	defer setupTeardown(dg)()
	txn := ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()

	// Set
	setString1 := fmt.Sprintf(`
    {
		"testName": "%s",
		"testAttribute": "%s",
		"dgraph.type": "TestObject"
	}`, firstName, firstAttr)
	_, err := txn.Set(setString1)
	require.NoError(t, err)
	// Setb
	setString2 := fmt.Sprintf(`
    {
		"testName": "%s",
		"testAttribute": "%s",
		"dgraph.type": "TestObject"
	}`, secondName, secondAttr)
	_, err = txn.Setb([]byte(setString2))
	require.NoError(t, err)

	// Seti
	s := testStruct{
		UID:  "_:newObj",
		Type: "TestObject",
		Name: thirdName,
		Attr: thirdAttr,
		Edge: nil,
	}
	_, err = txn.Seti(s)
	require.NoError(t, err)

	txn.Commit()
	ctx := context.Background()
	txn = ndgo.NewTxnWithContext(ctx, dg.NewTxn())
	defer txn.Discard()

	// Query
	queryString1 := fmt.Sprintf(`
	{
		%s(func: eq(%s, "%s")) {
			uid: uid
		},
		%s(func: eq(%s, "%s")) {
			uid: uid
		},
		%s(func: eq(%s, "%s")) {
			uid: uid
		}
	}
	`, "q1", predicateName, firstName,
		"q2", predicateName, secondName,
		"q3", predicateName, thirdName)
	resp, err := txn.Query(queryString1)
	require.NoError(t, err)
	t.Logf("Query ResultJSON: %+v", string(resp.GetJson()))
	type UID struct {
		Uid string `json:"uid,omitempty"`
	}
	var decode struct {
		Q1 []UID `json:"q1"`
		Q2 []UID `json:"q2"`
		Q3 []UID `json:"q3"`
	}
	err = json.Unmarshal(resp.GetJson(), &decode)
	require.NoError(t, err)
	t.Logf("Query ResultDecode: %+v", decode)
	require.Len(t, decode.Q1, 1, "should be 1")
	require.Len(t, decode.Q2, 1, "should be 1")
	require.Len(t, decode.Q3, 1, "should be 1")
	uid1 := decode.Q1[0].Uid
	uid2 := decode.Q2[0].Uid
	uid3 := decode.Q3[0].Uid

	// Delete
	deleteString1 := fmt.Sprintf(`
	{
		"uid": "%s"
	}
	`, uid1)
	_, err = txn.Delete(deleteString1)
	require.NoError(t, err)
	// Deleteb
	deleteString2 := fmt.Sprintf(`
	{
		"uid": "%s"
	}
	`, uid2)
	_, err = txn.Deleteb([]byte(deleteString2))
	require.NoError(t, err)
	// Deletei
	d := testStruct{
		UID: uid3,
	}
	_, err = txn.Deletei(d)
	require.NoError(t, err)

	// QueryWithVars
	QueryWithVarsTestQuery := `
		query withvar($testvar: string, $testvar2: string, $testvar3: string) {
			q1(func: eq(` + predicateName + `, $testvar)) {
				uid
			},
			q2(func: eq(` + predicateName + `, $testvar2)) {
				uid
			},
			q3(func: eq(` + predicateName + `, $testvar3)) {
				uid
			}
		}
	`
	resp, err = txn.QueryWithVars(QueryWithVarsTestQuery, map[string]string{"$testvar": firstName, "$testvar2": secondName, "$testvar3": thirdName})
	require.NoError(t, err)
	t.Logf("QueryWithVars ResultJSON: %+v", string(resp.GetJson()))
	var decodeAfter struct {
		Q1 []UID `json:"q1"`
		Q2 []UID `json:"q2"`
		Q3 []UID `json:"q3"`
	}
	err = json.Unmarshal(resp.GetJson(), &decodeAfter)
	require.NoError(t, err)
	t.Logf("QueryWithVars ResultDecode: %+v", decodeAfter)
	require.Len(t, decodeAfter.Q1, 0)
	require.Len(t, decodeAfter.Q2, 0)
	require.Len(t, decodeAfter.Q3, 0)

	txn.Commit()
	require.NotZero(t, txn.GetDatabaseTime(), "transaction should take some time, thus not be 0")
	require.NotZero(t, txn.GetNetworkTime(), "transaction should take some time, thus not be 0")
}

func TestTxnUpsert(t *testing.T) {
	dg := dgNewClient()
	defer setupTeardown(dg)()
	txn := ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()

	// if not found, upsert lang into db
	// query
	upsertQ := fmt.Sprintf(`
	{
		a as var(func: eq(`+predicateName+`, "%[1]s"))
		b as var(func: eq(`+predicateName+`, "%[2]s"))
	}
	`, firstName, secondName)

	// mutation
	s1 := testStruct{
		UID:  "uid(a)",
		Type: "TestObject",
		Name: firstName,
		Attr: firstAttr,
		Edge: nil,
	}
	s2 := testStruct{
		UID:  "uid(b)",
		Type: "TestObject",
		Name: secondName,
		Attr: secondAttr,
		Edge: nil,
	}
	resp, err := txn.DoSeti(upsertQ, s1, s2)
	require.NoError(t, err)
	require.Len(t, resp.Uids, 2, "Should have created 2 new nodes")

	// uida, uidb := resp.Uids["uid(a)"], resp.Uids["uid(b)"]
	resp, err = txn.DoSetbi(upsertQ, s1, s2)
	require.NoError(t, err)
	require.Len(t, resp.Uids, 0, "Should not have created any new nodes, as they are already created")

	upsertQ2 := fmt.Sprintf(`
	{
		c as var(func: eq(`+predicateName+`, "%[1]s"))
	}
	`, thirdName)
	s3 := testStruct{
		UID:  "uid(c)",
		Type: "TestObject",
		Name: thirdName,
		Attr: thirdAttr,
		Edge: nil,
	}

	jsonBytes, err := json.Marshal(s3)
	require.NoError(t, err)

	req := &api.Request{
		Query: upsertQ2,
		Mutations: []*api.Mutation{
			{
				SetJson: jsonBytes,
			},
		},
	}
	resp, err = txn.Do(req)
	require.NoError(t, err)
	require.Len(t, resp.Uids, 1, "Should have created one new node")

	nquads := `
	uid(c) <testName> "` + thirdName + `" .
	uid(c) <testAttribute> "` + thirdAttr + `" .
	`

	resp, err = txn.DoSetnq(upsertQ2, nquads)
	require.NoError(t, err)
	require.Len(t, resp.Uids, 0, "Should not have created any new nodes, as they are already created")

	setJson := `
		{
			"uid":"uid(c)",
			"dgraph.type":"TestObject",
			"testName":"third",
			"testAttribute":"attributest"
		}
	`
	resp, err = txn.DoSet(upsertQ2, setJson)
	require.NoError(t, err)
	require.Len(t, resp.Uids, 0, "Should not have created any new nodes, as they are already created")

}

// TestTxnErrorPaths tests txn error paths
func TestTxnErrorPaths(t *testing.T) {
	dg := dgNewClient()
	defer setupTeardown(dg)()

	txn := ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	_, err := txn.Set("incorrect value")
	t.Log(err)
	require.NotEqual(t, "Transaction has already been committed or discarded", err.Error(), "")
	require.Error(t, err, "should have errored")

	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	_, err = txn.Delete("incorrect value")
	t.Log(err)
	require.NotEqual(t, "Transaction has already been committed or discarded", err.Error(), "")
	require.Error(t, err, "should have errored")

	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	_, err = txn.Query("incorrect value")
	t.Log(err)
	require.NotEqual(t, "Transaction has already been committed or discarded", err.Error(), "")
	require.Error(t, err, "should have errored")

	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	_, err = txn.QueryWithVars("", nil)
	t.Log(err)
	require.NotEqual(t, "Transaction has already been committed or discarded", err.Error(), "")
	require.Error(t, err, "should have errored")

	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	_, err = txn.DoSeti("incorrect value", nil)
	t.Log(err)
	require.NotEqual(t, "Transaction has already been committed or discarded", err.Error(), "")
	require.Error(t, err, "should have errored")

	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	require.Panics(t, func() { txn.DoSeti("", make(chan int)) }, "Should have panicked on Marshal")

	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	require.Panics(t, func() { txn.Seti("", make(chan int)) }, "Should have panicked on Marshal")

	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	require.Panics(t, func() { txn.Deletei("", make(chan int)) }, "Should have panicked on Marshal")
}

// --------------------------------------------------------------------- Test Query{} ---------------------------------------------------------------------

type testObject struct {
	Name string       `json:"testName,omitempty"`
	Attr string       `json:"testAttribute,omitempty"`
	Edge []testObject `json:"testEdge,omitempty"`
}

func setNode(uid, name, attr string) ndgo.SetJSON {
	return ndgo.SetJSON(fmt.Sprintf(`
    {
		"uid": "_:%s",
		"testName": "%s",
		"testAttribute": "%s",
		"dgraph.type": "TestObject"
	}`, uid, name, attr))
}

func setEdge(from, to string) ndgo.SetJSON {
	return ndgo.SetJSON(fmt.Sprintf(`
    {
		"uid": "%s",
		"testEdge": [{
			"uid": "%s"
		  }]
	}`, from, to))
}

func populateDBSimple(txn *ndgo.Txn, t *testing.T) string {
	assigned, err := setNode("new", firstName, firstAttr).Run(txn)
	require.NoError(t, err)
	uid := assigned.Uids["new"]
	t.Logf("Assigned uid %+v ", uid)
	return uid
}

func populateDBComplex(txn *ndgo.Txn, t *testing.T) (string, string) {
	obj1 := setNode("new", firstName, firstAttr)
	obj2 := setNode("new", secondName, secondAttr)

	assigned, err := obj1.Run(txn)
	require.NoError(t, err)
	uid1 := assigned.Uids["new"]
	assigned, err = obj2.Run(txn)
	require.NoError(t, err)
	uid2 := assigned.Uids["new"]

	_, err = setEdge(uid1, uid2).Run(txn)
	require.NoError(t, err)
	t.Logf("Assigned uid1 %+v uid2 %+v", uid1, uid2)
	return uid1, uid2
}

// TestBasic tests simple "insert query delete query" flow
func TestBasic(t *testing.T) {
	dg := dgNewClient()
	defer setupTeardown(dg)()

	txn := ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()

	// insert data
	uid := populateDBSimple(txn, t)

	// query inserted
	resp, err := ndgo.Query{}.GetUIDExpandAll("q", uid).Run(txn)
	require.NoError(t, err)

	var decode struct {
		Q []testObject `json:"q"`
	}
	err = json.Unmarshal(resp.GetJson(), &decode)
	require.NoError(t, err)

	if len(decode.Q) != 1 {
		t.Logf("ResultJSON: %+v", string(resp.GetJson()))
		t.Logf("ResultDecode: %+v", decode)
		t.Errorf("TestBasic failed on insert. Query result length should be 1, but is: %d", len(decode.Q))
	}

	// delete inserted
	_, err = ndgo.Query{}.DeleteNode(uid).Run(txn)
	require.NoError(t, err)

	// query deleted
	resp, err = ndgo.Query{}.GetUIDExpandAll("q", uid).Run(txn)
	require.NoError(t, err)
	err = json.Unmarshal(resp.GetJson(), &decode)
	require.NoError(t, err)

	if len(decode.Q) > 0 {
		t.Logf("ResultJSON: %+v", string(resp.GetJson()))
		t.Logf("ResultDecode: %+v", decode)
		t.Errorf("TestBasic failed on deletion. Query result length should be 0, but is: %d", len(decode.Q))
	}
}

// TestComplex tests multi-node "insert query delete query" flow
func TestComplex(t *testing.T) {
	dg := dgNewClient()
	defer setupTeardown(dg)()

	txn := ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()

	// insert data
	uid1, uid2 := populateDBComplex(txn, t)
	_, _ = uid1, uid2
	// commit, so indexing works on queries
	txn.Commit()

	// ------------ test queries ------------
	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()

	// query HasPredExpandAll
	resp, err := ndgo.Query{}.HasPredExpandAll("q", predicateAttr).Run(txn)
	require.NoError(t, err)
	t.Logf("ResultJSON: %+v", string(resp.GetJson()))
	type decodeObj struct {
		Q []testObject `json:"q"`
	}
	decode := decodeObj{}
	err = json.Unmarshal(resp.GetJson(), &decode)
	require.NoError(t, err)
	t.Logf("ResultDecode: %+v", decode)
	require.Len(t, decode.Q, 2, "should have 2 objs")

	// query GetPredExpandAll
	decode = decodeObj{}
	resp, err = ndgo.Query{}.GetPredExpandAll("q", predicateName, secondName).Run(txn)
	require.NoError(t, err)
	t.Logf("ResultJSON: %+v", string(resp.GetJson()))
	err = json.Unmarshal(resp.GetJson(), &decode)
	require.NoError(t, err)
	t.Logf("ResultDecode: %+v", decode)
	require.Len(t, decode.Q, 1, "should have 1 obj")
	require.Equal(t, secondAttr, decode.Q[0].Attr, "attributes should match DB")

	// query GetPredExpandAllLevel2
	decode = decodeObj{}
	resp, err = ndgo.Query{}.GetPredExpandAllLevel2("q", predicateName, firstName).Run(txn)
	require.NoError(t, err)
	t.Logf("ResultJSON: %+v", string(resp.GetJson()))
	err = json.Unmarshal(resp.GetJson(), &decode)
	require.NoError(t, err)
	t.Logf("ResultDecode: %+v", decode)
	require.Len(t, decode.Q, 1, "should have 1 obj")
	require.Len(t, decode.Q[0].Edge, 1, "should have 1 obj")
	require.Equal(t, secondName, decode.Q[0].Edge[0].Name, "edge should point to obj2")

	// delete edge
	_, err = ndgo.Query{}.DeleteEdge(uid1, predicateEdge, uid2).Run(txn)
	require.NoError(t, err)

	// query GetPredExpandAllLevel2 after deletion to confirm edge is gone
	decode = decodeObj{}
	resp, err = ndgo.Query{}.GetPredExpandAllLevel2("q", predicateName, firstName).Run(txn)
	require.NoError(t, err)
	t.Logf("ResultJSON: %+v", string(resp.GetJson()))
	err = json.Unmarshal(resp.GetJson(), &decode)
	require.NoError(t, err)
	t.Logf("ResultDecode: %+v", decode)
	require.Len(t, decode.Q, 1, "should have 1 obj")
	require.Len(t, decode.Q[0].Edge, 0, "should have 0 objs, as edge was deleted")

	// query join
	type decodeObj2 struct {
		Q  []testObject `json:"q"`
		Q2 []testObject `json:"q2"`
	}
	decode2 := decodeObj2{}
	resp, err = ndgo.Query{}.
		GetPredUID("q", predicateName, firstName).Join(ndgo.Query{}.
		GetPredUID("q2", predicateName, secondName)).Run(txn)
	require.NoError(t, err)
	t.Logf("ResultJSON: %+v", string(resp.GetJson()))
	err = json.Unmarshal(resp.GetJson(), &decode2)
	require.NoError(t, err)
	t.Logf("ResultDecode: %+v", decode2)
	require.Len(t, decode2.Q, 1, "should have 1 obj")
	require.Len(t, decode2.Q2, 1, "should have 1 obj")
}

func TestJoin(t *testing.T) {
	dg := dgNewClient()
	defer setupTeardown(dg)()

	txn := ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()

	// join SetJSON
	_, err := setNode("new1", firstName, firstAttr).Join(
		setNode("new2", secondName, secondAttr)).Run(txn)
	require.NoError(t, err)

	txn.Commit()
	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()

	// join QueryJSON
	type UID struct {
		Uid string `json:"uid,omitempty"`
	}
	type decodeObj struct {
		Q  []UID `json:"q"`
		Q2 []UID `json:"q2"`
	}
	decode := decodeObj{}
	resp, err := ndgo.Query{}.
		GetPredUID("q", predicateName, firstName).Join(ndgo.Query{}.
		GetPredUID("q2", predicateName, secondName)).Run(txn)
	require.NoError(t, err)
	t.Logf("ResultJSON: %+v", string(resp.GetJson()))
	err = json.Unmarshal(resp.GetJson(), &decode)
	require.NoError(t, err)
	t.Logf("ResultDecode: %+v", decode)
	require.Len(t, decode.Q, 1, "should have 1 obj")
	require.Len(t, decode.Q2, 1, "should have 1 obj")

	// join DeleteJSON
	uid1 := decode.Q[0].Uid
	uid2 := decode.Q2[0].Uid
	_, err = ndgo.Query{}.
		DeleteNode(uid1).Join(ndgo.Query{}.
		DeleteNode(uid2)).Run(txn)
	require.NoError(t, err)

	// test if delete worked
	decode2 := decodeObj{}
	resp, err = ndgo.Query{}.
		GetPredUID("q", predicateName, firstName).Join(ndgo.Query{}.
		GetPredUID("q2", predicateName, secondName)).Run(txn)
	require.NoError(t, err)
	t.Logf("ResultJSON: %+v", string(resp.GetJson()))
	err = json.Unmarshal(resp.GetJson(), &decode2)
	require.NoError(t, err)
	t.Logf("ResultDecode: %+v", decode2)
	require.Len(t, decode2.Q, 0, "should have 0 objs")
	require.Len(t, decode2.Q2, 0, "should have 0 objs")
}

func TestLogging(t *testing.T) {
	dg := dgNewClient()
	defer setupTeardown(dg)()

	txn := ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()

	// insert data
	uid := populateDBSimple(txn, t)
	// enable logging
	ndgo.Debug()
	log.SetFlags(0)
	var logOutput bytes.Buffer
	log.SetOutput(&logOutput)
	// query inserted
	_, err := ndgo.Query{}.GetUIDExpandAll("q", uid).Run(txn)
	require.NoError(t, err)

	logString1 := `Query JSON: [`
	logString2 := `Query Resp:`
	require.Contains(t, logOutput.String(), logString1)
	require.Contains(t, logOutput.String(), logString2)
}

// --------------------------------------------------------------------- Test Flatten ---------------------------------------------------------------------

func TestFlatten(t *testing.T) {
	defer func() {
		e := recover()
		if e != nil {
			t.Errorf("Function panicked, but shouldn't have. %+v", e)
		}
	}()

	type objectWeWant struct {
		Num1, Num2 int
	}
	type objectToFlatten struct {
		Q []objectWeWant
	}

	// mimic json.Unmarshal output
	myObj := []objectWeWant{{Num1: 2, Num2: 5}}
	instance := objectToFlatten{
		myObj,
	}
	marsh, err := json.Marshal(instance)
	require.NoError(t, err)
	var unmarsh interface{}
	err = json.Unmarshal(marsh, &unmarsh)
	if err != nil {
		t.Error(err)
	}

	// flatten output
	t.Logf("Before: %+v", unmarsh)
	result := ndgo.Flatten(unmarsh)
	t.Logf("After: %+v", result)

	// check if identical to what we want
	marshMyObj, err := json.Marshal(myObj)
	require.NoError(t, err)
	t.Logf("marshMyObj: %+v", string(marshMyObj))

	marshFlattened, err := json.Marshal(result)
	require.NoError(t, err)
	t.Logf("marshFlattened: %+v", string(marshFlattened))

	require.JSONEq(t, string(marshMyObj), string(marshFlattened), "should be equal")
}

func TestFlattenPanic(t *testing.T) {
	defer func() {
		e := recover()
		if e != nil {
			t.Logf("Recovered as expected: %+v", e)
		} else {
			t.Errorf("Function should have panicked, but didn't.")
		}
	}()

	type objectWeWant struct {
		Num1, Num2 int
	}
	type objectToFlatten struct {
		Q  []objectWeWant
		Q2 []objectWeWant
	}

	// mimic json.Unmarshal output
	myObj := []objectWeWant{{Num1: 2, Num2: 5}}
	instance := objectToFlatten{
		Q:  myObj,
		Q2: myObj,
	}
	marsh, err := json.Marshal(instance)
	require.NoError(t, err)
	var unmarsh interface{}
	err = json.Unmarshal(marsh, &unmarsh)
	if err != nil {
		t.Error(err)
	}

	// flatten output
	t.Logf("Before: %+v", unmarsh)
	result := ndgo.Flatten(unmarsh)
	t.Logf("After: %+v", result)
}

func TestFlattenNil(t *testing.T) {
	defer func() {
		e := recover()
		if e != nil {
			t.Errorf("Function panicked, but shouldn't have. %+v", e)
		}
	}()

	// mimic json.Unmarshal output
	instance := make(map[string]interface{})

	t.Logf("Before: %+v", instance)
	result := ndgo.Flatten(instance)
	t.Logf("After: %+v", result)
	require.Nil(t, result, "should be nil")
}

// --------------------------------------------------------------------- Benchmarks ---------------------------------------------------------------------

// ------ RW vs RO Txn ------
func BenchmarkTxnRW(b *testing.B) {
	dg := dgNewClient()
	defer setupTeardown(dg)()
	// insert data
	txn := ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	_, err := setNode("new", firstName, firstAttr).Run(txn)
	if err != nil {
		b.Fatal("mutation failed")
	}
	err = txn.Commit()
	if err != nil {
		b.Fatal("commit failed")
	}

	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()

	time.Sleep(time.Second)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resp, err := ndgo.Query{}.GetPredUID("q", predicateName, firstName).Run(txn)
		if err != nil {
			b.Error(err)
		}
		_ = resp
	}
	b.StopTimer()
}

func BenchmarkTxnRO(b *testing.B) {
	dg := dgNewClient()
	defer setupTeardown(dg)()
	// insert data
	txn := ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	_, err := setNode("new", firstName, firstAttr).Run(txn)
	if err != nil {
		b.Fatal("mutation failed")
	}
	err = txn.Commit()
	if err != nil {
		b.Fatal("commit failed")
	}

	txn = ndgo.NewTxn(dg.NewReadOnlyTxn())
	defer txn.Discard()

	time.Sleep(time.Second)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resp, err := ndgo.Query{}.GetPredUID("q", predicateName, firstName).Run(txn)
		if err != nil {
			b.Error(err)
		}
		_ = resp
	}
	b.StopTimer()
}

func BenchmarkTxnBE(b *testing.B) {
	dg := dgNewClient()
	defer setupTeardown(dg)()
	// insert data
	txn := ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	_, err := setNode("new", firstName, firstAttr).Run(txn)
	if err != nil {
		b.Fatal("mutation failed")
	}
	err = txn.Commit()
	if err != nil {
		b.Fatal("commit failed")
	}

	txn = ndgo.NewTxn(dg.NewReadOnlyTxn().BestEffort())
	defer txn.Discard()

	time.Sleep(time.Second)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resp, err := ndgo.Query{}.GetPredUID("q", predicateName, firstName).Run(txn)
		if err != nil {
			b.Error(err)
		}
		_ = resp
	}
	b.StopTimer()
}

// ------ Casting ------

func Set(json string) (res []byte, err error) {
	return Setb([]byte(json))
}

func SetCast(json string) (res []byte, err error) {
	return []byte(json), nil
}

// Setb is equivalent to Mutate using SetJson
func Setb(json []byte) (res []byte, err error) {
	return json, nil
}

func BenchmarkCastingA(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, _ = Setb([]byte("dsadasddsa"))
	}
}

func BenchmarkCastingB(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, _ = Set("dsadasddsa")
	}
}

func BenchmarkCastingC(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, _ = SetCast("dsadasddsa")
	}
}

// ------ prepend & append ------

func BenchmarkSprint(b *testing.B) {
	str := `{		"testName": "abc",		"testAttribute": "def"	},{		"testName": "abc",		"testAttribute": "def"	}`
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		res := []byte(fmt.Sprint("[", str, "]"))
		_ = res
	}
}

func BenchmarkNaive(b *testing.B) {
	str := `{		"testName": "abc",		"testAttribute": "def"	},{		"testName": "abc",		"testAttribute": "def"	}`
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		res := []byte("[" + str + "]")
		_ = res
	}
}

func BenchmarkBytesBuffer(b *testing.B) {
	str := `{		"testName": "abc",		"testAttribute": "def"	},{		"testName": "abc",		"testAttribute": "def"	}`
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		var buffer bytes.Buffer
		buffer.WriteString("[")
		buffer.WriteString(str)
		buffer.WriteString("]")
		res := buffer.Bytes()
		_ = res
	}
}

func BenchmarkMakeSet(b *testing.B) {
	str := []byte(`{		"testName": "abc",		"testAttribute": "def"	},{		"testName": "abc",		"testAttribute": "def"	}`)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		res := make([]byte, len(str)+2)
		res[0] = '['
		for i, char := range str {
			res[i+1] = char
		}
		res[len(res)-1] = ']'
	}
}

func BenchmarkMakeCopy(b *testing.B) {
	str := `{		"testName": "abc",		"testAttribute": "def"	},{		"testName": "abc",		"testAttribute": "def"	}`
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		res := make([]byte, len(str)+2)
		res[0] = '['
		copy(res[1:], str)
		res[len(res)-1] = ']'
	}
}

func BenchmarkMakeCopyAsFx(b *testing.B) {
	strPutInBrackets := func(v string) []byte {
		res := make([]byte, len(v)+2)
		res[0] = '['
		copy(res[1:], v)
		res[len(res)-1] = ']'
		return res
	}

	str := `{		"testName": "abc",		"testAttribute": "def"	},{		"testName": "abc",		"testAttribute": "def"	}`
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		strPutInBrackets(str)
	}
}
