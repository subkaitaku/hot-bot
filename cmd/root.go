package cmd

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
)

type HotEntry struct {
	Items []*Item `xml:"item"`
}

type Item struct {
	Title         string `xml:"title"`
	Link          string `xml:"link"`
	Description   string `xml:"description"`
	Date          string `xml:"date"`
	BookmarkCount int    `xml:"bookmarkcount"`
}

type Options struct {
	popularSort bool
	linkmode    bool
}

type blockDomain string
type blockDomains []blockDomain

type blockWord string
type blockWords []blockWord

var o = &Options{}

func getSelectedUrl(hatebu string) string {
	return strings.TrimSpace(strings.Split(hatebu, "| ")[2])
}

func httpGet(url string) string {
	response, err := http.Get(url)
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(response.Body)
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

func replaceOverflowText(text string, width int) string {
	if runewidth.StringWidth(text) > width {
		return runewidth.Truncate(text, width-3, "...")
	} else {
		return text
	}
}

func orderPopular(entries []*Item) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].BookmarkCount > entries[j].BookmarkCount
	})
}

func init() {
	cobra.OnInitialize()
	RootCmd.Flags().BoolVarP(&o.popularSort, "popular", "p", false, "Order popular")
	RootCmd.Flags().BoolVarP(&o.linkmode, "linkmode", "l", false, "Enable link mode")
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

func openBrowser(url string) error {
	var openCmd string
	var args []string
	var stderr bytes.Buffer

	switch runtime.GOOS {
	case "darwin":
		openCmd = "open"
	case "windows":
		openCmd = "cmd"
		args = []string{"/c", "start"}
	case "linux": // TODO
		openCmd = "xdg-open"
	default:
		return fmt.Errorf("Not support os: %v", runtime.GOOS)
	}

	args = append(args, url)
	cmd := exec.Command(openCmd, args...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Open browser faild %v\n%v\n", err, stderr.String())
	}

	return nil
}


var RootCmd = &cobra.Command{
	Use:   "chv",
	Short: "chv is CLI client for hatebu viewer",
	Long:  "chv is CLI client for hatebu viewer",
	Run: func(cmd *cobra.Command, args []string) {
		data := httpGet("http://b.hatena.ne.jp/hotentry/it.rss")

		hotentry := HotEntry{}

		err := xml.Unmarshal([]byte(data), &hotentry)

		if err != nil {
			fmt.Printf("error: %v", err)
			os.Exit(1)
		}

		bookmarkWidth := 8
		bookmarkFmt := fmt.Sprintf("%%-%ds", bookmarkWidth)

		titleWidth := maxWidth(hotentry.Items, 50)
		titleFmt := fmt.Sprintf("%%-%ds", titleWidth)

		urlWidth := maxWidth(hotentry.Items, 100)
		urlFmt := fmt.Sprintf("%%-%ds", urlWidth)

		isPopularSort, err := cmd.Flags().GetBool("popular")
		if err != nil {
			fmt.Printf("error: %v", err)
			os.Exit(1)
		}

		if isPopularSort {
			orderPopular(hotentry.Items)
		}

		isLinkMode, err := cmd.Flags().GetBool("linkmode")
		if err != nil {
			fmt.Printf("error: %v", err)
			os.Exit(1)
		}

		if isLinkMode {
			// More space is needed in header at linkmode
			fmt.Printf(
				"   %s   | %s |  %s \n",
				fmt.Sprintf(bookmarkFmt, "Bookmark"),
				fmt.Sprintf(titleFmt, "Title"),
				fmt.Sprintf(urlFmt, "URL"),
			)

			fmt.Printf("%s\n", strings.Repeat("-", bookmarkWidth+titleWidth+urlWidth))

			var hatebuList []string

			for _, bookmark := range hotentry.Items {
				title := bookmark.Title
				link := bookmark.Link
				hatebuList = append(hatebuList, fmt.Sprintf(
					" %s | %s | %s",
					fmt.Sprintf(bookmarkFmt, strconv.Itoa(bookmark.BookmarkCount)),
					runewidth.FillRight(replaceOverflowText(title, titleWidth), titleWidth),
					runewidth.FillRight(replaceOverflowText(link, titleWidth), titleWidth),
				))
			}

			prompt := promptui.Select{
				Label: "選択した内容をブラウザで開くことができます",
				Items: hatebuList,
				Size:  30,
			}

			_, hatebu, err := prompt.Run()
			if err != nil {
				fmt.Printf("Select prompt failed %v\n", err)
				os.Exit(1)
			}

			selectUrl := getSelectedUrl(hatebu)
			urlLen := utf8.RuneCountInString(selectUrl)
			partialUrl := string([]rune(selectUrl)[:urlLen-3])

			// issue: If the partial URLs are identical, an unintended URL may be selected
			var openUrl string
			for _, h := range hotentry.Items {
				if strings.HasPrefix(h.Link, partialUrl) {
					openUrl = h.Link
					break
				}
			}

			fmt.Printf("Open URL: %s\n", openUrl)

			if err := openBrowser(openUrl); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(color.Output, " %s | %s | %s \n",
				color.YellowString(fmt.Sprintf(bookmarkFmt, "Bookmark")),
				color.CyanString(titleFmt, "Title"),
				fmt.Sprintf(urlFmt, "URL"),
			)

			fmt.Println(strings.Repeat("-", bookmarkWidth+titleWidth+urlWidth))

			for _, bookmark := range hotentry.Items {
				var bds blockDomains
				bds = []blockDomain{
					"anond.hatelabo.jp",
					"togetter.com",
					"gizmodo.jp",
					"blog.livedoor.jp",
					"twitter.com",
					"x.com",
				}
				if bds.Match(bookmark.Link) {
					continue
				}

				var bws blockWords
				bws = []blockWord{
					"ハッとした",
					"常識",
					"残念",
					"必見",
					"政治",
					"ヤバい",
					"初心者",
					"驚愕",
					"遺憾",
					"駆け出し",
					"マルチ",
				}
				if bws.Match(bookmark.Title) {
					continue
				}

				title := bookmark.Title
				link := bookmark.Link
				fmt.Fprintf(
					color.Output,
					" %s | %s | %s \n",
					color.YellowString(fmt.Sprintf(bookmarkFmt, strconv.Itoa(bookmark.BookmarkCount))),
					color.CyanString(runewidth.FillRight(replaceOverflowText(title, titleWidth), titleWidth)),
					fmt.Sprintf(urlFmt, link),
				)
			}
		}
	},
}
