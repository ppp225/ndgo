package ndgo

import "encoding/json"

// --------------------------------------- exported ---------------------------------------

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
