package handle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/netivism/goshort/backend/pkg/db"
	"github.com/netivism/goshort/backend/pkg/handler"
	"github.com/netivism/goshort/backend/pkg/model"
)

type BatchInfoItem struct {
	Id       string `json:"id"`
	Redirect string `json:"redirect"`
	Total    int64  `json:"total"`
}

type BatchInfoResp struct {
	Success int             `json:"success"`
	Message string          `json:"message"`
	Result  []BatchInfoItem `json:"result"`
}

func BatchInfo(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	var ids []string
	if err := json.NewDecoder(req.Body).Decode(&ids); err != nil {
		handler.HandlerError(w, "Request body must be a JSON array of redirect IDs.", http.StatusBadRequest)
		return
	}
	if len(ids) == 0 {
		handler.HandlerError(w, "At least one redirect ID is required.", http.StatusBadRequest)
		return
	}

	dbi := db.Get()

	var redirects []model.Redirect
	dbi.Where("id IN ?", ids).Find(&redirects)

	redirectMap := make(map[string]string, len(redirects))
	for _, r := range redirects {
		redirectMap[r.Id] = r.Redirect
	}

	var stats []model.Statistics
	dbi.Where("redirect_id IN ?", ids).Find(&stats)

	totalMap := make(map[string]int64, len(stats))
	for _, s := range stats {
		var res model.VisitsStatResult
		if err := json.Unmarshal([]byte(s.Result), &res); err == nil {
			totalMap[s.RedirectId] = res.Total
		}
	}

	result := make([]BatchInfoItem, 0, len(ids))
	for _, id := range ids {
		result = append(result, BatchInfoItem{
			Id:       id,
			Redirect: redirectMap[id],
			Total:    totalMap[id],
		})
	}

	resp := BatchInfoResp{
		Success: 1,
		Message: "Redirect info loaded successfully.",
		Result:  result,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

const batchCreateLimit = 1000

type BatchCreateItem struct {
	Redirect string `json:"redirect"`
}

type BatchCreateResult struct {
	Redirect string `json:"redirect"`
	Short    string `json:"short,omitempty"`
	Error    string `json:"error,omitempty"`
}

type BatchCreateResp struct {
	Success int                 `json:"success"`
	Message string              `json:"message"`
	Result  []BatchCreateResult `json:"result"`
}

func BatchCreate(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	var items []BatchCreateItem
	if err := json.NewDecoder(req.Body).Decode(&items); err != nil {
		handler.HandlerError(w, "Request body must be a JSON array of objects with a redirect field.", http.StatusBadRequest)
		return
	}
	if len(items) == 0 {
		handler.HandlerError(w, "At least one entry is required.", http.StatusBadRequest)
		return
	}
	if len(items) > batchCreateLimit {
		handler.HandlerError(w, fmt.Sprintf("Batch size exceeds limit of %d.", batchCreateLimit), http.StatusBadRequest)
		return
	}

	// Phase 1: validate all entries before inserting anything.
	for i, item := range items {
		if item.Redirect == "" {
			handler.HandlerError(w, fmt.Sprintf("Entry %d missing redirect address.", i), http.StatusBadRequest)
			return
		}
		r, err := url.Parse(item.Redirect)
		if err != nil || (r.Scheme != "http" && r.Scheme != "https") {
			handler.HandlerError(w, fmt.Sprintf("Entry %d: redirection format must be http/s url.", i), http.StatusBadRequest)
			return
		}
		if r.User.Username() != "" {
			handler.HandlerError(w, fmt.Sprintf("Entry %d: redirection format shouldn't include username or password.", i), http.StatusBadRequest)
			return
		}
	}

	// Phase 2: insert one by one; on error record and continue.
	dbi := db.Get()
	result := make([]BatchCreateResult, 0, len(items))
	successCount := 0

	for _, item := range items {
		uhash, err := GenerateUniqueHash()
		if err != nil {
			result = append(result, BatchCreateResult{
				Redirect: item.Redirect,
				Error:    fmt.Sprintf("hash generation error: %v", err),
			})
			continue
		}
		r, _ := url.Parse(item.Redirect)
		record := model.Redirect{
			Id:       uhash,
			Redirect: item.Redirect,
			Domain:   r.Host,
			Path:     r.Path,
		}
		if err := dbi.Create(&record).Error; err != nil {
			result = append(result, BatchCreateResult{
				Redirect: item.Redirect,
				Error:    fmt.Sprintf("error saving record: %v", err),
			})
			continue
		}
		result = append(result, BatchCreateResult{
			Redirect: item.Redirect,
			Short:    uhash,
		})
		successCount++
	}

	failCount := len(items) - successCount
	message := fmt.Sprintf("%d/%d URLs shortened successfully.", successCount, len(items))
	if failCount > 0 {
		message += fmt.Sprintf(" %d failed.", failCount)
	}

	resp := BatchCreateResp{
		Success: 1,
		Message: message,
		Result:  result,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
