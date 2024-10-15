package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/snipep/Ecommerce-application/pkg/handlers"
	"github.com/snipep/Ecommerce-application/pkg/repository"
)

var db *sql.DB

func initDB()  {
	var err error
	db, err = sql.Open("mysql", "root:root@(127.0.0.1:3306)/shopping?parseTime=true")
	if err != nil{
		log.Fatal(err)
	}

	if err = db.Ping();err != nil{
		log.Fatal(err)
	}
}

func main()  {
	r := mux.NewRouter()

	initDB()
	defer db.Close()

	fs :=http.FileServer(http.Dir("./static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	
	repo := repository.NewRepository(db)
	handlers := handlers.NewHandler(repo)

	//Seeding the dummy data into the database
	r.HandleFunc("/seed-products", handlers.SeedProduct).Methods("POST")
	//Handle	 the page showing all the products
	r.HandleFunc("/manageproducts", handlers.ProductPage).Methods("GET")
	//Handle the table/structure where the products will be viewed
	r.HandleFunc("/allproducts", handlers.AllProductsView).Methods("GET")
	//Present the product inside the table 
	r.HandleFunc("/products", handlers.ListProducts).Methods("GET")
	//Handle the product view
	r.HandleFunc("/products/{id}", handlers.GetProduct).Methods("GET")
	//Handle the create product page
	r.HandleFunc("/createproduct", handlers.CreatePoductView).Methods("GET")
	//Creates a new Product
	r.HandleFunc("/products", handlers.CreateProduct).Methods("POST")
	//Handle the product view template
	r.HandleFunc("/editproduct/{id}", handlers.EditProductView).Methods("GET")
	//updates the product
	r.HandleFunc("/products/{id}", handlers.UpdateProduct).Methods("PUT")
	//Deleting product
	r.HandleFunc("/products/{id}", handlers.DeleteProduct).Methods("DELETE")



	fmt.Println("Server running on Port :5000")
	http.ListenAndServe(":5000", r)
	
}