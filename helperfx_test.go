package ndgo_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ppp225/ndgo/v4"
	"github.com/stretchr/testify/require"
)

func TestFlattenRespToObject(t *testing.T) {
	var testData = []struct {
		in    []byte
		out   []byte
		panic bool
	}{
		{
			in:  []byte(`{"f":[{"testName":"first"}]}`),
			out: []byte(`{"testName":"first"}`),
		},
		{
			in:  []byte(`{"s":[{"s":"s"}]}`),
			out: []byte(`{"s":"s"}`),
		},
		{
			in:  []byte(`{"e":[]}`),
			out: []byte(`{}`),
		},
		{
			in:  []byte(`{"f":[{"testName":"first"},{"testName":"second"}]}`),
			out: []byte(`{"testName":"first"},{"testName":"second"}`),
		},
		{
			in:    []byte(`{"toolongname":[{"testName":"first"}]}`),
			out:   []byte(``),
			panic: true,
		},
	}

	for i, tt := range testData {
		if tt.panic {
			require.Panics(t, func() { ndgo.Unsafe{}.FlattenRespToObject(tt.in) }, "Should have panicked")
		} else {
			require.Exactly(t, tt.out, ndgo.Unsafe{}.FlattenRespToObject(tt.in), "Test i=%d", i)
		}
	}
}

func TestFlattenRespToArray(t *testing.T) {
	var testData = []struct {
		in    []byte
		out   []byte
		panic bool
	}{
		{
			in:  []byte(`{"f":[{"testName":"first"}]}`),
			out: []byte(`[{"testName":"first"}]`),
		},
		{
			in:  []byte(`{"s":[{"s":"s"}]}`),
			out: []byte(`[{"s":"s"}]`),
		},
		{
			in:  []byte(`{"e":[]}`),
			out: []byte(`[]`),
		},
		{
			in:  []byte(`{"f":[{"testName":"first"},{"testName":"second"}]}`),
			out: []byte(`[{"testName":"first"},{"testName":"second"}]`),
		},
		{
			in:    []byte(`{"toolongname":[{"testName":"first"}]}`),
			out:   []byte(``),
			panic: true,
		},
	}

	for i, tt := range testData {
		if tt.panic {
			require.Panics(t, func() { ndgo.Unsafe{}.FlattenRespToArray(tt.in) }, "Should have panicked")
		} else {
			require.Exactly(t, tt.out, ndgo.Unsafe{}.FlattenRespToArray(tt.in), "Test i=%d", i)
		}
	}
}

func BenchmarkFlattenRespToObject(b *testing.B) {
	data := []byte(`{"f":[{"testName":"first"}]}`)
	for n := 0; n < b.N; n++ {
		_ = ndgo.Unsafe{}.FlattenRespToObject(data)
	}
}

func BenchmarkFlattenRespToObjectEmpty(b *testing.B) {
	data := []byte(`{"f":[]}`)
	for n := 0; n < b.N; n++ {
		_ = ndgo.Unsafe{}.FlattenRespToObject(data)
	}
}

func TestFlattenRespToObjectWithDgraph(t *testing.T) {
	dg := dgNewClient()
	defer setupTeardown(dg)()
	// insert data and commit, so indexing works on queries
	txn := ndgo.NewTxnWithoutContext(dg.NewTxn())
	defer txn.Discard()
	populateDBComplex(txn, t)
	txn.Commit()
	// pre
	txn = ndgo.NewTxnWithoutContext(dg.NewTxn())
	defer txn.Discard()
	// check one result
	q := fmt.Sprintf(`
	{
		f(func: eq(`+predicateName+`, "%[1]s")) { testName }
	}
	`, firstName)

	resp, err := txn.Query(q)
	require.NoError(t, err)

	var decode testObject
	err = json.Unmarshal(ndgo.Unsafe{}.FlattenRespToObject(resp.GetJson()), &decode)
	require.NoError(t, err)
	require.Equal(t, firstName, decode.Name)

	// check empty
	q2 := fmt.Sprintf(`
	{
		f(func: has(` + predicateName + `))
	}
	`)

	resp, err = txn.Query(q2)
	require.NoError(t, err)

	var decode2 testObject
	err = json.Unmarshal(ndgo.Unsafe{}.FlattenRespToObject(resp.GetJson()), &decode2)
	require.NoError(t, err)
	require.Equal(t, "", decode2.Name)
}
