package main

import (
    "github.com/russross/blackfriday"
	"io/ioutil"
	"bufio"
	"bytes"
	"strings"
"os"
	"article"
)

func main() {
        if (len(os.Args) <= 2) {
          println("usage: ./gitblog mdfile  index.html")
          return
        }
	input,_ := ioutil.ReadFile(os.Args[1])
	out := blackfriday.MarkdownCommon(input)
	b := bufio.NewReader(bytes.NewReader(out))
	title,_ := b.ReadString('\n')
	out = out[len(title):]
	title = nohtmltag(title)
	article.AppendArticle(os.Args[2], title, out)
        return 
}

func nohtmltag(src string) string {
	dst := strings.Replace(src,"<h1>","",-1)
	dst = strings.Replace(dst,"</h1>","",-1)
	dst = strings.Trim(dst,"\n ")
	
	return dst
}




