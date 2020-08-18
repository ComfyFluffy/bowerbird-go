package model

import (
	"encoding/json"
	"sort"
	"testing"
)

func TestDDSort(t *testing.T) {
	d := DD{{Key: "b", Value: 1}, {Key: "a", Value: 2}}
	sort.Sort(d)
	b, err := json.Marshal(d)
	var dd DD

	t.Log(d, string(b), err, json.Unmarshal([]byte(`{"b": 11, "a": 22}`), &dd), dd)
}
