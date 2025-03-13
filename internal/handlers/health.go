package handlers

import (
	"fmt"
	"net/http"

	"github.com/babyfaceeasy/lema/internal/utils"
)

func (h Handler) Ping(w http.ResponseWriter, r *http.Request) {
	R := ResponseFormat{}

	R.Message = fmt.Sprintf("welcome to %s", h.config.AppName)
	code, res := h.response(http.StatusOK, R)
	utils.SendResponse(w, code, res)
}
