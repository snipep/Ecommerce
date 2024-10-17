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

var (
	tmpl *template.Template
	currentCartOrderId uuid.UUID
	cartItems []models.OrderItem
)

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

func (h *Handler) ShoppingHomepage(w http.ResponseWriter, r *http.Request) {
	data := struct{
		OrderItems []models.OrderItem
	}{
		OrderItems: cartItems,
	}

	tmpl.ExecuteTemplate(w, "homepage", data)
}

func (h *Handler) ShoppingItemView(w http.ResponseWriter, r *http.Request) {
	// fake latency 
	time.Sleep(2 * time.Second)

	products, _ := h.Repo.Product.GetProducts("product_image !=''")

	tmpl.ExecuteTemplate(w, "shoppingItems", products)
}

func (h *Handler) CartView(w http.ResponseWriter, r *http.Request) {
	data := struct{
		OrderItems []models.OrderItem
		Message string
		AlertType string
		TotalCost float64
	}{
		OrderItems: cartItems,
		Message: "",
		AlertType: "",
		TotalCost: getTotalCartCost(),
	}

	tmpl.ExecuteTemplate(w, "cartItems", data)
}

func getTotalCartCost() float64 {
	totaCost := 0.0
	for _, item := range cartItems {
		totaCost += float64(item.Quantity) * item.Product.Price
	}

	return totaCost
}

func (h *Handler) AddToCart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID, err := uuid.Parse(vars["product_id"])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		fmt.Println("failedd bruhh")
		return 
	}

	//Generate a new order id for the session if one does not exist
	if currentCartOrderId == uuid.Nil {
		currentCartOrderId = uuid.New()
	}

	//Check if profuct already exist in order items
	exist := false
	for _, item := range cartItems{
		if item.ProductID == productID{
			exist = true
			break
		}
	}

	// Get the Product 
	product, _ := h.Repo.Product.GetProductByID(productID)

	cartMessage := ""
	alertType := ""

	if !exist {
		//Create a new order item
		newOrderItem := models.OrderItem{
			OrderID: currentCartOrderId,
			ProductID: productID,
			Quantity: 1,
			Product: *product,
		}	 

		// Add new order item to the array 
		cartItems = append(cartItems, newOrderItem)

		cartMessage = product.ProductName + " successfully added"
		alertType = "Success"
	}else {
		cartMessage = product.ProductName + " already in the cart "
		alertType = "danger"
	}

	data := struct {
		OrderItems []models.OrderItem
		Message string
		AlertType string
		TotalCost float64
	}{
		OrderItems: cartItems,
		Message: cartMessage,
		AlertType: alertType,
		TotalCost: getTotalCartCost(),
	}

	tmpl.ExecuteTemplate(w,  "cartItems", data)
}

func (h *Handler) ShoppingCartView(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "shoppingCart", cartItems)
}

func (h *Handler) UpdateorderItemQuantity(w http.ResponseWriter, r *http.Request) {
	//Get profuct ID and sction from URL parameters 
	cartMessage := ""
	refreshCartList := false //Signals a refresh of cart items when an item is removed

	productID, err := uuid.Parse(r.URL.Query().Get("product_id"))
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return 
	}
	action := r.URL.Query().Get("action")

	// Find the order item 
	var itemIndex int
	for i, item := range cartItems {
		if item.ProductID == productID {
			itemIndex = i
			break
		}
	}
	if itemIndex == -1 {
		http.Error(w, "Product not found in order", http.StatusNotFound)
		return 
	}

	//Update quantitiy based on action
	switch action {
	case "add":
		cartItems[itemIndex].Quantity++
	case "subtract":
		cartItems[itemIndex].Quantity--
		//Remove item if quantity gets to 0
		if cartItems[itemIndex].Quantity == 0 {
			cartItems = append(cartItems[:itemIndex], cartItems[itemIndex + 1:]...)
			refreshCartList = true
		}
	case "remove":
		//Remove item regarless of the quantity
		cartItems = append(cartItems[ : itemIndex], cartItems[itemIndex + 1 : ]...)
		refreshCartList = true
	default:
		/* http.Error(w, "Invaliud action", http.StatusBadRequest)
		return*/
		cartMessage = "Invalid Action"
	}

	//Respond to teh request
	//fmt.Fprintf(w, "Order item updated")
	data := struct {
		OrderItems 		[]models.OrderItem
		Message			string
		AlertType 		string
		TotalCost 		float64
		Action 			string
		RefreshCartItems bool
	}{
		OrderItems: cartItems,
		Message: cartMessage,
		AlertType: "info",
		TotalCost: getTotalCartCost(),
		Action: action,
		RefreshCartItems: refreshCartList,
	}

	tmpl.ExecuteTemplate(w, "updateShoppingCart", data)
}

func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	for i := range cartItems{
		cartItems[i].Cost = float64(cartItems[i].Quantity) * cartItems[i].Product.Price
	}

	err := h.Repo.Order.PlaceOrderWithItems(cartItems)
	if err != nil{
		http.Error(w, "Error Placing Order " + err.Error(), http.StatusBadRequest)
		return 
	}

	displayItems := cartItems
	totalCost := getTotalCartCost()

	//Empty the cart items
	cartItems = []models.OrderItem{}
	currentCartOrderId = uuid.Nil

	data := struct {
		OrderItems []models.OrderItem
		TotalCost float64
	}{
		OrderItems: displayItems,
		TotalCost: totalCost,
	}

	tmpl.ExecuteTemplate(w, "orderComplete", data)
}

func (h *Handler) OrdersPage(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "orders", nil)
}

func (h *Handler) AllordersView(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "allOrders", nil)
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1{
		page = 1
	}
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10 //default
	}
	
	offset := (page - 1) * limit

	orders, err := h.Repo.Order.ListOrders(limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return 
	}
	totalOrders, err := h.Repo.Order.GetToatlOrdersCount()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return 
	}


	totalPages := int(math.Ceil(float64(totalOrders) / float64(limit)))
	previousPage := page - 1
	nextPage := page + 1
	pageButtonsRange := makeRange(1, totalPages)

	data := struct {
		Orders           []models.Order
		CurrentPage      int
		TotalPages       int
		Limit            int
		PreviousPage     int
		NextPage         int
		PageButtonsRange []int
	}{
		Orders:           orders,
		CurrentPage:      page,
		TotalPages:       totalPages,
		Limit:            limit,
		PreviousPage:     previousPage,
		NextPage:         nextPage,
		PageButtonsRange: pageButtonsRange,
	}

	// Fake latency
	// time.Sleep(2 * time.Second) 

	tmpl.ExecuteTemplate(w, "orderRows", data)
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	order, err := h.Repo.Order.GetOrderWithProducts(orderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return 
	}
	
	totalCost := 0.0 
	for _, item := range order.Items {
		totalCost += float64(item.Quantity) * item.Product.Price
	}

	order.OrderStatus = strings.ToUpper(order.OrderStatus)

	data := struct {
		Order models.Order
		TotalCost float64
	}{
		Order: *order,
		TotalCost: totalCost,
	}

	tmpl.ExecuteTemplate(w, "viewOrder", data)



}