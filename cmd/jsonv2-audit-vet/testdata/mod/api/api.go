package api

import (
	"encoding/json"

	"example.com/mod/model"
)

type Reply struct {
	Pets []model.Pet `json:"pets,omitempty"`
}

func Encode(r Reply) ([]byte, error) { return json.Marshal(r) }
