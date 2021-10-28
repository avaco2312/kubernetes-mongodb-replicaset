package main

import (
	"log"
	"fmt"
	"gopkg.in/mgo.v2"
	"net/http"
	"github.com/gorilla/mux"
)

var
	sep *mgo.Session

type Persona struct {
	Nombre string
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", getFunc).Methods("GET")
	r.HandleFunc("/{pers}", postFunc).Methods("POST")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func postFunc(w http.ResponseWriter, r *http.Request) {
	ses := sep.Copy()
	vars := mux.Vars(r)
 	pers := Persona{Nombre: vars["pers"]}
	err := ses.DB("prueba").C("personas").Insert(&pers)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)		
	}
	fmt.Fprintf(w, "OK")
}

func getFunc(w http.ResponseWriter, r *http.Request) {
	ses := sep.Copy()
	var personas []Persona
	err := ses.DB("prueba").C("personas").Find(nil).All(&personas)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)	
	}
	fmt.Fprint(w, personas)
}

func init() {
	var err error
	sep, err = mgo.Dial("mongodb://mongo-0:27017,mongo-1:27017,mongo-2:27017/?replicaSet=rs0")
	if err != nil {
		log.Fatal(err)
	}
}
