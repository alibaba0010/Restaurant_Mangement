package main

import (
	"encoding/json"
	"net/http"
	"fmt"
)

type User struct {
	Name string `json:"name"`
	Email string `json:"email"`
}
type Book struct {
	Name string  `json:"name"`
	Author string  `json:"author"`
}

func getUserHandler(writer http.ResponseWriter, request *http.Request){

	user:= User{
		Name: "Ali Zakariyah",
		Email: "ali@test.com",
	}
	writer.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(writer).Encode(user)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}
func GetBookHandler(writer http.ResponseWriter, request *http.Request){
	
	book:= Book{
		Name: "The Go Programming Language",	
		Author: "Ngugi Wa Thiongo",}
		writer.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(writer).Encode(book)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
}


func httpHandler(writer http.ResponseWriter, request *http.Request){
	if request.URL.Path != "/" {
		http.NotFound(writer, request)
		return
	}
	fmt.Fprintf(writer, "Hello World, Welcome to Go, The requested URL path is %s",request.URL.Path)
}