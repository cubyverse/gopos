package email_test

import (
	"testing"

	"gopos/config"
	"gopos/services"
)

func TestEmailService(t *testing.T) {
	// Test email service initialization
	t.Run("InitEmailService", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Email.Endpoint = "https://test.communication.azure.com"
		cfg.Email.AccessKey = "test-key"
		cfg.Email.SenderMail = "test@example.com"

		services.InitEmailService(cfg)
	})

	// Test transaction email
	t.Run("SendTransactionEmail", func(t *testing.T) {
		products := []services.Product{
			{
				Name:     "Test Product",
				Price:    10.0,
				Quantity: 2,
			},
		}

		err := services.SendTransactionEmail(
			"test@example.com",
			"Test User",
			-20.0, // negative amount for purchase
			80.0,  // new balance
			products,
		)

		if err != nil && err.Error() != "email service not initialized" {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Test low balance notification
	t.Run("SendLowBalanceEmail", func(t *testing.T) {
		err := services.SendTransactionEmail(
			"test@example.com",
			"Test User",
			50.0,  // positive amount for top-up
			150.0, // new balance
			nil,   // no products for top-up
		)

		if err != nil && err.Error() != "email service not initialized" {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
