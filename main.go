package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"

	"github.com/maxmcd/baodata"

	"github.com/maxmcd/gitbao/builder"
	"github.com/maxmcd/gitbao/config"
	"github.com/maxmcd/gitbao/logger"
	"github.com/maxmcd/gitbao/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
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

func main() {
	r := mux.NewRouter()
	r.StrictSlash(true)

	// core application
	r.HandleFunc("/", IndexHandler).Methods("GET")
	r.HandleFunc("/admin/", AdminHandler).Methods("GET")
	r.HandleFunc("/build/{gist-id}", BuildHandler).Methods("POST")
	r.HandleFunc("/bao/{id}", BaoHandler).Methods("GET")
	if config.C["env"] == "dev" {
		r.HandleFunc("/{username}/{gist-id}", GistHandler).Methods("GET")
	} else {
		r.HandleFunc("/{username}/{gist-id}", GistHandler).Methods("GET").Host("{subdomain:gist}.{host:.*}")
	}

	// baodata
	baodata.SecretHandler(func(baoId string, secret string) (isValid bool, err error) {
		return model.ConfirmSecret(baoId, secret)
	})
	baodata.Connect()
	r.HandleFunc("/ds/", baodata.Handler)

	// static files and middleware
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("public/")))
	http.Handle("/", Middleware(r))

	fmt.Println("Broadcasting Gitbao on port 8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
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

	port := req.FormValue("port")

	if gistId == "" || username == "" {
		http.Error(w, "not found", 404)
		return
	}

	gist, err := builder.FetchGistData(gistId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var bf string
	fileList := make([]string, len(gist.Files))
	i := 0
	for k, file := range gist.Files {
		fmt.Println(file.Filename)
		if file.Filename == "Baofile" {
			bf = file.Content
		}

		fileList[i] = k
		i++
	}
	sort.Strings(fileList)

	var name string
	if gist.Description == "" {
		if len(gist.Files) > 0 {
			name = fileList[0]
		}
	} else {
		name = gist.Description
	}

	response := GistResponse{
		Name:      name,
		Filenames: fileList,
		Gist:      gist,
		Baofile:   bf,
		Port:      port,
	}
	RenderTemplate(w, "gist", response)
}

func BuildHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-type", "text/html")
	vars := mux.Vars(req)
	gistId := vars["gist-id"]

	if gistId == "" {
		http.Error(w, "not found", 404)
		return
	}

	cfg := req.FormValue("baofile")
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

	bao, err := model.CreateBao(build.GistId, build.Directory)
	if err != nil {
		return
	}

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
	err = build.CreateZip(bao)
	if err != nil {
		return
	}

	l.Write("Uploading packaged contents")
	err = build.CreateLambda()
	if err != nil {
		return
	}

	l.Write("Bao successfully published at:")
	l.Write("%s.gitbao.com", bao.ID.Hex())

	l.Write("cleaning up")
	// err = build.CleanUp()
	// if err != nil {
	// 	return
	// }

	l.Write(`
	<script type="text/javascript">
		console.log(parent)
		parent.postMessage("%s", "*");
	</script>
		`, bao.ID.Hex())

	fmt.Println(err)

	return
}

func BaoHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	baoId := vars["id"]
	bao, err := model.GetBaoById(baoId)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}

	gist, err := builder.FetchGistData(bao.GistId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	svc := cloudwatch.New(&aws.Config{Region: aws.String("us-east-1")})

	params := &cloudwatch.GetMetricStatisticsInput{
		EndTime:    aws.Time(time.Now()),
		MetricName: aws.String("Invocations"),
		Namespace:  aws.String("AWS/Lambda"),
		Period:     aws.Int64(480 * 8),
		StartTime:  aws.Time(time.Now().Add(-time.Hour * 24 * 20)),
		Statistics: []*string{
			aws.String("Sum"),
		},
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("FunctionName"),
				Value: aws.String(bao.FunctionName),
			},
		},
		Unit: aws.String("Count"),
	}
	resp, err := svc.GetMetricStatistics(params)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
	fmt.Println(err)

	response := BaoResponse{
		Id:          baoId,
		Gist:        gist,
		Bao:         bao,
		Stats:       resp.String(),
		Root:        config.C["root"],
		DateCreated: bao.Ts.Format("3:04pm Jan 2, 2006"),
	}
	RenderTemplate(w, "bao", response)
}

func AdminHandler(w http.ResponseWriter, r *http.Request) {
	username := "admin"
	password := os.Getenv("AdminPassword")

	authError := func() {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Zork\"")
		http.Error(w, "authorization failed", http.StatusUnauthorized)
	}

	if password == "" {
		authError()
		return
	}

	auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(auth) != 2 || auth[0] != "Basic" {
		authError()
		return
	}

	payload, err := base64.StdEncoding.DecodeString(auth[1])
	if err != nil {
		authError()
		return
	}

	pair := strings.SplitN(string(payload), ":", 2)
	if len(pair) != 2 || !(pair[0] == username && pair[1] == password) {
		authError()
		return
	}

	baos := model.GetAllBaos()

	// reverse
	for i, j := 0, len(baos)-1; i < j; i, j = i+1, j-1 {
		baos[i], baos[j] = baos[j], baos[i]
	}

	var adminResponse []AdminResponse
	for _, bao := range baos {
		adminResponse = append(adminResponse, AdminResponse{
			Bao:         bao,
			HexId:       bao.ID.Hex(),
			DateCreated: bao.Ts.Format("3:04pm Jan 2, 2006"),
		})
	}

	RenderTemplate(w, "admin", adminResponse)
}

type AdminResponse struct {
	Bao         model.Bao
	HexId       string
	DateCreated string
}

type GistResponse struct {
	Gist      builder.GithubGist
	Name      string
	Baofile   string
	Filenames []string
	Port      string
}

type BaoResponse struct {
	Id          string
	Gist        builder.GithubGist
	Bao         model.Bao
	Root        string
	Stats       string
	DateCreated string
}
