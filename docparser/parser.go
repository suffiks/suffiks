package docparser

import (
	"io/fs"
	"regexp"
	"sort"
	"strings"

	"github.com/adrg/frontmatter"
)

type Category struct {
	Name   string   `json:"name"`
	Slug   string   `json:"slug"`
	Groups []*Group `json:"groups"`
}

type Group struct {
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Pages Pages  `json:"pages"`
}

func Parse(files fs.FS) ([]*Category, error) {
	categories := map[string]*Category{}
	groups := map[*Category]map[string]*Group{}

	res := []*Category{}

	fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		page, err := readFile(files, path)
		if err != nil {
			return err
		}

		cat, ok := categories[page.Category]
		if !ok {
			cat = &Category{Name: page.Category}
			categories[page.Category] = cat
			res = append(res, cat)
		}

		group, ok := groups[cat][page.Group]
		if !ok {
			group = &Group{Name: page.Group}
			if _, ok := groups[cat]; !ok {
				groups[cat] = map[string]*Group{}
			}
			groups[cat][page.Group] = group
			cat.Groups = append(cat.Groups, group)
		}

		group.Pages = append(group.Pages, page)
		return err
	})

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
	return res, nil
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

func readFile(files fs.FS, path string) (*Page, error) {
	page := &Page{}
	f, err := files.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	page.Body, err = frontmatter.Parse(f, &page)
	if err != nil {
		return nil, err
	}
	return page, nil
}
