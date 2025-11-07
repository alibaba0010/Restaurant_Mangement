package errors

import "net/http"

func ValidationError(message string) *AppError {
	return New("Validation Error", message, http.StatusBadRequest, nil)
}

func DuplicateError(field string) *AppError {
	return New("Duplicate Value", "Duplicate value entered for "+field+" field, please choose another value", http.StatusBadRequest, nil)
}

func NotFoundError(message string) *AppError {
	return New("Not Found", message, http.StatusNotFound, nil)
}

func InternalError(err error) *AppError {
	return New("Internal Server Error", "Something went wrong, try again later", http.StatusInternalServerError, err)
}

func RouteNotExist() *AppError {
	return New("Route Error", "Route does not exist", http.StatusNotFound, nil)
}
