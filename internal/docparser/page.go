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

func (p *Page) copy() *Page {
	return &Page{
		Category: p.Category,
		Group:    p.Group,
		Title:    p.Title,
		Weight:   p.Weight,
		Slug:     p.Slug,
		Body:     p.Body,
		Document: p.Document.copy().(*Document),
	}
}

type Pages []*Page

func (p Pages) Len() int { return len(p) }
func (p Pages) Less(i, j int) bool {
	return p[i].Weight < p[j].Weight
}
func (p Pages) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (p Pages) copy() Pages {
	res := make(Pages, len(p))
	for i, page := range p {
		res[i] = page.copy()
	}
	return res
}

type Category struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Groups Groups `json:"groups"`
}

func (c *Category) copy() *Category {
	return &Category{
		Name:   c.Name,
		Slug:   c.Slug,
		Groups: c.Groups.copy(),
	}
}

type Group struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Weight int    `json:"weight"`
	Pages  Pages  `json:"pages"`
}

func (g *Group) copy() *Group {
	return &Group{
		Name:   g.Name,
		Slug:   g.Slug,
		Weight: g.Weight,
		Pages:  g.Pages.copy(),
	}
}

type Groups []*Group

func (g Groups) Len() int { return len(g) }
func (g Groups) Less(i, j int) bool {
	return g[i].Weight < g[j].Weight
}
func (g Groups) Swap(i, j int) { g[i], g[j] = g[j], g[i] }

func (g *Groups) copy() Groups {
	res := Groups{}
	for _, group := range *g {
		res = append(res, group.copy())
	}
	return res
}
