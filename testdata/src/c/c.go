package c

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"time"
)

type Plain struct {
	Name string `json:"name"`
	N    int    `json:"n"`
}

type Slices struct {
	Tags []string `json:"tags"`
}

type Timed struct {
	D time.Duration `json:"d"`
}

var shared = json.DefaultOptionsV1()

func direct() {
	_, _ = jsonv2.Marshal(Plain{}, json.DefaultOptionsV1())        // want `Plain: Marshal with json.DefaultOptionsV1\(\).*this value uses none of the 12 options`
	_, _ = jsonv2.Marshal(Slices{}, json.DefaultOptionsV1())       // want `Slices: Marshal with json.DefaultOptionsV1\(\).*this value needs jsonv2.FormatNilSliceAsNull; pass these instead; not needed by this value: jsonv2.FormatNilMapAsNull`
	_ = jsonv2.Unmarshal(nil, new(Timed), json.DefaultOptionsV1()) // want `Timed: Unmarshal with json.DefaultOptionsV1\(\).*this value needs json.FormatDurationAsNano, jsonv2.MatchCaseInsensitiveNames, json.MergeWithLegacySemantics`
}

func viaVariable() {
	_, _ = jsonv2.Marshal(Slices{}, shared) // want `Slices: Marshal with json.DefaultOptionsV1\(\)`
	local := json.DefaultOptionsV1()
	_, _ = jsonv2.Marshal(Slices{}, local) // want `Slices: Marshal with json.DefaultOptionsV1\(\)`
}

func reassigned() {
	o := json.DefaultOptionsV1()
	o = jsonv2.JoinOptions(o, jsonv2.Deterministic(true))
	_, _ = jsonv2.Marshal(Slices{}, o) // a variable written twice is not followed
}

func noV1() {
	_, _ = jsonv2.Marshal(Slices{}) // plain v2 call: not this rule's business
	_, _ = jsonv2.Marshal(Slices{}, jsonv2.Deterministic(true))
}
