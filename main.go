package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

type Person struct {
	gorm.Model

	Name      string
	Skills    string
	Email     string    `gorm:"typevarchar(100);unique_index"`
	Addresses []Address `gorm:"foreignKey:PersonID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;polymorphic:Owner;"`
}

type Address struct {
	gorm.Model

	PersonID uint
	City     string
	State    string
	Mobile   string `gorm:"long;unique_index"`
}

var db *gorm.DB
var err error

func main() {
	dialect := os.Getenv("DIALECT")
	host := os.Getenv("HOST")
	dbport := os.Getenv("DBPORT")
	user := os.Getenv("USER")
	dbname := os.Getenv("NAME")
	password := os.Getenv("PASSWORD")

	dbURI := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable", host, dbport, user, dbname, password)

	db, err = gorm.Open(dialect, dbURI)

	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("successfully connected to database")
	}

	defer db.Close()

	db.AutoMigrate(&Person{})
	db.AutoMigrate(&Address{})

	router := mux.NewRouter()

	router.HandleFunc("/person", getPeople).Methods("GET")
	router.HandleFunc("/person/{id}", getPerson).Methods("GET")

	router.HandleFunc("/address", getAddresses).Methods("GET")
	router.HandleFunc("/address/{id}", getAddress).Methods("GET")

	router.HandleFunc("/person", createPerson).Methods("POST")
	router.HandleFunc("/address", createAddress).Methods("POST")

	router.HandleFunc("/person/{id}", deletePerson).Methods("DELETE")
	router.HandleFunc("/address/{id}", deleteAddress).Methods("DELETE")

	router.HandleFunc("/person/{id}", updatePerson).Methods("PUT")
	router.HandleFunc("/address/{id}", updateAddress).Methods("PUT")

	log.Fatal(http.ListenAndServe(":8080", router))

}

func getPeople(w http.ResponseWriter, r *http.Request) {
	var People []Person

	db.Find(&People)

	json.NewEncoder(w).Encode(&People)
}

func getPerson(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	var Person Person
	var addresses []Address

	db.First(&Person, params["id"])
	if Person.ID == 0 {
		err = errors.New("no persons found for id")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	db.Model(&Person).Related(&addresses)
	//db.Where(&Address{PersonID: Person.ID}).Find(&addresses)
	Person.Addresses = addresses

	json.NewEncoder(w).Encode(Person)
}

func isEmailValid(e string) bool {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return emailRegex.MatchString(e)
}

func createPerson(w http.ResponseWriter, r *http.Request) {
	var Person Person

	json.NewDecoder(r.Body).Decode(&Person)
	if !isEmailValid(Person.Email) {
		err = errors.New("invalid email")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	createdPerson := db.Create(&Person)
	err = createdPerson.Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(&createdPerson)
}

func deletePerson(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	var Person Person

	id := params["id"]
	//db.First(&Person, id)
	db.Where("person_id = ?", id).Delete(Address{})
	db.Delete(&Person)
	//db.Model(&Person).Association("Addresses").Delete(&Person.Addresses)
	resp := map[string]string{"ID": id}
	respBytes, _ := json.Marshal(resp)

	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}
func updatePerson(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	var person, reqPerson Person
	id := params["id"]
	db.First(&person, id)
	if person.ID == 0 {
		err = errors.New("no persons found for id")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewDecoder(r.Body).Decode(&reqPerson)

	if !isEmailValid(reqPerson.Email) {
		err = errors.New("invalid email")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updatedPerson := db.Model(&person).Updates(reqPerson)

	err = updatedPerson.Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		json.NewEncoder(w).Encode(&person)
	}
}

func getAddresses(w http.ResponseWriter, r *http.Request) {
	var address []Address

	db.Find(&address)

	json.NewEncoder(w).Encode(&address)
}

func getAddress(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	var address Address

	db.First(&address, params["id"])
	if address == (Address{}) {
		err = errors.New("no address found for id")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(&address)
}

func createAddress(w http.ResponseWriter, r *http.Request) {
	var address Address

	json.NewDecoder(r.Body).Decode(&address)

	createdAddress := db.Create(&address)
	err = createdAddress.Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		json.NewEncoder(w).Encode(&address)
	}
}

func deleteAddress(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	var address Address
	id := params["id"]
	db.First(&address, id)
	db.Delete(&address)

	resp := map[string]string{"ID": id}
	respBytes, _ := json.Marshal(resp)

	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)

	json.NewEncoder(w).Encode(&address)
}

func updateAddress(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)

	var address, reqAddress Address
	id := params["id"]
	db.First(&address, id)
	if address == (Address{}) {
		err = errors.New("no address found for id")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewDecoder(r.Body).Decode(&reqAddress)

	updatedAddress := db.Model(&address).Updates(reqAddress)
	err = updatedAddress.Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		json.NewEncoder(w).Encode(&address)
	}
}
