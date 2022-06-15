package gcolle

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"

	"github.com/javtube/javtube-sdk-go/common/parser"
	"github.com/javtube/javtube-sdk-go/model"
	"github.com/javtube/javtube-sdk-go/provider"
	"github.com/javtube/javtube-sdk-go/provider/internal/scraper"
)

var _ provider.MovieProvider = (*Gcolle)(nil)

const (
	Name     = "Gcolle"
	Priority = 1000
)

const (
	baseURL  = "https://gcolle.net/"
	movieURL = "https://gcolle.net/product_info.php/products_id/%s"
)

type Gcolle struct {
	*scraper.Scraper
}

func New() *Gcolle {
	return &Gcolle{scraper.NewDefaultScraper(Name, baseURL, Priority, scraper.WithDetectCharset())}
}

func (gcl *Gcolle) NormalizeID(id string) string {
	if ss := regexp.MustCompile(`^(?i)(?:GCOLLE-)?(\w+)$`).FindStringSubmatch(id); len(ss) == 2 {
		return ss[1]
	}
	return ""
}

func (gcl *Gcolle) GetMovieInfoByID(id string) (info *model.MovieInfo, err error) {
	return gcl.GetMovieInfoByURL(fmt.Sprintf(movieURL, id))
}

func (gcl *Gcolle) ParseIDFromURL(rawURL string) (string, error) {
	homepage, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return path.Base(homepage.Path), nil
}

func (gcl *Gcolle) GetMovieInfoByURL(rawURL string) (info *model.MovieInfo, err error) {
	id, err := gcl.ParseIDFromURL(rawURL)
	if err != nil {
		return
	}

	info = &model.MovieInfo{
		ID:            id,
		Number:        fmt.Sprintf("GCOLLE-%s", id),
		Provider:      gcl.Name(),
		Homepage:      rawURL,
		Actors:        []string{},
		PreviewImages: []string{},
		Tags:          []string{},
	}

	c := gcl.ClonedCollector()

	// Age check
	c.OnHTML(`#main_content > table:nth-child(5) > tbody > tr > td:nth-child(2) > table > tbody > tr > td > h4 > a:nth-child(2)`, func(e *colly.HTMLElement) {
		href := e.Attr("href")
		if !strings.Contains(href, "age_check") {
			return
		}
		d := c.Clone()
		d.OnResponse(func(r *colly.Response) {
			e.Response.Body = r.Body // Replace HTTP body
		})
		d.Visit(e.Request.AbsoluteURL(href))
	})

	// Title
	c.OnXML(`//*[@id="cart_quantity"]/table/tbody/tr[1]/td/h1`, func(e *colly.XMLElement) {
		info.Title = strings.TrimSpace(e.Text)
	})

	// Summary
	c.OnXML(`//*[@id="cart_quantity"]/table/tbody/tr[3]/td/p`, func(e *colly.XMLElement) {
		info.Summary = strings.TrimSpace(e.Text)
	})

	// Tags
	c.OnXML(`//*[@id="cart_quantity"]/table/tbody/tr[4]/td/a`, func(e *colly.XMLElement) {
		info.Tags = append(info.Tags, strings.TrimSpace(e.Text))
	})

	// Thumb+Cover
	c.OnXML(`//*[@id="cart_quantity"]/table/tbody/tr[3]/td/table/tbody/tr/td/a`, func(e *colly.XMLElement) {
		info.CoverURL = e.Request.AbsoluteURL(e.Attr("href"))
		info.ThumbURL = e.Request.AbsoluteURL(e.ChildAttr(`.//img`, "src"))
	})

	// Preview Images
	c.OnXML(`//*[@id="cart_quantity"]/table/tbody/tr[3]/td/div/img`, func(e *colly.XMLElement) {
		info.PreviewImages = append(info.PreviewImages,
			e.Request.AbsoluteURL(e.Attr("src")))
	})

	// Preview Images (extra?)
	c.OnXML(`//*[@id="cart_quantity"]/table/tbody/tr[3]/td/div/a/img`, func(e *colly.XMLElement) {
		info.PreviewImages = append(info.PreviewImages,
			e.Request.AbsoluteURL(e.Attr("src")))
	})

	// Fields
	c.OnXML(`//table[@class="filesetumei"]//tr`, func(e *colly.XMLElement) {
		switch e.ChildText(`.//td[1]`) {
		case "商品番号":
			// should use id from url.
			// info.ID = e.ChildText(`.//td[2]`)
		case "商品登録日":
			info.ReleaseDate = parser.ParseDate(e.ChildText(`.//td[2]`))
		}
	})

	err = c.Visit(info.Homepage)
	return
}

func init() {
	provider.RegisterMovieFactory(Name, New)
}