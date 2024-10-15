package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/snipep/Ecommerce-application/pkg/models"
)

type OrderRepository struct {
	DB *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{DB: db}
}

func (r *OrderRepository) PlaceOrderWithItems(orderItems []models.OrderItem) error {
	//Begin transaction
	tx, err := r.DB.Begin()
	if err != nil{
		return err
	}

	order := models.Order{
		OrderID: uuid.New(),
		User_id: "vishnu@gmailcom",
		Order_Status: "ordered",
		Order_Date: time.Now(),
		Items: orderItems,
	}

	//insert order into orders table 
	_, err = tx.Exec("INSERT INTO orders (order_id, user_id, order_status, order_date) VALUES (?, ?, ?, ?)",order.OrderID, order.User_id, order.Order_Status, order.OrderID)
	if err != nil{
		tx.Rollback()
		return err
	}

	//Insert order items into order items table
	for _, item := range order.Items{
		_, err = tx.Exec("INSERT INTO order_items (order_id, product_id, quantity, cost) VALUES (?, ?, ?, ?)", order.OrderID, item.ProductID, item.Quantity, item.Cost)
		if err != nil{
			tx.Rollback()
			return err
		}
	}

	//Commit the transaction
	if err = tx.Commit();err != nil{
		return err
	}
	return nil
}

func (r *OrderRepository) ListOrders(limit, offset int) ([]models.Order, error) {
	query := `SELECT order_id, order_status, order_date, FROM orders ORDER BY order_date DESC LIMIT	? OFFSET ?`

	rows, err := r.DB.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders [] models.Order
	for rows.Next(){
		var order models.Order
		if err := rows.Scan(
			&order.OrderID,
			&order.Order_Status, 
			&order.Order_Status,
			&order.Order_Date, 
		);err != nil{
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *OrderRepository) GetToatlOrdersCount() (int, error) {
	var count int
	err := r.DB.QueryRow("SELECT COUNT(*) FROM orders").Scan(&count)
	if err != nil{
		return 0, nil
	}
	return count, nil 
}

func (r *OrderRepository) CreateOrder(order *models.Order) error {
	query := `INSERT INTO orders (orders_id, user_id, order_status, order_date) VALUES (?, ?, ?, ?)`
	order.OrderID = uuid.New()
	order.Order_Date = time.Now()

	_, err := r.DB.Exec(
		query, 
		order.OrderID, 
		order.User_id, 
		order.Order_Status, 
		order.OrderID,
	)
	return err
}

func (r *OrderRepository) AddOrderItem(orderItem *models.OrderItem) error {
	query := `INSERT INTO order_items (order_id, product_id, quantity) VALUES (?, ?, ?)`

	_, err := r.DB.Exec(
		query, 
		orderItem.OrderID, 
		orderItem.ProductID, 
		orderItem.Quantity,
	)
	return err
}

func (r *OrderRepository) GetOrderWithProducts(orderID uuid.UUID) (*models.Order, error) {
	//First, get the order details
	orderQuery := `SELECT order_id, user_id, order_status, order_date FROM orders WHERE order_id = ?`
	var order models.Order
	err := r.DB.QueryRow(orderQuery, orderID).Scan(
		&order.OrderID, 
		&order.User_id, 
		&order.Order_Status, 
		&order.Order_Date,
	)
	if err != nil{
		return nil, err
	}

	//Then get all order item their corresponding products
	itemsQuery := `
		SELECT oi.product_id, oi.quantity, p.product_name, p.price, p.description, p.product_image, p.date_created, p.date_modified 
		FROM order_items oi 
		JOIN products p ON oi.product_id = p.product-id
		WHERE order_id = ?
	`
	rows, err := r.DB.Query(itemsQuery, orderID)		
	if err != nil{
		return nil, err
	}
	defer rows.Close()

	for rows.Next(){
		var item models.OrderItem
		err := rows.Scan(
			&item.ProductID, 
			&item.Quantity, 
			&item.Product.ProductName,
			&item.Product.Price,
			&item.Product.Description, 
			&item.Product.ProductImage,
			&item.Product.DateCreated,
			&item.Product.DateModified,
		)
		if err != nil{
			return nil, err
		}
		item.OrderID = orderID
		item.Cost = float64(item.Quantity) * item.Product.Price
		item.Product.ProductID = item.ProductID
		order.Items = append(order.Items, item)
	}
	return &order, nil
}