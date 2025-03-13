package utils

import (
	"net/http"

	"github.com/bytedance/sonic"
)

func SendResponse(w http.ResponseWriter, statusCode int, response any) {
	data, _ := sonic.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)
}
