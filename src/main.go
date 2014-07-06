package main

import (
    "github.com/russross/blackfriday"
	"io/ioutil"
	"bufio"
	"bytes"
	"strings"

	"article"
)

func main() {
	input,_ := ioutil.ReadFile("blog.md")
	out := blackfriday.MarkdownCommon(input)
	b := bufio.NewReader(bytes.NewReader(out))
	title,_ := b.ReadString('\n')
	out = out[len(title):]
	title = nohtmltag(title)
	article.AppendArticle("article/tlp.html", title, out)
    return 
}

func nohtmltag(src string) string {
	dst := strings.Replace(src,"<h1>","",-1)
	dst = strings.Replace(dst,"</h1>","",-1)
	dst = strings.Trim(dst,"\n ")
	
	return dst
}




