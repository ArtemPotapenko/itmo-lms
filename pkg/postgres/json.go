package postgres

import "encoding/json"

func Marshal(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func Unmarshal[T any](raw []byte) T {
	var value T
	if len(raw) == 0 {
		return value
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		panic(err)
	}
	return value
}
