package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	ini "gopkg.in/ini.v1"
)

type Configuration struct {
	host     string
	port     int
	user     string
	password string
	dbname   string
}

type Dish struct {
	ID       string
	DishName string
	RecipeId string
	ImageUri string
}

var dishes []Dish

func Index(w http.ResponseWriter, r *http.Request) {
	p := "./web"
	// set header
	w.Header().Set("Content-type", "text/html")
	http.ServeFile(w, r, p)
}

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

// our main function
func main() {
	router := mux.NewRouter()
	router.HandleFunc("/dish/q={query}", GetDish).Methods("GET")
	router.HandleFunc("/dish/insert/{dish}", PostDish).Methods("POST")

	dir := "./web"
	flag.StringVar(&dir, "dir", ".", "./web")
	flag.Parse()

	router.HandleFunc("/", Index)
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(dir))))

	log.Fatal(http.ListenAndServe(":8000", router))
}
