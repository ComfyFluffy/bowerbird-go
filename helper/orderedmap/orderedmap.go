package orderedmap

import (
	"bytes"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

// inspired by https://gitlab.com/c0b/go-ordered-json/-/blob/master/ordered.go

// O is an ordered key-value map struct extends bson.D
type O bson.D

// E is a struct stores key and value. It is usually in O
type E = bson.E

func (o O) Len() int           { return len(o) }
func (o O) Swap(i, j int)      { o[i], o[j] = o[j], o[i] }
func (o O) Less(i, j int) bool { return o[i].Key < o[j].Key }

// MarshalJSON marshals O into json bytes.
func (o O) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteRune('{')
	for i, e := range o {
		if i > 0 {
			buf.WriteRune(',')
		}

		k, err := json.Marshal(e.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(k)

		buf.WriteRune(':')

		v, err := json.Marshal(e.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(v)
	}
	buf.WriteRune('}')

	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshals O from json bytes.
func (o *O) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))

	t, err := dec.Token()
	if err != nil {
		return err
	}
	r, err := parseToken(dec, t, true)
	if err != nil {
		return err
	}
	if ro, ok := r.(O); ok {
		*o = ro
	}

	return nil
}

func parseToken(dec *json.Decoder, t json.Token, requireObject bool) (interface{}, error) {
	if d, ok := t.(json.Delim); ok {
		if requireObject {
			if d != '{' {
				return nil, nil
			}
		}
		switch d {
		case '{':
			o, err := parseObject(dec)
			if err != nil {
				return nil, err
			}
			dec.Token()
			return o, nil
		case '[':
			a, err := parseArray(dec)
			if err != nil {
				return nil, err
			}
			dec.Token()
			return a, nil
		default:
			return nil, fmt.Errorf("unexpected delimiter: %q", d)
		}
	}
	return t, nil
}

func parseObject(dec *json.Decoder) (O, error) {
	o := O{}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		k := t.(string)

		t, err = dec.Token()
		if err != nil {
			return nil, err
		}

		v, err := parseToken(dec, t, false)
		if err != nil {
			return nil, err
		}
		o = append(o, bson.E{Key: k, Value: v})
	}
	return o, nil
}

func parseArray(dec *json.Decoder) ([]interface{}, error) {
	r := []interface{}{}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}

		v, err := parseToken(dec, t, false)
		if err != nil {
			return nil, err
		}
		r = append(r, v)
	}
	return r, nil
}
