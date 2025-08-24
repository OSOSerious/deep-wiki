package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
	"github.com/stripe/stripe-go/customer"
	"github.com/stripe/stripe-go/token"
)

// Models
type Order struct {
	ID        int    `json:"id"`
	 UserID    int    `json:"user_id"`
	 ProductID int    `json:"product_id"`
	 Quantity  int    `json:"quantity"`
	 Total     float64 `json:"total"`
}

type PaymentMethod struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type PaymentIntent struct {
	ID          string `json:"id"`
	Amount      int    `json:"amount"`
	Currency    string `json:"currency"`
	PaymentMethod string `json:"payment_method"`
	Status      string `json:"status"`
}

// Interfaces
type PaymentGateway interface {
	ChargeCard(string, int, string) (*stripe.Charge, error)
	CreateCustomer(string, string) (*stripe.Customer, error)
}

type Database interface {
	GetOrder(int) (*Order, error)
	UpdateOrder(int, float64) error
}

// Stripe Payment Gateway Implementation
type StripePaymentGateway struct{}

func (s *StripePaymentGateway) ChargeCard(token string, amount int, currency string) (*stripe.Charge, error) {
	params := &stripe.ChargeParams{
		Amount: stripe.Int64(int64(amount)),
		Currency: stripe.String(currency),
		Source: &stripe.SourceParams{
			Token: stripe.String(token),
		},
	}
	return charge.New(params)
}

func (s *StripePaymentGateway) CreateCustomer(email string, description string) (*stripe.Customer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Description: stripe.String(description),
	}
	return customer.New(params)
}

// Database Implementation
type MySQLDatabase struct{}

func (m *MySQLDatabase) GetOrder(id int) (*Order, error) {
	// Implement database query to retrieve order
	db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/database")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var order Order
	err = db.QueryRow("SELECT * FROM orders WHERE id = ?", id).Scan(&order.ID, &order.UserID, &order.ProductID, &order.Quantity, &order.Total)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (m *MySQLDatabase) UpdateOrder(id int, total float64) error {
	// Implement database query to update order
	db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/database")
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("UPDATE orders SET total = ? WHERE id = ?", total, id)
	return err
}

// API Handlers
func CheckoutHandler(w http.ResponseWriter, r *http.Request) {
	var order Order
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := &MySQLDatabase{}
	order, err = db.GetOrder(order.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	paymentMethod := &PaymentMethod{}
	err = json.NewDecoder(r.Body).Decode(paymentMethod)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	paymentGateway := &StripePaymentGateway{}
	customer, err := paymentGateway.CreateCustomer("customer@example.com", "Customer Name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := token.New(&stripe.TokenParams{
		Customer: stripe.String(customer.ID),
		Card: &stripe.CardParams{
			Number: stripe.String(paymentMethod.ID),
			ExpMonth: stripe.Int64(12),
			ExpYear: stripe.Int64(2025),
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	charge, err := paymentGateway.ChargeCard(token.ID, int(order.Total*100), "usd")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	paymentIntent := &PaymentIntent{
		ID:          charge.ID,
		Amount:      int(order.Total * 100),
		Currency:    "usd",
		PaymentMethod: paymentMethod.Type,
		Status:      "succeeded",
	}

	err = json.NewEncoder(w).Encode(paymentIntent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/checkout", CheckoutHandler)
	http.ListenAndServe(":8080", nil)
}