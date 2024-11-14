# OwlDB - Network Accessible NoSQL Document Database

OwlDB is a RESTful web service implemented in Go, providing a NoSQL document database that supports the creation, modification, retrieval, and deletion of JSON documents. This project is developed as part of a course requirement, focusing on concurrency, atomic operations, authentication, and document subscription.

## Features
- **HTTP Methods**: Supports standard CRUD operations through `GET`, `PUT`, `POST`, `PATCH`, and `DELETE`.
- **Document Structure & Validation**: Validates documents against a JSON schema using `github.com/santhosh-tekuri/jsonschema/v5`.
- **Hierarchical Data Organization**: Organizes documents in nested databases and collections with hierarchical paths.
- **Authentication**: Minimalist token-based authentication with expiring tokens.
- **Subscriptions**: Real-time updates via Server-Sent Events for subscribed documents or collections.
- **Atomic Operations**: Supports conditional writes and patching for single documents.
- **Concurrent Skip List**: Efficient indexing using a custom, thread-safe skip list implementation.

## Usage
Run OwlDB with the following command-line options:
```bash
./owldb -p <port> -s <schema-file> -t <token-file>
```
- `-p <port>`: Port number (default is 3318).
- `-s <schema-file>`: Path to JSON schema for validating documents.
- `-t <token-file>`: Path to token JSON file for user authentication.

## Project Structure
- **API Endpoints**: Implements API routes for database, document, collection management, and subscriptions.
- **Concurrency**: Uses goroutines and channels for handling multiple clients, with atomic operations for critical sections.
- **Testing**: Comprehensive unit tests in each package, executable via `go test ./...`.
- **Logging**: Structured logging with slog for debug, info, and error messages.

## Development & Testing
Clone the repository and use the Go tools to build and test:
```bash
go build -o owldb
go test ./...
```

## Documentation
API documentation is accessible via Swagger when the server is running:
```
http://localhost:8318/swagger/
```

For detailed design information, refer to `design.pdf` in the repository.

---


