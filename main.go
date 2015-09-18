package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"

	"github.com/maxmcd/gitbao/builder"
	"github.com/maxmcd/gitbao/logger"
)

var T *template.Template

func init() {
	t, err := template.ParseGlob("templates/*")
	if err != nil {
		log.Fatal(err)
	}
	T = template.Must(t, err)
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

	gist, err := builder.FetchGistData(gistId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	RenderTemplate(w, "bao", gist)
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

	wlog := logger.CreateLog(w)
	wlog.Write("New bao: %s", gistId)

	err, name := builder.Build(gistId, cfg, wlog)
	if err != nil {
		wlog.Write(err.Error())
	}
	wlog.Write(name)
}
