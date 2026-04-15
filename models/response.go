package models

import "fmt"

// ResponseJsonSingle digunakan untuk response dengan satu data.
type ResponseJsonSingle[T any] struct {
	Info   string `json:"info"`
	Result T      `json:"result"`
}

// ResponseJsonList digunakan untuk response dengan array data.
type ResponseJsonList[T any] struct {
	Info   string `json:"info"`
	Count  int    `json:"count"`
	Result []T    `json:"result"`
}

// ResponseMessage adalah payload sederhana berisi pesan string.
type ResponseMessage struct {
	Message string `json:"message"`
}

// Ok membuat response sukses single data.
func Ok[T any](result T) ResponseJsonSingle[T] {
	return ResponseJsonSingle[T]{
		Info:   "200 - OK",
		Result: result,
	}
}

// OkList membuat response sukses list data.
func OkList[T any](result []T) ResponseJsonList[T] {
	return ResponseJsonList[T]{
		Info:   "200 - OK",
		Count:  len(result),
		Result: result,
	}
}

// Err membuat response error dengan kode HTTP dan pesan.
func Err(code int, message string) ResponseJsonSingle[ResponseMessage] {
	return ResponseJsonSingle[ResponseMessage]{
		Info:   fmt.Sprintf("%d - Error", code),
		Result: ResponseMessage{Message: message},
	}
}
