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
	Category string    `yaml:"category" json:"category"`
	Group    string    `yaml:"group" json:"group"`
	Title    string    `yaml:"title" json:"title"`
	Weight   int       `yaml:"weight" json:"weight"`
	Slug     string    `yaml:"slug" json:"slug"`
	Body     PageBody  `json:"body"`
	Document *Document `json:"document"`
}

type Pages []*Page

func (p Pages) Len() int { return len(p) }
func (p Pages) Less(i, j int) bool {
	return p[i].Weight < p[j].Weight
}
func (p Pages) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

type Category struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Groups Groups `json:"groups"`
}

type Group struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Weight int    `json:"-"`
	Pages  Pages  `json:"pages"`
}

type Groups []*Group

func (g Groups) Len() int { return len(g) }
func (g Groups) Less(i, j int) bool {
	return g[i].Weight < g[j].Weight
}
func (g Groups) Swap(i, j int) { g[i], g[j] = g[j], g[i] }
