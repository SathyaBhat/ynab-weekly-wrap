# YNAB Weekly Wrap

A Go-based automation tool that fetches your YNAB budget data weekly and sends a formatted financial wrap report to a Telegram private group.

## Features

- Weekly budget analysis and reporting
- Category spending breakdown and insights
- Overspend detection and alerts
- Automated Telegram notifications, including support for publishing to a topic in a supergroup
- Cron-based scheduling (configurable)
- Dry-run mode for testing (prints to stdout instead of Telegram)

## Requirements

- Go 1.21+ (for local development)
- Docker & Docker Compose (for containerized deployment)
- YNAB Account with API access
- Telegram Bot Token
- Private Telegram Group

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/sathyabhat/ynab-weekly-wrap.git
cd ynab-weekly-wrap
```

### 2. Setup Configuration

```bash
cp .env.example .env
# Edit .env with your credentials
```

Required environment variables:
- `YNAB_API_TOKEN` - Your YNAB API token
- `YNAB_BUDGET_ID` - Your YNAB budget ID
- `TELEGRAM_BOT_TOKEN` - Your Telegram bot token
- `TELEGRAM_CHAT_ID` - Target Telegram chat ID

Optional environment variables:
- `SCHEDULE_CRON` - Cron expression for scheduling (default: `0 9 * * 1`)
- `LOG_LEVEL` - Log level: debug, info, warn, error (default: `info`)
- `TELEGRAM_TOPIC_ID` - Telegram topic ID (optional - if you wish to publish to a topic)

### 3. Local Development

#### Build

```bash
make build
```

#### Run

```bash
make run
```

#### Test

```bash
make test
```

#### Dry-Run (Test without sending to Telegram)

```bash
./bin/ynab-weekly-wrap -dry-run
```

This will fetch your YNAB data and print the formatted message to stdout without sending to Telegram. Perfect for testing configuration and verifying output.

### 4. Docker Deployment

#### Using Docker Compose (Recommended)

```bash
docker compose up -d
```

View logs:
```bash
docker compose logs -f
```

Stop services:
```bash
docker compose down
```

#### Manual Docker Build

```bash
make docker-build
make docker-run
```

## Setup Instructions

### Getting Your YNAB API Token

1. Go to https://app.ynab.com/settings/developer
2. Click "New Token"
3. Copy the generated token to `.env` or `config.yaml`

### Getting Your Budget ID

1. Log into YNAB and open your budget
2. The URL will be: `https://app.ynab.com/budgets/{BUDGET_ID}`
3. Copy the `{BUDGET_ID}` portion

### Creating a Telegram Bot

1. Message @BotFather on Telegram
2. Use `/newbot` to create a new bot
3. Copy the bot token to `.env` or `config.yaml`

### Getting Your Telegram Chat ID

1. Create a private group in Telegram
2. Add your bot to the group as an admin
3. Send a message to the group
4. Get chat ID using: `curl https://api.telegram.org/bot{BOT_TOKEN}/getUpdates`
5. Look for the `"chat":{"id":...}` value
6. Use this ID (usually negative) in `.env` or `config.yaml`

## Configuration

### Schedule Configuration

The `SCHEDULE_CRON` environment variable uses standard cron syntax. See [crontab.guru](https://crontab.guru/) for more details.

### Message Format

The bot sends messages in this format:


📊 **Weekly Financial Wrap - 2026-01-07 to 2026-01-14**

💰 **Total Spent**: $7518.83

🏆 **Top Spending Categories**    
- **🧘 Fitness**: Activity: $200  Remaining: $20    
- **🥡 Dining Out**: Activity: $250.65  Remaining: $0    
- **🍌 Groceries**: Activity: $123.63  Remaining: $200.04     

⚠️ **Over Budget Categories**        
- **🙂 Entertainment**: Activity: $100 Remaining: - $100    

## Development

### Project Structure

```
.
├── cmd/
│   └── app/
│       └── main.go           # Entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration management
│   ├── ynab/
│   │   ├── client.go         # YNAB API client
│   │   └── models.go         # Data models
│   ├── telegram/
│   │   └── bot.go            # Telegram bot client
│   ├── processor/
│   │   ├── analyzer.go       # Data analysis engine
│   │   └── models.go         # Analysis result models
│   └── scheduler/
│       └── cron.go           # Cron scheduler
├── Dockerfile                # Docker image definition
├── docker-compose.yml        # Docker Compose configuration
├── Makefile                  # Build automation
└── go.mod                    # Go module definition
```

### Command-Line Flags

The application supports the following flags:

```bash
./bin/ynab-weekly-wrap -dry-run    # Test mode: print message to stdout instead of sending to Telegram
./bin/ynab-weekly-wrap -once       # Run once and exit (useful for manual testing)
./bin/ynab-weekly-wrap -help       # Show available flags
```

Examples:
```bash
# Test configuration without sending to Telegram
./bin/ynab-weekly-wrap -dry-run

# Run a single report and send to Telegram
./bin/ynab-weekly-wrap -once

# Run with Docker
docker run --rm --env-file .env ynab-weekly-wrap -dry-run
```

See [DRY_RUN.md](DRY_RUN.md) for detailed dry-run usage and troubleshooting.

### Available Make Commands

```bash
make help              # Display all available commands
make build             # Build the application
make build-linux       # Build for Linux
make build-macos       # Build for macOS
make run              # Build and run locally
make test             # Run tests
make test-coverage    # Run tests with coverage report
make lint             # Run golangci-lint
make fmt              # Format code
make vet              # Run go vet
make clean            # Clean build artifacts
make deps             # Download dependencies
make tidy             # Tidy dependencies
make docker-build     # Build Docker image
make docker-run       # Run Docker container
make docker-compose-up    # Start with docker-compose
make docker-compose-down  # Stop services
make docker-push      # Push to registry
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage report
make test-coverage
```

### Code Quality

```bash
# Format code
make fmt

# Run linter (requires golangci-lint)
make lint

# Run go vet
make vet
```

## License

MIT License - see LICENSE file for details

## Support

For issues, questions, or suggestions:

1. Check existing issues on GitHub
2. Create a new issue with detailed information
3. Include relevant logs and configuration (without sensitive data)
