package orderedmap

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestOJSON(t *testing.T) {
	j := []string{
		`{"str":"xxx","arr":[123,456],"n":null,"obj":{},"estr":"","ei":0,"i":1}`,
		`{"str":"xxx","arr":["xxx",1234.24,{"emm":true,"qaq":null,"eobj":{},"ei":0}],"eobj":{},"estr":""}`,
		`{}`, `{"a":[[[[]]]],"b":123.34}`,
	}
	for _, s := range j {
		o := O{}
		b := []byte(s)
		err := json.Unmarshal(b, &o)
		if err != nil {
			t.Error(err)
		}
		t.Logf("%+v", o)
		bm, err := json.Marshal(&o)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(b, bm) {
			t.Error("not equal", string(bm))
		}
		t.Log(string(b))
	}
}
