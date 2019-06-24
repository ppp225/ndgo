package ndgo_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"github.com/ppp225/ndgo"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const (
	predicateName = "testName"
	predicateAttr = "testAttribute"
	predicateEdge = "testEdge"
	firstName     = "first"
	secondName    = "second"
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
	fmt.Printf("db.cfg ip address: '%s' | Using address: '%s'\n", string(dat), ip)
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
		<testName>: string @index(hash) .
		<testAttribute>: string .
		<testEdge>: uid .
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
		"testAttribute": "%s"
	}`, firstName, firstAttr)
	_, err := txn.Set(setString1)
	require.NoError(t, err)
	// Setb
	setString2 := fmt.Sprintf(`
    {
		"testName": "%s",
		"testAttribute": "%s"
	}`, secondName, secondAttr)
	_, err = txn.Setb([]byte(setString2))
	require.NoError(t, err)

	txn.Commit()
	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()

	// Query
	queryString1 := fmt.Sprintf(`
	{
		%s(func: eq(%s, "%s")) {
			uid: uid
		},
		%s(func: eq(%s, "%s")) {
			uid: uid
		}
	}
	`, "q1", predicateName, firstName,
		"q2", predicateName, secondName)
	resp, err := txn.Query(queryString1)
	require.NoError(t, err)
	t.Logf("Query ResultJSON: %+v", string(resp.GetJson()))
	type UID struct {
		Uid string `json:"uid,omitempty"`
	}
	var decode struct {
		Q1 []UID `json:"q1"`
		Q2 []UID `json:"q2"`
	}
	err = json.Unmarshal(resp.GetJson(), &decode)
	require.NoError(t, err)
	t.Logf("Query ResultDecode: %+v", decode)
	require.Len(t, decode.Q1, 1, "should be 1")
	require.Len(t, decode.Q2, 1, "should be 1")
	uid1 := decode.Q1[0].Uid
	uid2 := decode.Q2[0].Uid

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

	// QueryWithVars
	QueryWithVarsTestQuery := `
		query withvar($testvar: string, $testvar2: string) {
			q1(func: eq(` + predicateName + `, $testvar)) {
				uid
			},
			q2(func: eq(` + predicateName + `, $testvar2)) {
				uid
			}
		}
		`
	resp, err = txn.QueryWithVars(QueryWithVarsTestQuery, map[string]string{"$testvar": firstName, "$testvar2": secondName})
	require.NoError(t, err)
	t.Logf("QueryWithVars ResultJSON: %+v", string(resp.GetJson()))
	var decodeAfter struct {
		Q1 []UID `json:"q1"`
		Q2 []UID `json:"q2"`
	}
	err = json.Unmarshal(resp.GetJson(), &decodeAfter)
	require.NoError(t, err)
	t.Logf("QueryWithVars ResultDecode: %+v", decodeAfter)
	require.Len(t, decodeAfter.Q1, 0)
	require.Len(t, decodeAfter.Q2, 0)

	txn.Commit()
	require.NotZero(t, txn.GetDatabaseTime(), "transaction should take some time, thus not be 0")
	require.NotZero(t, txn.GetNetworkTime(), "transaction should take some time, thus not be 0")
}

// TestTxnErrorPaths tests txn error paths
func TestTxnErrorPaths(t *testing.T) {
	dg := dgNewClient()
	defer setupTeardown(dg)()

	txn := ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	_, err := txn.Set("")
	t.Log(err)
	require.NotEqual(t, "Transaction has already been committed or discarded", err.Error(), "")
	require.Error(t, err, "should have errored")

	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	_, err = txn.Delete("")
	t.Log(err)
	require.NotEqual(t, "Transaction has already been committed or discarded", err.Error(), "")
	require.Error(t, err, "should have errored")

	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	_, err = txn.Query("")
	t.Log(err)
	require.NotEqual(t, "Transaction has already been committed or discarded", err.Error(), "")
	require.Error(t, err, "should have errored")

	_, err = txn.QueryWithVars("", nil)
	t.Log(err)
	require.NotEqual(t, "Transaction has already been committed or discarded", err.Error(), "")
	require.Error(t, err, "should have errored")
}

// --------------------------------------------------------------------- Test Query{} ---------------------------------------------------------------------

type testObject struct {
	Name string       `json:"testName,omitempty"`
	Attr string       `json:"testAttribute,omitempty"`
	Edge []testObject `json:"testEdge,omitempty"`
}

func setNode(name, attr string) ndgo.SetJSON {
	return ndgo.SetJSON(fmt.Sprintf(`
    {
		"testName": "%s",
		"testAttribute": "%s"
	}`, name, attr))
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
	assigned, err := setNode(firstName, firstAttr).Run(txn)
	require.NoError(t, err)
	uid := assigned.Uids["blank-0"]
	t.Logf("Assigned uid %+v ", uid)
	return uid
}

func populateDBComplex(txn *ndgo.Txn, t *testing.T) (string, string) {
	obj1 := setNode(firstName, firstAttr)
	obj2 := setNode(secondName, secondAttr)

	assigned, err := obj1.Run(txn)
	require.NoError(t, err)
	uid1 := assigned.Uids["blank-0"]
	assigned, err = obj2.Run(txn)
	require.NoError(t, err)
	uid2 := assigned.Uids["blank-0"]

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
	_, err := setNode(firstName, firstAttr).Join(
		setNode(secondName, secondAttr)).Run(txn)

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

func BenchmarkTxnRW(b *testing.B) {
	dg := dgNewClient()
	defer setupTeardown(dg)()
	// insert data
	txn := ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()
	_, err := setNode(firstName, firstAttr).Run(txn)
	if err != nil {
		b.Fatal("mutation failed")
	}
	txn.Commit()

	txn = ndgo.NewTxn(dg.NewTxn())
	defer txn.Discard()

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
	_, err := setNode(firstName, firstAttr).Run(txn)
	if err != nil {
		b.Fatal("mutation failed")
	}
	txn.Commit()

	txn = ndgo.NewTxn(dg.NewReadOnlyTxn())
	defer txn.Discard()

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
