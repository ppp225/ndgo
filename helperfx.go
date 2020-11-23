package ndgo

import (
	"encoding/json"
)

// --------------------------------------- exported ---------------------------------------

// Unsafe collects helpers, which require knowledge of how they work to operate correctly
type Unsafe struct{}

// FlattenRespToObject flattens resp.GetJson() to only contain a single object, without array ot query block.
// i.e. transforms `{"q":[{...}]}` to `{...}`.
// QueryBlockID must be single letter. One QueryBlock supported.
// Response array should only have 0 or 1 object, otherwise will return gibberish and unmarshal will error.
func (Unsafe) FlattenRespToObject(toFlatten []byte) []byte {
	switch {
	case toFlatten[6] == '{': // if results, will look like `{"q":[{`
		return toFlatten[6 : len(toFlatten)-2]
	case toFlatten[6] == ']': // if empty, will look like `{"q":[]`
		return []byte{'{', '}'}
	default:
		panic(`ndgo.Unsafe{}.FlattenJSON: query block name must be of length 1, i.e. {"q":[{"field":"42"}]}`)
	}
}

// FlattenRespToArray flattens resp.GetJson() to only contain array without query block.
// i.e. transforms `{"q":[...]}` to `[...]`.
// QueryBlockID must be single letter. One QueryBlock supported, otherwise will return gibberish and unmarshal will error.
func (Unsafe) FlattenRespToArray(bytes []byte) []byte {
	if bytes[5] == '[' {
		return bytes[5 : len(bytes)-1]
	}
	panic(`ndgo.Unsafe{}.FlattenRespToArray: query block name must be of length 1, i.e. {"q":[{"field":"42"}]}`)
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
