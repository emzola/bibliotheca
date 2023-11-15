# BIBLIOTHECA API

The **Bibliotheca API** is a book upload and download service that allows users to upload, manage, and download their favourite books. Whether you're a reader looking to access an online library of books from anywhere or a book lover aiming to easily access all kinds of books, this API provides a seamless and efficient solution. Uploaded books are stored on Amazon S3 object storage.

## Table of Contents

- [Features](#features)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
- [Usage](#usage)
  - [Uploading a Book](#uploading-a-book)
  - [Downloading a Book](#downloading-a-book)
  - [Managing Books](#managing-books)
- [API Documentation](#api-documentation)
- [License](#license)

## <a id="features"></a>Features

- **Secure Authentication:** Utilizes token-based authentication for secure access to the API.
- **Upload Books:** Users can easily upload their books in various formats (PDF, ePub, etc.).
- **Download Books:** Users can download their uploaded books from any device.
- **Book Management:** CRUD operations to manage book metadata (title, author, genre, etc.).
- **Review and Rating:** Review and rating feature for books.
- **Booklists:** Users can easily create booklists according to interests.
- **Book Requests:** Users can request for books not yet on the database. Such books can be added by any other user who has it.
- **Search and Filters:** Search for books, booklists and requests based on different criteria such as title, author, and genre.
- **User-Friendly:** Intuitive API endpoints for smooth integration into different applications.

## <a id="getting-started"></a>Getting Started

### <a id="prerequisites"></a>Prerequisites

Before you begin, ensure you have the following:

- Go 1.19 or higher installed
- PostgreSQl up and running

### <a id="installation"></a>Installation

1. Clone this repository:

   ```bash
   git clone https://github.com/emzola/bibliotheca.git

2. Create database and configure environment variables:

   Required variables can be found in main.go.

3. Run database migration:

   ```bash
   make -f Makefile db/migrations/up

4. Start the server:

   ```bash
   make -f Makefile run/api

## <a id="usage"></a>Usage

### <a id="uploading-a-book"></a>Uploading a Book

**Endpoint:** `POST /v1/books`

Upload a book file (PDF, ePub, etc.) along with its metadata.

### <a id="downloading-a-book"></a>Downloading a Book

**Endpoint:** `GET /v1/books/:id/download`

Download a book by providing its unique identifier (`:id`).

### <a id="managing-books"></a>Managing Books

- **Get All Books:** `GET /v1/books`
- **Get Book by ID:** `GET /v1/books/:id`
- **Update Book:** `PUT /v1/books/:id`
- **Delete Book:** `DELETE /v1/books/:id`

## <a id="api-documentation"></a>API Documentation

For detailed API documentation and request/response examples, refer to [API Documentation](https://bibliotheca-api-dev-xfnt.4.us-1.fl0.io/api-docs).


## <a id="license"></a>License

This project is licensed under the [MIT License](LICENSE).



