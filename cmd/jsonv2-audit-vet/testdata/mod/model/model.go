package model

// Pet is declared here and marshaled in package api.
type Pet struct {
	Nicknames []string `json:"nicknames"`
	Age       int      `json:"age,omitempty"`
}
