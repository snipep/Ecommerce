package models

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	OrderID 		uuid.UUID
	User_id 		string
	Order_Status 	string
	Order_Date 		time.Time
	Items 			[]OrderItem
}
