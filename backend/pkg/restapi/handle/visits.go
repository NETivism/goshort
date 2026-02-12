package handle

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/netivism/goshort/backend/pkg/db"
	"github.com/netivism/goshort/backend/pkg/model"
)

type VisitsResp struct {
	Success int            `json:"success"`
	Message string         `json:"message"`
	Result  []model.Visits `json:"result"`
}

func Visits(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	id := params["id"]

	dbi := db.Get()
	var visits []model.Visits
	dbi.Where("redirect_id = ?", id).Find(&visits)

	resp := VisitsResp{
		Success: 1,
		Message: "Visits loaded successfully.",
		Result:  visits,
	}
	if len(visits) == 0 {
		resp.Result = make([]model.Visits, 0)
		resp.Message = "No visits found."
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
