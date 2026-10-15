package main

import (
	"bytes"
	"html/template"
	"net/http"
	"os"
	"strings"
	"log"
	"time"
	"slices"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer/html"
)

type Post struct {
    Slug, Title, Summary string
	Date time.Time
}

var p = parser.New()
var r = html.New()

var layout = template.Must(template.ParseFiles("templates/post.html"))

// splitFrontMatter pulls "key: value" lines out of a leading --- block
// and returns them along with the remaining Markdown.
func splitFrontMatter(src []byte) (map[string]string, []byte) {
	meta := map[string]string{}
	s := strings.ReplaceAll(string(src), "\r\n", "\n")

	if !strings.HasPrefix(s, "---\n") {
			return meta, src
	}
	end := strings.Index(s[4:], "\n---\n")
	if end == -1 {
			return meta, src
	}

	for _, line := range strings.Split(s[4:4+end], "\n") {
			if k, v, ok := strings.Cut(line, ":"); ok {
					meta[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
	}
	return meta, []byte(s[4+end+5:])
}

func page(w http.ResponseWriter, req *http.Request) {
	name := req.PathValue("name")

	if strings.Contains(name, "/") || strings.Contains(name, "..") {
			http.NotFound(w, req)
			return
	}

	source, err := os.ReadFile("content/" + name + ".md")
	if err != nil {
			http.NotFound(w, req)
			return
	}

	meta, body := splitFrontMatter(source)

	var buf bytes.Buffer
	doc := p.Parse(body)
	if err := r.Render(&buf, body, doc); err != nil {
			http.Error(w, "could not render page", http.StatusInternalServerError)
			return
	}

	title := meta["title"]
	if title == "" {
			title = name
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := layout.Execute(w, struct {
			Title string
			Body  template.HTML
	}{
			Title: title,
			Body:  template.HTML(buf.String()),
	}); err != nil {
			log.Println(err)
	}
}

var index = template.Must(template.ParseFiles("templates/index.html"))

func home(w http.ResponseWriter, req *http.Request) {
    files, err := os.ReadDir("content")
    if err != nil {
        http.Error(w, "could not read content", http.StatusInternalServerError)
        return
    }

    var posts []Post
    for _, f := range files {
        if !strings.HasSuffix(f.Name(), ".md") {
                      continue
              }
              slug := strings.TrimSuffix(f.Name(), ".md")

              src, err := os.ReadFile("content/" + f.Name())
              if err != nil {
                      continue
              }
              meta, _ := splitFrontMatter(src)

              title := meta["title"]
              if title == "" {
                      title = slug
              }
			  date, _ := time.Parse("02.01.06", meta["date"])
              posts = append(posts, Post{Slug: slug, Title: title, Summary: meta["summary"], Date:date})
    }

	slices.SortFunc(posts, func(a, b Post) int {
    	return b.Date.Compare(a.Date)
	})
	if err := index.Execute(w, posts); err != nil {
              log.Println(err)
    }
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/{name}", page)
	http.HandleFunc("/{$}", home)
	http.ListenAndServe(":8080", nil)

}
