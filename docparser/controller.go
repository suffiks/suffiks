package docparser

import (
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type Controller struct {
	lock sync.RWMutex
	docs map[string][]*Category
}

func NewController() *Controller {
	return &Controller{
		docs: map[string][]*Category{},
	}
}

func (c *Controller) Store(name string, pages []*Category) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.docs[name] = pages
}

func (c *Controller) Remove(name string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.docs, name)
}

func (c *Controller) GetAll() []*Category {
	c.lock.RLock()
	defer c.lock.RUnlock()

	categories := []*Category{}

	find := func(c *Category) *Category {
		for _, e := range categories {
			if e.Name == c.Name {
				return e
			}
		}
		return nil
	}

	for _, docs := range c.docs {
		for _, doc := range docs {
			parent := find(doc)
			if parent == nil {
				categories = append(categories, doc)
			} else {
				parent.Groups = append(parent.Groups, doc.Groups...)
			}
		}
	}

	for _, c := range categories {
		sort.Sort(c.Groups)
		for _, g := range c.Groups {
			sort.Sort(g.Pages)
		}
	}

	return categories
}

func (c *Controller) AddFS(name string, files fs.FS) error {
	mdFiles := [][]byte{}
	err := fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		b, err := fs.ReadFile(files, path)
		if err != nil {
			return err
		}
		mdFiles = append(mdFiles, b)
		return nil
	})
	if err != nil {
		return err
	}

	return c.Parse(name, mdFiles)
}

func (c *Controller) Parse(name string, pages [][]byte) error {
	categories := map[string]*Category{}
	groups := map[*Category]map[string]*Group{}

	res := []*Category{}

	for _, md := range pages {
		page, err := parseMD(md)
		if err != nil {
			return err
		}

		cat, ok := categories[page.Category]
		if !ok {
			cat = &Category{Name: page.Category}
			categories[page.Category] = cat
			res = append(res, cat)
		}

		groupName, weight := parseGroupName(page.Group)
		group, ok := groups[cat][groupName]
		if !ok {
			group = &Group{Name: groupName, Weight: weight}
			if _, ok := groups[cat]; !ok {
				groups[cat] = map[string]*Group{}
			}
			groups[cat][groupName] = group
			if group.Weight < 0 {
				group.Weight = weight
			}
			cat.Groups = append(cat.Groups, group)
		}

		group.Pages = append(group.Pages, page)
	}

	for _, cat := range res {
		if cat.Slug == "" {
			cat.Slug = sluggify(cat.Name)
		}

		for _, group := range cat.Groups {
			if group.Slug == "" {
				group.Slug = sluggify(group.Name)
			}

			sort.Sort(group.Pages)

			for _, page := range group.Pages {
				if page.Slug == "" {
					page.Slug = sluggify(page.Title)
				}
			}
		}
	}

	c.Store(name, res)
	return nil
}

var regGroupWeight = regexp.MustCompile(`^(.*?)(\[(\d+)])$`)

func parseGroupName(gn string) (name string, weight int) {
	sub := regGroupWeight.FindAllStringSubmatch(gn, 1)

	if len(sub) > 0 {
		weight, _ := strconv.Atoi(sub[0][3])
		return strings.TrimSpace(sub[0][1]), weight
	}

	return gn, -1
}

var (
	regSlug      = regexp.MustCompile(`[^a-zA-Z0-9-_]+`)
	regSlugClean = regexp.MustCompile(`\-{2,}`)
)

func sluggify(s string) string {
	slug := regSlug.ReplaceAllString(s, "-")
	slug = regSlugClean.ReplaceAllString(slug, "-")
	slug = strings.ToLower(slug)
	return slug
}

func parseMD(b []byte) (*Page, error) {
	md := goldmark.New(
		goldmark.WithExtensions(meta.New(meta.WithStoresInDocument()), extension.GFM, &admonition{}),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	document := md.Parser().Parse(text.NewReader(b))
	metaData := document.OwnerDocument().Meta()

	page := &Page{}
	var err error
	page.Document, err = convert(document, b)
	if err != nil {
		return nil, err
	}
	page.Category, _ = metaData["category"].(string)
	page.Group, _ = metaData["group"].(string)
	page.Title, _ = metaData["title"].(string)
	page.Weight, _ = metaData["weight"].(int)
	page.Slug, _ = metaData["slug"].(string)
	return page, nil
}
