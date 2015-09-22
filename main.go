package main

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"

	"github.com/maxmcd/gitbao/builder"
	"github.com/maxmcd/gitbao/logger"
	"github.com/maxmcd/gitbao/model"
)

var T *template.Template

func init() {
	t, err := template.ParseGlob("templates/*")
	t.Parse(
		fmt.Sprintf(`{{define "hash"}}%d{{end}}`, time.Now().Unix()),
	)
	if err != nil {
		log.Fatal(err)
	}
	T = template.Must(t, err)
}

type GistResponse struct {
	Gist      builder.GithubGist
	Name      string
	Config    string
	Filenames []string
}

func main() {
	r := mux.NewRouter()
	r.StrictSlash(true)
	r.HandleFunc("/", IndexHandler).Methods("GET")
	r.HandleFunc("/{username}/{gist-id}", GistHandler).Methods("GET")
	r.HandleFunc("/build/{gist-id}", BuildHandler).Methods("POST")
	//.Host("{subdomain:gist}.{host:.*}")
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("public/")))
	http.Handle("/", Middleware(r))
	fmt.Println("Broadcasting Kitchen on port 8000")
	http.ListenAndServe(":8000", nil)
}

func Middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Host, r.URL)
		h.ServeHTTP(w, r)
	})
}

func RenderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	T.ExecuteTemplate(w, tmpl+".html", data)
}

func IndexHandler(w http.ResponseWriter, req *http.Request) {
	RenderTemplate(w, "index", nil)
}

func GistHandler(w http.ResponseWriter, req *http.Request) {

	vars := mux.Vars(req)

	gistId := vars["gist-id"]
	username := vars["username"]

	if gistId == "" || username == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404"))
		return
	}

	build := builder.Build{
		GistId: gistId,
	}
	err := build.FetchGistData()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	var baofileConfig string
	fileList := make([]string, len(build.Gist.Files))
	i := 0
	for k, file := range build.Gist.Files {
		fmt.Println(file.Filename)
		if file.Filename == "Baofile" {
			baofileConfig = file.Content
		}

		fileList[i] = k
		i++
	}
	sort.Strings(fileList)

	var name string
	if build.Gist.Description == "" {
		if len(build.Gist.Files) > 0 {
			name = fileList[0]
		}
	} else {
		name = build.Gist.Description
	}

	response := GistResponse{
		Name:      name,
		Filenames: fileList,
		Gist:      build.Gist,
		Config:    baofileConfig,
	}
	RenderTemplate(w, "bao", response)
}

func BuildHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-type", "text/html")
	vars := mux.Vars(req)
	gistId := vars["gist-id"]

	if gistId == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404"))
		return
	}

	cfg := req.FormValue("config")
	l := logger.CreateLog(w)
	err := buildHandler(gistId, cfg, l)
	if err != nil {
		l.Write(err.Error())
	}
}

func buildHandler(gistId, cfg string, l logger.Log) (err error) {
	build, err := builder.CreateBuild(gistId, cfg, l)
	l.Write("New bao: %s", gistId)
	l.Write("Fetching gist data")
	err = build.FetchGistData()
	if err != nil {
		return err
	}
	l.Write("%d files found:", len(build.Gist.Files))

	var hasGoFiles bool
	for _, file := range build.Gist.Files {
		filenameParts := strings.Split(file.Filename, ".")
		if filenameParts[len(filenameParts)-1] == "go" {
			hasGoFiles = true
		}
		l.Write("&nbsp;&nbsp;- %s", file.Filename)
	}
	if hasGoFiles != true {
		l.Write("No Go files found in this gist. Exiting.")
		return
	}
	l.Write("Downloading gist contents for build")
	err = build.DownloadFromRepo()
	l.Write("Files downloaded in directory: %s", build.Directory)

	l.Write("Downloading Dependencies")
	err = build.DownloadDependencies()
	if err != nil {
		return
	}
	l.Write("Building")
	err = build.GoBuild()
	if err != nil {
		return
	}
	l.Write("Build successful")

	l.Write("Zipping contents")
	err = build.CreateZip()
	if err != nil {
		return
	}

	l.Write("Uploading packaged contents")
	err = build.CreateLambda()
	if err != nil {
		return
	}

	id, err := model.CreateBao(build.GistId, build.Directory)
	if err != nil {
		return
	}
	l.Write("Bao successfully published at:")
	l.Write("%s.gitbao.com", id.Hex())

	l.Write("cleaning up")
	err = build.CleanUp()
	if err != nil {
		return
	}

	l.Write(`
	<script type="text/javascript">
		console.log(parent)
		parent.postMessage("%s.gitbao.com", "*");
	</script>
		`, id.Hex())

	fmt.Println(err)

	return
}
