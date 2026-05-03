package handlers

import (
	"net/http"

	"holo/internal/components"
)

func ConnectPage(w http.ResponseWriter, r *http.Request) {
	components.ConnectPage().Render(r.Context(), w)
}
