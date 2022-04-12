package docparser

import (
	"bytes"
	"strconv"
)

type PageBody []byte

func (p PageBody) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(string(bytes.TrimSpace(p)))), nil
}

type Page struct {
	Category string   `yaml:"category" json:"category"`
	Group    string   `yaml:"group" json:"group"`
	Title    string   `yaml:"title" json:"title"`
	Weight   int      `yaml:"weight" json:"weight"`
	Slug     string   `yaml:"slug" json:"slug"`
	Body     PageBody `json:"body"`
}

type Pages []*Page

func (p Pages) Len() int { return len(p) }
func (p Pages) Less(i, j int) bool {
	return p[i].Weight < p[j].Weight
}
func (p Pages) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
