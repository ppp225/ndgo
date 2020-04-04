package ndgo

import (
	"encoding/json"
)

// --------------------------------------- exported ---------------------------------------

// Unsafe collects helpers, which require knowledge of how they work to operate correctly
type Unsafe struct{}

// FlattenJSON flattens json []byte's by 1 level. Root array should only have 0 or 1 child - otherwise will return gibberish and panic on unmarshal.
func (Unsafe) FlattenJSON(toFlatten []byte) []byte {
	// the beginning will eitter be `{"f":[{` for results, or `{"f":[]` for empty, so using this to determine what to return
	switch {
	case toFlatten[6] == '{':
		return toFlatten[6 : len(toFlatten)-2]
	case toFlatten[6] == ']':
		return []byte{'{', '}'}
	default:
		panic(`ndgo.FlattenJSON:: query block name must be of length 1, i.e. {"f":[{"field":"42"}]}`)
	}
}

// --------------------------------------- unexported ---------------------------------------

func interfaces2Bytes(jsonMutations ...interface{}) []byte {
	allBytes := make([][]byte, len(jsonMutations))
	for i := 0; i < len(jsonMutations); i++ {
		jsonBytes, err := json.Marshal(jsonMutations[i])
		if err != nil {
			panic(err.Error()) // should never happen
		}
		allBytes[i] = jsonBytes
	}
	return byteJoinByCommaAndPutInBrackets(allBytes...)
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
