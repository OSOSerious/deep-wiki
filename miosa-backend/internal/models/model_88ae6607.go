package main

import (
	"encoding/json"
	"errors"
	"fmt"
)

// CheckoutSystem is the main system for handling checkout processes
type CheckoutSystem struct {
	paymentGateway PaymentGateway
}

// NewCheckoutSystem returns a new instance of CheckoutSystem
func NewCheckoutSystem(pg PaymentGateway) *CheckoutSystem {
	return &CheckoutSystem{pg}
}

// ProcessCheckout processes a checkout request
func (cs *CheckoutSystem) ProcessCheckout(req *CheckoutRequest) (*CheckoutResponse, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Create a new order
	order := NewOrder(req.Cart, req.Customer)

	// Process payment
	payment, err := cs.paymentGateway.ProcessPayment(order.Total, req.PaymentMethod)
	if err != nil {
		return nil, err
	}

	// Update the order with payment info
	order.Payment = payment

	// Return the checkout response
	return &CheckoutResponse{Order: order}, nil
}

// CheckoutRequest represents a checkout request
type CheckoutRequest struct {
	Cart        []*CartItem `json:"cart"`
	Customer    *Customer  `json:"customer"`
	PaymentMethod string    `json:"payment_method"`
}

func (cr *CheckoutRequest) Validate() error {
	if len(cr.Cart) == 0 {
		return errors.New("cart is empty")
	}
	if cr.Customer == nil {
		return errors.New("customer is required")
	}
	if cr.PaymentMethod == "" {
		return errors.New("payment method is required")
	}
	return nil
}

// CartItem represents an item in the cart
type CartItem struct {
	ProductID int    `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Price     float64 `json:"price"`
}

// Customer represents a customer
type Customer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Order represents an order
type Order struct {
	ID        int      `json:"id"`
	Cart      []*CartItem `json:"cart"`
	Customer  *Customer `json:"customer"`
	Payment   *Payment  `json:"payment"`
	Total     float64  `json:"total"`
}

func NewOrder(cart []*CartItem, customer *Customer) *Order {
	order := &Order{
		Cart:     cart,
		Customer: customer,
	}
	order.Total = calculateTotal(cart)
	return order
}

func calculateTotal(cart []*CartItem) float64 {
	total := 0.0
	for _, item := range cart {
		total += item.Price * float64(item.Quantity)
	}
	return total
}

// PaymentGateway is an interface for payment gateways
type PaymentGateway interface {
	ProcessPayment(amount float64, paymentMethod string) (*Payment, error)
}

// Payment represents a payment
type Payment struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Amount     float64 `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

// StripePaymentGateway is a payment gateway implementation for Stripe
type StripePaymentGateway struct{}

func (pg *StripePaymentGateway) ProcessPayment(amount float64, paymentMethod string) (*Payment, error) {
	// Implement Stripe payment processing logic here
	// For demo purposes, return a successful payment
	return &Payment{
		ID:         "stripe_payment_id",
		Status:     "success",
		Amount:     amount,
		PaymentMethod: paymentMethod,
	}, nil
}

// CheckoutResponse represents a checkout response
type CheckoutResponse struct {
	Order *Order `json:"order"`
}

func main() {
	// Create a new payment gateway
	pg := &StripePaymentGateway{}

	// Create a new checkout system
	cs := NewCheckoutSystem(pg)

	// Create a sample checkout request
	req := &CheckoutRequest{
		Cart: []*CartItem{
			{ProductID: 1, Quantity: 2, Price: 10.99},
			{ProductID: 2, Quantity: 1, Price: 5.99},
		},
		Customer: &Customer{
			Name:  "John Doe",
			Email: "johndoe@example.com",
		},
		PaymentMethod: "stripe",
	}

	// Process the checkout request
	resp, err := cs.ProcessCheckout(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Marshal the response to JSON
	jsonResp, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(jsonResp))
}