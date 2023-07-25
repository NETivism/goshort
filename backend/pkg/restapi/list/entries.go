package list

import (
	"encoding/json"
	"net/http"
	"strconv"

	blt "github.com/netivism/goshort/backend/pkg/bolt"
	"github.com/netivism/goshort/backend/pkg/goshort"
	"github.com/netivism/goshort/backend/pkg/handler"
	bolt "go.etcd.io/bbolt"
)

func Entries(w http.ResponseWriter, req *http.Request) {
	queryParams := req.URL.Query()
	offsetStr := queryParams.Get("offset")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}
	limit := 100

	result := make([]goshort.GoShort, 0)

	blt.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blt.GoshortBucket)
		c := bucket.Cursor()

		count := -1
		for k, v := c.First(); k != nil; k, v = c.Next() {
			count++
			if count < offset {
				continue
			}
			var val goshort.GoShort
			err := json.Unmarshal(v, &val)

			if err == nil {
				result = append(result, val)
				if len(result) > limit {
					break
				}
			}
		}

		return nil
	})
	if len(result) > 0 {
		handler.HandlerSuccess(w, "Entries loaded successfully.", result, http.StatusOK)
	} else {
		handler.HandlerSuccess(w, "No entry found.", result, http.StatusOK)
	}
}
