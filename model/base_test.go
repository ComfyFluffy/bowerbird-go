package model

import (
	"encoding/json"
	"sort"
	"testing"
)

func TestDDSort(t *testing.T) {
	d := DD{{"b", 1}, {"a", 2}}
	sort.Sort(d)
	b, err := json.Marshal(d)
	var dd DD

	t.Log(d, string(b), err, json.Unmarshal([]byte(`{"b": 11, "a": 22}`), &dd), dd)
}
