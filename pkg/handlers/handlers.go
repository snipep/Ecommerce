package handlers

import (
	"fmt"
	"html/template"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/snipep/Ecommerce-application/pkg/models"
	"github.com/snipep/Ecommerce-application/pkg/repository"
)

var tmpl *template.Template

type Handler struct {
	Repo *repository.Repoitory
}

func NewHandler(repo *repository.Repoitory) *Handler {
	return &Handler{
		Repo: repo,
	}
}

func init()  {
	templateDir := "./templates"
	pattern := filepath.Join(templateDir, "**", "*.html")
	tmpl = template.Must(template.ParseGlob(pattern))
}

type ProductCRUDTemplatData struct {
	Messages []string
	Product *models.Product 
}

func sendProductMessage(w http.ResponseWriter, message []string, product *models.Product)  {
	data := ProductCRUDTemplatData{
		Messages: message,
		Product: product,
	}
	tmpl.ExecuteTemplate(w, "messages", data)
}

func makeRange(min, max int) []int {
	rangeArray := make([]int, max - min + 1)
	for i := range rangeArray{
		rangeArray[i] = min + i
	}
	return rangeArray
}

func (h *Handler) SeedProduct(w http.ResponseWriter, r *http.Request)  {
	//Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Number of products to generate 
	numProducts := 20

	// An array of realistic product names to puck from 
	productTypes := []string{"Laptop", "Smartphone", "Tablet", "Headphone", "Speaker", "Camera", "TV", "Watch", "Printer", "Monitor"}

	for i := 0; i < numProducts; i++ {
		//Generate the random but more realistic product type
		productType := productTypes[rand.Intn(len(productTypes))]
		productName := strings.Title(faker.Word()) + " " + productType

		product := models.Product{
			ProductName: productName,
			Price: float64(rand.Intn(100000))/100,
			Description: faker.Sentence(),
			ProductImage: "placeholder.jpg",
		}
		err := h.Repo.Product.CreateProduct(&product)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error creating product %s:%v", product.ProductName, err), http.StatusInternalServerError)
			return 
		}
	}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "Successfully seeded %d dummy products", numProducts)
}

func (h *Handler) ProductPage(w http.ResponseWriter, r *http.Request)  {
	tmpl.ExecuteTemplate(w, "products", nil)
}

func (h *Handler) AllProductsView(w http.ResponseWriter, r *http.Request)  {
	tmpl.ExecuteTemplate(w, "allProducts", nil)
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request)  {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10		//Default limit
	}

	offset := (page - 1) * limit

	products, err := h.Repo.Product.ListProducts(limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return 
	}

	totalProducts, err := h.Repo.Product.GetTotalProductsCount()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return 
	}

	totalPage := int(math.Ceil(float64(totalProducts) / float64(limit)))
	previousPage := page - 1
	nextPage := page + 1 
	pageButtonsRange := makeRange(1, totalPage)

	data := struct {
		Products 			[]models.Product
		CurrentPage			int
		TotalPages			int
		Limit 				int
		PreviousPage		int
		NextPage 			int
		PageButtonsRange 	[]int
	}{
		Products: 			products,
		CurrentPage: 		page,
		TotalPages: 		totalPage,
		Limit: 				limit,
		PreviousPage: 		previousPage,
		NextPage: 			nextPage,
		PageButtonsRange: 	pageButtonsRange,
	}

	//Fake latency
	// time.Sleep(3 * time.Second)

	tmpl.ExecuteTemplate(w, "productRows", data)
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request)  {
	vars := mux.Vars(r)
	productID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return 
	}

	product, err := h.Repo.Product.GetProductByID(productID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return 
	}
	tmpl.ExecuteTemplate(w, "viewProduct", product)
}

func (h Handler) CreatePoductView(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "createProduct", nil)
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	//Parse the multipart form, 10MB max upload size
	r.ParseMultipartForm(10 << 20)

	// Initialize error messagees slice
	var responseMessage []string

	//Check for empty fields
	ProductName := r.FormValue("product_name")
	ProductPrice := r.FormValue("price")
	ProductDescription := r.FormValue("description")

	if ProductName == "" || ProductPrice == "" || ProductDescription == "" {
		responseMessage = append(responseMessage, "All field are required")
		sendProductMessage(w, responseMessage, nil)
		return
	}

	/* Process File Upload */

	//Retirve the file from the data
	file, handler, err := r.FormFile("product_image")
	if err != nil{
		if err == http.ErrMissingFile {
			responseMessage = append(responseMessage, "Select an image for the product")
		} else {
			responseMessage = append(responseMessage, "Error retrieving the file")
		}

		if len(responseMessage) > 0 {
			fmt.Println(responseMessage)
			sendProductMessage(w, responseMessage, nil)
			return 
		}
	}

	defer file.Close()

	//Generate a unique filename to prevent overwriting and conflict
	uuid, err := uuid.NewRandom()
	if err != nil{
		responseMessage = append(responseMessage, "Error generating new Unique identifier")
		return 
	}
	filename := uuid.String() + filepath.Ext(handler.Filename) // append file extension

	// Create the full path for saving the file 
	filePath := filepath.Join("static/uploads", filename)

	// Save the file to the sever 
	dst, err := os.Create(filePath)
	if err != nil {
		responseMessage = append(responseMessage, "Error saving the file")
		sendProductMessage(w, responseMessage, nil)
		return 
	}

	defer dst.Close()
	if _, err = io.Copy(dst, file);err != nil{
		responseMessage = append(responseMessage, "Error saving the file")
		sendProductMessage(w, responseMessage, nil)
		return 
	}

	price, err := strconv.ParseFloat(r.FormValue("price"), 64)
	if err != nil {
		responseMessage = append(responseMessage, "invalid price")
		sendProductMessage(w, responseMessage, nil)
		return 
	}

	product := models.Product{
		ProductName: ProductName,
		Price: price,
		Description: ProductDescription,
		ProductImage: filename,
	}

	err = h.Repo.Product.CreateProduct(&product)
	if err != nil {
		responseMessage = append(responseMessage, "Invalid price" + err.Error())
		sendProductMessage(w, responseMessage, nil)
		return 
	}

	// Fake latency 
	time.Sleep(2 * time.Second)

	sendProductMessage(w, []string{}, &product)
}

func (h *Handler) EditProductView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID, err := uuid.Parse(vars["id"])
	if err != nil{
		http.Error(w, "Invalid Product ID", http.StatusBadRequest)
		return 
	}
	product, err := h.Repo.Product.GetProductByID(productID)
	if err != nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	tmpl.ExecuteTemplate(w, "editProduct", product)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return 
	}

	err = r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Initialize error messagees slice
	var responseMessage []string

	//Check for empty fields
	ProductName := r.FormValue("product_name")
	ProductPrice := r.FormValue("price")
	ProductDescription := r.FormValue("description")

	if ProductName == "" || ProductPrice == "" || ProductDescription == "" {
		responseMessage = append(responseMessage, "All field are required")
		sendProductMessage(w, responseMessage, nil)
		return
	}

	price, err := strconv.ParseFloat(ProductPrice, 64)
	if err != nil {
		responseMessage = append(responseMessage, "Invalid Price")
		sendProductMessage(w, responseMessage, nil)
		return
	}

	product := models.Product{
		ProductID: productID,
		ProductName: ProductName,
		Price: price,
		Description: ProductDescription,
	}

	err = h.Repo.Product.UpdateProduct(&product)
	if err != nil {
		responseMessage = append(responseMessage, "Invalid price" + err.Error())
		sendProductMessage(w, responseMessage, nil)
		return 
	}

	//Get and send updated product
	updatedProduct, _ := h.Repo.Product.GetProductByID(productID)

	// Fake latency 
	time.Sleep(2 * time.Second)

	sendProductMessage(w, []string{}, updatedProduct)
}

func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID, err := uuid.Parse(vars["id"])
	if err != nil{
		http.Error(w, "Invalid Product id", http.StatusBadRequest)
		return
	}
	product, _ := h.Repo.Product.GetProductByID(productID)

	err = h.Repo.Product.DeleteProduct(productID)
	if err != nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return 
	}

	//Remove product image 
	productImagePath := filepath.Join("/static/upload", product.ProductImage)
	os.Remove(productImagePath)

	// Fake latency 
	time.Sleep(2 * time.Second)

	tmpl.ExecuteTemplate(w, "allProducts", nil)
}