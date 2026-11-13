package a

import (
	"b"
	"encoding/json"
	"io"
	"time"
)

type Pet struct {
	Name      string   `json:"name"`
	Nicknames []string `json:"nicknames"`     // want `Pet.Nicknames: a nil slice encodes as null in v1`
	Age       int      `json:"age,omitempty"` // want `Pet.Age: omitempty omits false and 0 in v1`
	Owner     *Owner   `json:"owner,omitempty"`
	secret    []int
	Skip      []int `json:"-"`
	Known     []int `json:"known,omitempty"`
}

type Owner struct {
	TTL time.Duration `json:"ttl"` // want `Owner.TTL: time.Duration is integer nanoseconds in v1`
}

type Box struct {
	Embedded
	Raw json.RawMessage `json:"raw"`
}

type Embedded struct {
	Codes map[string]int `json:"codes"` // want `Embedded.Codes: a nil map encodes as null in v1`
}

type Input struct { // want `Input: object names match field names case-insensitively in v1` `Input: a JSON null leaves a struct`
	ID string `json:"id"`
}

type self struct {
	Next *self `json:"next,omitempty"`
	N    string
}

func use(w io.Writer, r io.Reader, v any) {
	_, _ = json.Marshal(Pet{})
	_, _ = json.Marshal(&Pet{}) // same findings, reported once
	_ = json.NewEncoder(w).Encode(Box{})
	_ = json.NewDecoder(r).Decode(new(Input))
	_, _ = json.Marshal(v)          // interface: nothing to check
	_, _ = json.Marshal(b.Remote{}) // want `b.Remote.Tags: a nil slice encodes as null in v1`
	_, _ = json.Marshal([]int(nil)) // want `value passed to Marshal: a nil slice encodes as null in v1`
	_, _ = json.Marshal(self{})
	_, _ = json.Marshal(time.Now()) // custom marshaler: nothing
}
