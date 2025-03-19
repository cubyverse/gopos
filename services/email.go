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

// EmailType represents different types of notifications
type EmailType string

const (
	EmailTypeTransaction EmailType = "transaction"
	EmailTypeTopup       EmailType = "topup"
	EmailTypeUserUpdated EmailType = "user_updated"
)

// sendEmail is a generic function to send emails using Azure Communication Services
func sendEmail(toEmail, userName, subject, plainText, htmlContent string) error {
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

	payload := emails.Payload{
		Headers: emails.Headers{
			ClientCorrelationID:    fmt.Sprintf("email-%d", time.Now().Unix()),
			ClientCustomHeaderName: "gopos-Email",
		},
		SenderAddress: emailConfig.Email.SenderMail,
		Content: emails.Content{
			Subject:   subject,
			PlainText: plainText,
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

// SendTransactionEmail sends an email notification for a transaction
func SendTransactionEmail(toEmail, userName string, amount float64, newBalance float64, products []Product) error {
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
        .greeting { margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="header">
        <h2>Transaktion Bestätigt</h2>
    </div>
    <div class="content">
        <p class="greeting">Hallo %s,</p>
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

	return sendEmail(toEmail, userName, subject, messageText, htmlContent)
}

// SendTopupEmail sends an email notification for a balance top-up
func SendTopupEmail(toEmail, userName string, amount float64, newBalance float64) error {
	subject := fmt.Sprintf("Guthaben aufgeladen: %.2f€", amount)
	messageText := fmt.Sprintf(`Hallo %s,

Ihr Guthaben wurde erfolgreich aufgeladen:

Aufgeladener Betrag: %.2f €
Neuer Kontostand: %.2f €
Zeitpunkt: %s`, userName, amount, newBalance, time.Now().Format("02.01.2006 15:04:05"))

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
        .amount { font-size: 24px; color: #28a745; font-weight: bold; }
        .balance { color: #28a745; font-weight: bold; }
        .timestamp { color: #666; font-size: 14px; }
        .footer { margin-top: 20px; padding-top: 20px; border-top: 1px solid #ddd; color: #666; font-size: 14px; }
        .greeting { margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="header">
        <h2>Guthaben Aufgeladen</h2>
    </div>
    <div class="content">
        <p class="greeting">Hallo %s,</p>
        <p>Ihr Guthaben wurde erfolgreich aufgeladen:</p>
        
        <div class="transaction-details">
            <p>Aufgeladener Betrag: <span class="amount">%.2f €</span></p>
            <p>Neuer Kontostand: <span class="balance">%.2f €</span></p>
            <p class="timestamp">Zeitpunkt: %s</p>
        </div>
    </div>
</body>
</html>`, userName, amount, newBalance, time.Now().Format("02.01.2006 15:04:05"))

	return sendEmail(toEmail, userName, subject, messageText, htmlContent)
}

// SendUserUpdatedEmail sends an email notification when user information is updated
func SendUserUpdatedEmail(toEmail, userName string, isNewUser bool, changes map[string]map[string]string) error {
	var subject, messageText, htmlContent string

	if isNewUser {
		subject = "Willkommen bei GoPOS"

		// Build account details for plain text
		var detailsText strings.Builder
		detailsText.WriteString("Ihre Kontoinformationen:\n\n")
		if changes != nil {
			for field, values := range changes {
				detailsText.WriteString(fmt.Sprintf("- %s: %s\n", field, values["new"]))
			}
		}

		messageText = fmt.Sprintf(`Hallo %s,

Ihr Benutzerkonto bei GoPOS wurde erfolgreich erstellt. Sie können jetzt alle Funktionen des Systems nutzen.

%s
Zeitpunkt: %s`, userName, detailsText.String(), time.Now().Format("02.01.2006 15:04:05"))

		// Build account details for HTML
		var detailsHTML strings.Builder
		if changes != nil && len(changes) > 0 {
			for field, values := range changes {
				detailsHTML.WriteString(fmt.Sprintf(`                    <tr>
                        <td style="padding: 8px; font-weight: bold;">%s</td>
                        <td style="padding: 8px;">%s</td>
                    </tr>`, field, values["new"]))
			}
		}

		htmlContent = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; }
        .header { background-color: #1a73e8; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { padding: 20px; background-color: #fff; border: 1px solid #ddd; border-radius: 0 0 5px 5px; }
        .welcome { font-size: 20px; color: #1a73e8; font-weight: bold; }
        .timestamp { color: #666; font-size: 14px; }
        .footer { margin-top: 20px; padding-top: 20px; border-top: 1px solid #ddd; color: #666; font-size: 14px; }
        .details-table { width: 100%%; border-collapse: collapse; margin: 15px 0; }
        .details-table th { background-color: #f8f9fa; text-align: left; padding: 8px; border-bottom: 2px solid #ddd; }
        .details-table tr:nth-child(even) { background-color: #f8f9fa; }
        .details-table td { border-bottom: 1px solid #ddd; }
        .greeting { margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="header">
        <h2>Willkommen bei GoPOS</h2>
    </div>
    <div class="content">
        <p class="greeting">Hallo %s,</p>
        <p class="welcome">Willkommen bei GoPOS!</p>
        <p>Ihr Benutzerkonto wurde erfolgreich erstellt. Sie können jetzt alle Funktionen des Systems nutzen.</p>
        
        <div class="update-info">
            <h3>Ihre Kontoinformationen</h3>
            <table class="details-table">
                <thead>
                    <tr>
                        <th>Feld</th>
                        <th>Wert</th>
                    </tr>
                </thead>
                <tbody>
                    %s
                </tbody>
            </table>
            <p class="timestamp">Zeitpunkt: %s</p>
        </div>
    </div>
</body>
</html>`, userName, detailsHTML.String(), time.Now().Format("02.01.2006 15:04:05"))
	} else {
		subject = "Ihre Kontoinformationen wurden aktualisiert"

		// Build changed fields for plain text
		var changesText strings.Builder
		changesText.WriteString("Folgende Änderungen wurden vorgenommen:\n\n")
		if len(changes) > 0 {
			for field, values := range changes {
				changesText.WriteString(fmt.Sprintf("- %s: '%s' → '%s'\n", field, values["old"], values["new"]))
			}
		} else {
			changesText.WriteString("- Allgemeine Kontoaktualisierung\n")
		}

		messageText = fmt.Sprintf(`Hallo %s,

Ihre Kontoinformationen bei GoPOS wurden aktualisiert.

%s
Zeitpunkt: %s

Wenn Sie diese Änderung nicht vorgenommen haben, kontaktieren Sie bitte den Administrator.`, userName, changesText.String(), time.Now().Format("02.01.2006 15:04:05"))

		// Build changed fields for HTML
		var changesHTML strings.Builder
		if len(changes) > 0 {
			for field, values := range changes {
				changesHTML.WriteString(fmt.Sprintf(`                    <tr>
                        <td style="padding: 8px; font-weight: bold;">%s</td>
                        <td style="padding: 8px;"><span class="old-value">%s</span></td>
                        <td style="padding: 8px;"><span class="new-value">%s</span></td>
                    </tr>`, field, values["old"], values["new"]))
			}
		} else {
			changesHTML.WriteString(`                    <tr>
                        <td colspan="3" style="padding: 8px; text-align: center;">Allgemeine Kontoaktualisierung</td>
                    </tr>`)
		}

		htmlContent = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; }
        .header { background-color: #1a73e8; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { padding: 20px; background-color: #fff; border: 1px solid #ddd; border-radius: 0 0 5px 5px; }
        .update-info { background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 15px 0; }
        .note { color: #d73a49; }
        .timestamp { color: #666; font-size: 14px; }
        .footer { margin-top: 20px; padding-top: 20px; border-top: 1px solid #ddd; color: #666; font-size: 14px; }
        .changes-table { width: 100%%; border-collapse: collapse; margin: 15px 0; }
        .changes-table th { background-color: #f8f9fa; text-align: left; padding: 8px; border-bottom: 2px solid #ddd; }
        .changes-table tr:nth-child(even) { background-color: #f8f9fa; }
        .changes-table td { border-bottom: 1px solid #ddd; }
        .old-value { color: #d73a49; text-decoration: line-through; }
        .new-value { color: #28a745; font-weight: bold; }
        .arrow { color: #666; font-size: 18px; }
        .greeting { margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="header">
        <h2>Kontoinformationen Aktualisiert</h2>
    </div>
    <div class="content">
        <p class="greeting">Hallo %s,</p>
        <p>Ihre Kontoinformationen bei GoPOS wurden aktualisiert.</p>
        
        <div class="update-info">
            <h3>Änderungen</h3>
            <table class="changes-table">
                <thead>
                    <tr>
                        <th>Feld</th>
                        <th>Alter Wert</th>
                        <th>Neuer Wert</th>
                    </tr>
                </thead>
                <tbody>
                    %s
                </tbody>
            </table>
            <p class="timestamp">Zeitpunkt: %s</p>
        </div>
        
        <p class="note">Wenn Sie diese Änderung nicht vorgenommen haben, kontaktieren Sie bitte den Administrator.</p>
    </div>
</body>
</html>`, userName, changesHTML.String(), time.Now().Format("02.01.2006 15:04:05"))
	}

	return sendEmail(toEmail, userName, subject, messageText, htmlContent)
}
