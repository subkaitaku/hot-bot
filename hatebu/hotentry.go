package hatebu

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
	"unicode/utf8"

	"database/sql"

	"github.com/mattn/go-runewidth"
	_ "github.com/mattn/go-sqlite3"
)

type HotEntry struct {
	Items []*Item `xml:"item"`
}

type Item struct {
	Title         string `xml:"title"`
	Link          string `xml:"link"`
	ImageURL      string `xml:"imageurl"`
	Description   string `xml:"description"`
	Date          string `xml:"date"`
	BookmarkCount int    `xml:"bookmarkcount"`
}

type Content struct {
	Title       string
	URL         string
	ImageURL    string
	Description string
}

type blockDomain string
type blockDomains []blockDomain

type blockWord string
type blockWords []blockWord

func httpGet(url string) string {
	response, err := http.Get(url)
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}

	defer response.Body.Close()
	return string(body)
}

func maxWidth(entries []*Item, max int) int {
	width := 0

	for _, e := range entries {
		count := utf8.RuneCountInString(e.Title)
		if count > width {
			width = count
		}

		if width > max {
			return max
		}
	}

	return width
}

func (ds blockDomains) Match(url string) bool {
	for _, d := range ds {
		if strings.Contains(url, string(d)) {
			return true
		}
	}
	return false
}

func (ws blockWords) Match(title string) bool {
	for _, w := range ws {
		if strings.Contains(title, string(w)) {
			return true
		}
	}
	return false
}

func replaceOverflowText(text string, width int) string {
	if runewidth.StringWidth(text) > width {
		return runewidth.Truncate(text, width-3, "...")
	} else {
		return text
	}
}

func RenderHotentry(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "./hotentry.db")
	if err != nil {
		fmt.Printf("sqlite open error: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS BlockDomains (domain TEXT PRIMARY KEY)")
	if err != nil {
		fmt.Printf("table error: %v", err)
		os.Exit(1)
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS BlockWords (word TEXT PRIMARY KEY)")
	if err != nil {
		fmt.Printf("table error: %v", err)
		os.Exit(1)
	}

	data := httpGet("http://b.hatena.ne.jp/hotentry/it.rss")

	hotentry := HotEntry{}

	err = xml.Unmarshal([]byte(data), &hotentry)

	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}

	titleWidth := maxWidth(hotentry.Items, 200)
	urlWidth := maxWidth(hotentry.Items, 200)
	urlFmt := fmt.Sprintf("%%-%ds", urlWidth)

	contents := []Content{}
	for _, bookmark := range hotentry.Items {
		var bds blockDomains
		rows, err := db.Query("SELECT domain FROM BlockDomains")
		if err != nil {
			fmt.Printf("select error: %v", err)
			os.Exit(1)
		}
		defer rows.Close()
		for rows.Next() {
			var domain string
			err = rows.Scan(&domain)
			if err != nil {
				fmt.Printf("scan error: %v", err)
				os.Exit(1)
			}
			bds = append(bds, blockDomain(domain))
		}
		if bds.Match(bookmark.Link) {
			continue
		}

		var bws blockWords
		rows, err = db.Query("SELECT word FROM BlockWords")
		if err != nil {
			fmt.Printf("select error: %v", err)
			os.Exit(1)
		}
		defer rows.Close()
		for rows.Next() {
			var word string
			err = rows.Scan(&word)
			if err != nil {
				fmt.Printf("scan error: %v", err)
				os.Exit(1)
			}
			bws = append(bws, blockWord(word))
		}
		if bws.Match(bookmark.Title) {
			continue
		}

		title := bookmark.Title
		link := bookmark.Link
		imageURL := bookmark.ImageURL
		description := bookmark.Description
		contents = append(contents, Content{
			runewidth.FillRight(replaceOverflowText(title, titleWidth), titleWidth),
			fmt.Sprintf(urlFmt, link),
			imageURL,
			description,
		})
	}

	htmlTemplate := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Hatebu Hotentry</title>
	</head>
	<body>
		<h1>Hatebu Hotentry</h1>
		<form action="/register" method="post">
			<label for="domain">Block Domain:</label><br>
			<input type="text" id="domain" name="domain" value=""><br>

			<label for="word">Block Word:</label><br>
			<input type="text" id="word" name="word" value=""><br>

			<input type="submit" value="Submit">
		</form>
		<ul>
			{{range .}}
				<li><a href="{{.URL}}" target="_blank">{{.Title}}</a></li>
				<p>{{.Description}}</p>
				{{if .ImageURL}}
				<img src="{{.ImageURL}}" alt="alt" width="227" height="127">
				{{else}}
				<img src="https://placehold.jp/227x127.png?text=noimage" alt="alt" width="227" height="127">
				{{end}}
			{{end}}
		</ul>
	</body>
	</html>
`

	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, contents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func RegisterBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	domain := r.FormValue("domain")
	word := r.FormValue("word")

	db, err := sql.Open("sqlite3", "./hotentry.db")
	if err != nil {
		fmt.Printf("sqlite open error: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS BlockDomains (domain TEXT PRIMARY KEY)")
	if err != nil {
		fmt.Printf("table error: %v", err)
		os.Exit(1)
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS BlockWords (word TEXT PRIMARY KEY)")
	if err != nil {
		fmt.Printf("table error: %v", err)
		os.Exit(1)
	}

	if domain != "" {
		_, err = db.Exec("INSERT INTO BlockDomains (domain) VALUES (?)", domain)
		if err != nil {
			fmt.Printf("insert error: %v", err)
			os.Exit(1)
		}
	}

	if word != "" {
		_, err = db.Exec("INSERT INTO BlockWords (word) VALUES (?)", word)
		if err != nil {
			fmt.Printf("insert error: %v", err)
			os.Exit(1)
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
