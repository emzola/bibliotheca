package main

// import (
// 	"errors"
// 	"net/http"

// 	"github.com/emzola/bibliotheca/internal/data"
// )

// func (app *application) addBookToBooklistHandler(w http.ResponseWriter, r *http.Request) {
// 	booklistId, err := app.readIDParam(r, "booklistId")
// 	if err != nil {
// 		app.notFoundResponse(w, r)
// 		return
// 	}
// 	booklist, err := app.models.Booklists.Get(booklistId)
// 	if err != nil {
// 		switch {
// 		case errors.Is(err, data.ErrRecordNotFound):
// 			app.notFoundResponse(w, r)
// 		default:
// 			app.serverErrorResponse(w, r, err)
// 		}
// 		return
// 	}
// 	bookId, err := app.readIDParam(r, "bookId")
// 	if err != nil {
// 		app.notFoundResponse(w, r)
// 		return
// 	}
// 	book, err := app.models.Books.Get(bookId)
// 	if err != nil {
// 		switch {
// 		case errors.Is(err, data.ErrRecordNotFound):
// 			app.notFoundResponse(w, r)
// 		default:
// 			app.serverErrorResponse(w, r, err)
// 		}
// 		return
// 	}
// 	err = app.models.BooklistsBooks.Insert(booklist.ID, book.ID)
// 	if err != nil {
// 		app.serverErrorResponse(w, r, err)
// 		return
// 	}
// 	err = app.encodeJSON(w, http.StatusOK, envelope{"message": "book successfully added to booklist"}, nil)
// 	if err != nil {
// 		app.serverErrorResponse(w, r, err)
// 	}
// }
