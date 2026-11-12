package skip

import "encoding/json"

type T struct {
	Tags   []string          `json:"tags"`   // nil-slice is skipped, so no finding here
	Labels map[string]string `json:"labels"` // want `T.Labels: a nil map encodes as null in v1 and \{\} in v2`
}

func f(t T) { _, _ = json.Marshal(t) }
