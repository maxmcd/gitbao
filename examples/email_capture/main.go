package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/maxmcd/baodata"
)

func main() {

	r := mux.NewRouter()
	r.StrictSlash(true)
	r.HandleFunc("/", IndexHandler).Methods("GET")
	r.HandleFunc("/submit", SubmitHandler).Methods("POST")
	r.HandleFunc("/admin", AdminHandler).Methods("GET")
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func IndexHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(`
    <!DOCTYPE html>
    <html lang="en"><head><meta charset="UTF-8" /><title>Email Signup</title></head>
    <body>
    <h4>Enter your email, if you dare</h4>
    <form action="/submit" method="POST">
    <input type="email" name="email" />
    <input type="submit" />
    </form>
    </body>
    </html>
  `))
}

func SubmitHandler(w http.ResponseWriter, req *http.Request) {
	email := req.FormValue("email")

	_, err := baodata.Put("/users", baodata.Data{"email": email, "created_at": time.Now().String()})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte(`thanks for the email!`))
}

func AdminHandler(w http.ResponseWriter, req *http.Request) {
	admin := `
    <!DOCTYPE html>
    <html lang="en"><head><meta charset="UTF-8" /><title>Email Signup</title></head>
    <body>
    <table>
    <tr>
    <th>Email</th>
    <th>Time</th>
    </tr>
    `
	resp, err := baodata.Get("/users")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	for _, user := range resp {
		admin += fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
            </tr>
        `, user["email"], user["created_at"], user["id"])
	}
	w.Write([]byte(admin))

}
