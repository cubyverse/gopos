# GoPOS

A modern Point of Sale system for managing customer accounts, products, and transactions.

## Features

- Customer accounts with balance management
- Product management with barcode scanning
- Transaction processing with real-time updates
- Role-based access control (Admin, Cashier, Customer)
- Automatic email notifications
- Audit logging of all system activities
- Responsive web interface

## Technology

- Backend: Go 1.21+
- Frontend: HTMX + Tailwind CSS
- Database: SQLite
- Email: Azure Communication Services
- Authentication: Session-based with CSRF protection

## Installation

1. Download the latest binary from GitHub Releases:
```bash
# Windows
curl -sLO https://github.com/cubyverse/gopos/releases/latest/download/gopos-windows-amd64.exe
mv gopos-windows-amd64.exe gopos.exe

# Linux
curl -sLO https://github.com/cubyverse/gopos/releases/latest/download/gopos-linux-amd64
chmod +x gopos-linux-amd64

# macOS
curl -sLO https://github.com/cubyverse/gopos/releases/latest/download/gopos-darwin-amd64
chmod +x gopos-darwin-amd64
```

2. Create configuration file in one of the following locations (in order of precedence):
- Current directory: `./config.yaml`
- User config directory:
  - Windows: `%APPDATA%\gopos\config.yaml`
  - Linux/macOS: `~/.config/gopos/config.yaml`
- System config directory:
  - Windows: `C:\ProgramData\gopos\config.yaml`
  - Linux/macOS: `/etc/gopos/config.yaml`

```yaml
database:
  path: "/opt/gopos/gopos.db"

email:
  endpoint: "https://your-resource-name.communication.azure.com"
  access_key: "your-access-key-from-azure"
  sender_mail: "noreply@your-verified-domain.com"

session:
  key: "your-secure-session-key"
```

3. Start the application:
```bash
# Windows
.\gopos.exe

# Linux/macOS
./gopos-linux-amd64
```

## Development

- `go run main.go` - Starts the application in development mode
- `templ generate` - Generates template code
- `./tailwindcss -i ./static/css/input.css -o ./static/css/output.css --watch` - Develop CSS with hot-reload