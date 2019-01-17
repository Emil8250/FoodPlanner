package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
	ini "gopkg.in/ini.v1"
)

type Configuration struct {
	host     string
	port     int
	user     string
	password string
	dbname   string
	key      []byte
}

type Dish struct {
	ID       string
	DishName string
	RecipeId string
	ImageUri string
}

var dishes []Dish

var cfg, err = ini.Load("config.ini")
var store = sessions.NewCookieStore([]byte(cfg.Section("server").Key("key").String()))

func Connect() *sql.DB {

	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		cfg.Section("server").Key("host").String(), cfg.Section("server").Key("port").MustInt(9999), cfg.Section("server").Key("user").String(), cfg.Section("server").Key("password").String(), cfg.Section("server").Key("dbname").String())

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected!")

	return db
}

func GetDish(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "authenticated")
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	fmt.Println(vars["query"])
	//rows, err := db.Query(`SELECT * FROM dishes WHERE dishname=$1`, `Lasagne`)
	db := Connect()
	rows, err := db.Query(`SELECT * FROM dishes WHERE dishname=$1`, vars["query"])
	if err != nil {
		panic(err)
	}

	//dishes := []string{"postgres"}

	var dish Dish
	for rows.Next() {
		rows.Scan(&dish.ID, &dish.DishName, &dish.RecipeId, &dish.ImageUri)
		dishes = append(dishes, dish)
		fmt.Println(dish.ID, dish.DishName, dish.RecipeId, dish.ImageUri)
	}

	json.NewEncoder(w).Encode(dishes)
	defer db.Close()
}

func PostDish(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "cookie-name")
	session.Values["authenticated"] = true
	session.Save(r, w)
	vars := mux.Vars(r)
	fmt.Println(vars["dish"])
	//rows, err := db.Query(`SELECT * FROM dishes WHERE dishname=$1`, `Lasagne`)
	db := Connect()
	rows, err := db.Query(`INSERT INTO dishes (dishname, recipeid, imageuri) VALUES ($1, '3', 'URL HERE')`, vars["dish"])
	if err != nil {
		panic(err)
	}
	json.NewEncoder(w).Encode(rows)
	defer db.Close()
}

func secret(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "cookie-name")

	// Check if user is authenticated
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Print secret message
	fmt.Fprintln(w, "The cake is a lie!")
}

func login(w http.ResponseWriter, r *http.Request) {

	//get form data.
	r.ParseForm()
	formPass := r.Form["password"]

	//convert the password to []byte
	buf := &bytes.Buffer{}
	gob.NewEncoder(buf).Encode(formPass)
	passBytes := buf.Bytes()

	db := Connect()
	//Encrypt the recieved password.
	encPasswd := fmt.Sprintf("%x", sha256.Sum256([]byte(passBytes)))

	//Convert the username slice to string.
	usernameString := strings.Join(r.Form["username"], " ")
	rows, err := db.Query(`SELECT password FROM USERS WHERE username=$1`, usernameString)

	if err != nil {
		panic(err)
	}
	var dbPassword string
	for rows.Next() {
		rows.Scan(&dbPassword)
	}
	fmt.Printf("%s : %s", encPasswd, dbPassword)

	//The account check.
	if dbPassword == encPasswd {
		session, _ := store.Get(r, "cookie-name")
		session.Values["authenticated"] = true
		session.Save(r, w)
		fmt.Println("Success")
	} else {
		session, _ := store.Get(r, "cookie-name")
		session.Values["authenticated"] = false
		session.Save(r, w)
		fmt.Println("Failed!")
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "cookie-name")

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Save(r, w)
}

// our main function
func main() {

	router := mux.NewRouter()
	//API HANDLERS
	router.HandleFunc("/dish/q={query}", GetDish).Methods("GET")
	router.HandleFunc("/check/{password}", CheckLogin).Methods("GET")
	router.HandleFunc("/dish/insert/{dish}", PostDish).Methods("GET")

	router.HandleFunc("/secret", secret)
	router.HandleFunc("/login", login).Methods("POST")
	router.HandleFunc("/logout", logout)
	//STATIC HTML
	router.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./web"))))

	//LISTEN AND SERVE
	log.Fatal(http.ListenAndServe(":8000", router))
}
