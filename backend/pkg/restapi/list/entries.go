package list

import (
	"net/http"
	"strconv"

	"github.com/netivism/goshort/backend/pkg/db"
	"github.com/netivism/goshort/backend/pkg/goshort"
	"github.com/netivism/goshort/backend/pkg/handler"
	"github.com/netivism/goshort/backend/pkg/model"
)

func Entries(w http.ResponseWriter, req *http.Request) {
	queryParams := req.URL.Query()
	offsetStr := queryParams.Get("offset")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}
	limit := 100

	dbi := db.Get()

	result := make([]goshort.GoShort, 0)
	records := []model.Redirect{}
	dbi.Limit(limit).Offset(offset).Find(&records)
	for _, record := range records {
		val := goshort.GoShort{
			Short:    record.Id,
			Redirect: record.Redirect,
		}
		result = append(result, val)
	}
	if len(result) > 0 {
		handler.HandlerSuccess(w, "Entries loaded successfully.", result, http.StatusOK)
	} else {
		handler.HandlerSuccess(w, "No entry found.", result, http.StatusOK)
	}
}
