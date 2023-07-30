package handler

import (
	"errors"
	"net/http"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/service"
)

func (h *Handler) listCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.ListCategories()
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"categories": categories}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

func (h *Handler) showCategoryHandler(w http.ResponseWriter, r *http.Request) {
	var qsInput dto.QsShowCategory
	v := validator.New()
	qs := r.URL.Query()
	qsInput.Filters.Page = h.readInt(qs, "page", 1, v)
	qsInput.Filters.PageSize = h.readInt(qs, "page_size", 10, v)
	qsInput.Filters.Sort = h.readString(qs, "sort", "-datetime")
	qsInput.Filters.SortSafeList = []string{"title", "size", "year", "datetime", "-title", "-size", "-year", "-datetime"}
	categoryID, err := h.readIDParam(r, "categoryId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	// Fetch category in order to use it as field name in JSON reponse.
	category, err := h.service.GetCategory(categoryID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	books, metadata, err := h.service.ShowCategory(category.ID, qsInput.Filters)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{category.Name: books, "metadata": metadata}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
