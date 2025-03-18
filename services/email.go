package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"gopos/config"

	"github.com/karim-w/go-azure-communication-services/emails"
)

var emailConfig *config.Config

func InitEmailService(cfg *config.Config) {
	emailConfig = cfg
	log.Printf("[EMAIL] Service initialized with endpoint: %s", cfg.Email.Endpoint)
}

type Config struct {
	Email struct {
		Endpoint   string `yaml:"endpoint"`
		AccessKey  string `yaml:"access_key"`
		SenderMail string `yaml:"sender_mail"`
	} `yaml:"email"`
}

type Product struct {
	Name     string
	Price    float64
	Quantity int
}

func SendTransactionEmail(toEmail, userName string, amount float64, newBalance float64, products []Product) error {
	if emailConfig == nil {
		return fmt.Errorf("email service not initialized")
	}

	if emailConfig.Email.Endpoint == "" || emailConfig.Email.AccessKey == "" || emailConfig.Email.SenderMail == "" {
		return fmt.Errorf("email configuration incomplete: endpoint=%s, access_key=%s, sender_mail=%s",
			emailConfig.Email.Endpoint,
			emailConfig.Email.AccessKey,
			emailConfig.Email.SenderMail)
	}

	endpoint := emailConfig.Email.Endpoint

	log.Printf("[EMAIL] Attempting to send email to: %s", toEmail)
	log.Printf("[EMAIL] Using sender: %s", emailConfig.Email.SenderMail)
	log.Printf("[EMAIL] Using endpoint: %s", endpoint)

	client := emails.NewClient(endpoint, emailConfig.Email.AccessKey, nil)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build product list for plain text
	var productListText strings.Builder
	productListText.WriteString("\nAbgerechnete Produkte:\n")
	for _, p := range products {
		productListText.WriteString(fmt.Sprintf("- %dx %s (%.2f €)\n", p.Quantity, p.Name, p.Price))
	}

	subject := fmt.Sprintf("Transaktion: %.2f€", amount)
	messageText := fmt.Sprintf(`Hallo %s,

Ihre Transaktion wurde erfolgreich durchgeführt:

Betrag: %.2f €
Neuer Kontostand: %.2f €
Zeitpunkt: %s
%s`, userName, amount, newBalance, time.Now().Format("02.01.2006 15:04:05"), productListText.String())

	// Build product list for HTML
	var productListHTML strings.Builder
	for _, p := range products {
		productListHTML.WriteString(fmt.Sprintf(`
            <tr>
                <td style="padding: 8px">%s</td>
                <td style="padding: 8px; text-align: center">%d</td>
                <td style="padding: 8px; text-align: right">%.2f €</td>
                <td style="padding: 8px; text-align: right">%.2f €</td>
            </tr>`, p.Name, p.Quantity, p.Price, float64(p.Quantity)*p.Price))
	}

	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; }
        .header { background-color: #1a73e8; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { padding: 20px; background-color: #fff; border: 1px solid #ddd; border-radius: 0 0 5px 5px; }
        .transaction-details { background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 15px 0; }
        .amount { font-size: 24px; color: #1a73e8; font-weight: bold; }
        .balance { color: #28a745; font-weight: bold; }
        .timestamp { color: #666; font-size: 14px; }
        .footer { margin-top: 20px; padding-top: 20px; border-top: 1px solid #ddd; color: #666; font-size: 14px; }
        .products-table { width: 100%%; border-collapse: collapse; margin: 15px 0; }
        .products-table th { background-color: #f8f9fa; text-align: left; padding: 8px; border-bottom: 2px solid #ddd; }
        .products-table tr:nth-child(even) { background-color: #f8f9fa; }
        .products-table td { border-bottom: 1px solid #ddd; }
    </style>
</head>
<body>
    <div class="header">
        <h2>Transaktion Bestätigt</h2>
    </div>
    <div class="content">
        <p>Hallo %s,</p>
        <p>Ihre Transaktion wurde erfolgreich durchgeführt:</p>
        
        <div class="transaction-details">
            <p>Betrag: <span class="amount">%.2f €</span></p>
            <p>Neuer Kontostand: <span class="balance">%.2f €</span></p>
            <p class="timestamp">Zeitpunkt: %s</p>
        </div>

        <h3>Abgerechnete Produkte</h3>
        <table class="products-table">
            <thead>
                <tr>
                    <th>Produkt</th>
                    <th style="text-align: center">Menge</th>
                    <th style="text-align: right">Preis</th>
                    <th style="text-align: right">Gesamt</th>
                </tr>
            </thead>
            <tbody>
                %s
            </tbody>
        </table>
    </div>
</body>
</html>`, userName, amount, newBalance, time.Now().Format("02.01.2006 15:04:05"), productListHTML.String())

	payload := emails.Payload{
		Headers: emails.Headers{
			ClientCorrelationID:    fmt.Sprintf("tx-%d", time.Now().Unix()),
			ClientCustomHeaderName: "gopos-Transaction",
		},
		SenderAddress: emailConfig.Email.SenderMail,
		Content: emails.Content{
			Subject:   subject,
			PlainText: messageText,
			HTML:      htmlContent,
		},
		Recipients: emails.Recipients{
			To: []emails.ReplyTo{
				{
					Address:     toEmail,
					DisplayName: userName,
				},
			},
		},
	}

	log.Printf("[EMAIL] Sending email with subject: %s", subject)
	result, err := client.SendEmail(ctx, payload)
	if err != nil {
		log.Printf("[EMAIL] Error sending email: %+v", err)
		log.Printf("[EMAIL] API Response: %+v", result)
		return fmt.Errorf("failed to send email: %+v", err)
	}
	log.Printf("[EMAIL] Email sent successfully with response: %+v", result)
	return nil
}
