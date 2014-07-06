package article

import (
	"html/template"
	"os"
	"time"
	"bytes"
	"strings"
	"bufio"
	"io"
)





var  htmlPath = "./tlp.html"
var tail = `		  <!--appendend-->
		</div>
	  </div>
	</div>
  </div>
</body>`



func seekPosition(file string) int{
	fhtml,_ := os.Open(file)
	defer fhtml.Close()
	r := bufio.NewReader(fhtml)
	offset := 0
	for {
		line ,err := r.ReadString('\n')
		if err!=nil {
			break
		}

		l := strings.Trim(line," \n\t")
		println(l)
		if l==`<!--appendend-->` {
			println("find append end")
			break
		}
		offset += len(line)
	}
	return offset
}

func AppendArticle(file string, title string,content []byte) {
	articleDiv := `<div id="post-449" class="post-449 post type-post status-publish format-standard hentry category-64 tag-django tag-vps tag-150 tag-4">`
	articleDivEnd := `</div>`
	offset := int64(seekPosition(file))
	
	//	_,err = fw.Seek(offset,os.SEEK_SET)
	err := os.Truncate(file, offset)
	fw,err := os.OpenFile(file, os.O_RDWR|os.O_APPEND, 0666)
	if err!=nil {
		panic(err.Error())
	}

	defer fw.Close()

	if err!=nil {
		panic(err.Error())
	}

	
	
	fw.Write([]byte(articleDiv))
	fw.Write([]byte("\n"))
	titleContent(fw, title)
	genTime(fw)
	articleContent(fw, content)

	fw.Write([]byte(articleDivEnd))
	fw.Write([]byte("\n"))
	fw.WriteString(tail)
	
}

type ATitle struct {
	Title string
}
func titleContent(w io.Writer,title string)  {
	atitle := ATitle{Title:title}
	tmpl, _ := template.New("name").Parse(`<h2 class="entry-title"> <font color="green">{{.Title}}</font> </h2>`)
	tmpl.Execute(w, atitle)
	w.Write([]byte("\n"))
}

func articleContent(w io.Writer,content []byte) string {
	br := bytes.NewReader(content)
	cr := bufio.NewReader(br)
	bdiv := `<div class="entry-content clearfix">`
	ediv := `</div>`
	begin := `<p style="padding-left: 30px;">`
	end := `</p>`
	dst := ""
	for {
		line,err := cr.ReadString('\n')
		if err!=nil {
			break
		}
		dst = dst + begin + string(line) + end
	}
	w.Write([]byte(bdiv + dst + ediv))
	w.Write([]byte("\n"))

	return  bdiv +  dst + ediv
}

type  ATime struct {
    Time string
}

func genTime(w io.Writer) {
	tm := time.Now().Format("2006-01-02 15:02")
	t := ATime{Time:tm}
	tmpl, _ := template.New("name").Parse(`<div class="entry-meta">{{.Time}}</div>`)
	err := tmpl.Execute(w, t)
	if err!=nil {
		println(err.Error())
	}

	w.Write([]byte("\n"))


}





