# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a weighbridge (electronic scale) reader service written in Go that reads weight data from serial port connected scales and forwards it via WebSocket connections. The application also includes PDF printing capabilities for receipt/ticket printing.

## Core Architecture

The application follows a modular architecture with these main components:

- **main.go**: Entry point that initializes logging, config, serial manager, WebSocket hub, and HTTP routes
- **internal/config**: Configuration management with JSON-based config loading
- **internal/serial**: Serial port communication manager with automatic reconnection
- **internal/ws**: WebSocket hub for broadcasting scale data to connected clients  
- **internal/print**: PDF printing system with queue-based processing and history saving
- **mock-reader/**: Standalone mock service for testing WebSocket connections

## Development Commands

### Dependencies
```bash
go mod tidy
```

### Running the application
```bash
# Main weighbridge reader service
go run main.go

# Mock reader for testing (from mock-reader directory)
cd mock-reader && go run main.go
```

### Building
```bash
# Standard build
go build

# Windows build with hidden console (production)
go build -ldflags -H=windowsgui
```

### Testing
```bash
# Run concurrent print tests
go run test/concurrent_print_test.go
```

## Configuration

The application uses `config.json` for configuration with these key settings:
- `serial_port`: COM port for scale connection (e.g., "COM1")
- `baud_rate`: Serial communication baud rate (typically 9600)
- `websocket_port`: WebSocket server port (default 9900)
- `printer_name`: Default printer name for PDF printing

## Key Implementation Details

### Serial Communication
- Uses `go.bug.st/serial` library for cross-platform serial port access
- Implements automatic reconnection logic with 5-second retry intervals
- Filters for messages starting with "ST,GS" (scale data format)
- Broadcasts latest weight data every 500ms to WebSocket clients

### WebSocket Broadcasting
- Supports multiple concurrent client connections
- Uses gorilla/websocket with origin checking disabled for development
- Implements proper connection cleanup and resource management
- Buffered channels prevent blocking on slow clients

### PDF Printing System
- Queue-based printing to handle concurrent requests
- Uses external `PDFtoPrinterWin7.exe` tool located in `assets/` directory
- Automatic history saving to `history/` directory with duplicate name handling
- Temporary file management for print jobs

### Logging
- Uses lumberjack for log rotation (20MB max size, 7 backups, 30-day retention)
- Daily log files created in `logs/` directory
- Structured logging with timestamps and source file info

## API Endpoints

- `GET /ws`: WebSocket endpoint for receiving scale data
- `POST /print`: PDF upload and printing endpoint (multipart/form-data)

## Windows Service Installation

```powershell
# Install as Windows service
sc.exe create "WeighbridgeReaderSVC" binPath= "C:\Program Files\WeighbridgeReader\main.exe" start= auto

# Uninstall service  
sc.exe delete "WeighbridgeReaderSVC"
```

## Testing Strategy

The project includes concurrent printing tests in `test/concurrent_print_test.go` that simulate multiple simultaneous print jobs to verify queue handling and prevent resource conflicts.

## Dependencies

Key external dependencies:
- `github.com/gorilla/mux`: HTTP routing
- `github.com/gorilla/websocket`: WebSocket implementation
- `go.bug.st/serial`: Serial port communication
- `github.com/natefinch/lumberjack`: Log rotation

## Mock Reader

The `mock-reader/` directory contains a standalone WebSocket server for testing client connections without requiring actual hardware. It can be configured to send different message patterns and intervals.