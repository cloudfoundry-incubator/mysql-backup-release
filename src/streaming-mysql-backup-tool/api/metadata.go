package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

type metadata struct {
	IsFollower bool `json:"is_follower"`
}

func Metadata(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		rows, err := db.Query(`SHOW SLAVE STATUS`)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, err.Error())
		}

		defer rows.Close()

		var response metadata
		if rows.Next() { // non-followers will not have a next row
			response.IsFollower = true
		}

		responseBytes, err := json.Marshal(&response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, _ = w.Write(responseBytes)
	}
}
