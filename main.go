package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/maxmcd/gitbao/builder"
	"github.com/maxmcd/gitbao/logger"
)

func main() {
	r := mux.NewRouter()
	r.StrictSlash(true)
	r.HandleFunc("/{username}/{gist-id}", CreateHandler).Methods("GET")
	//.Host("{subdomain:gist}.{host:.*}")
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

func CreateHandler(w http.ResponseWriter, req *http.Request) {

	vars := mux.Vars(req)

	gistId := vars["gist-id"]
	username := vars["username"]

	if gistId == "" || username == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404"))
		return
	}

	wlog := logger.CreateLog(w)
	wlog.Write("New bao: %s %s", gistId, username)

	w.Header().Set("Content-type", "text/html")

	err, name := builder.Build(gistId, wlog)
	if err != nil {
		wlog.Write(err.Error())
	}
	wlog.Write(name)
}
