package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"mypaas/internal/dbstudio"
)

type output struct {
	OK       bool                          `json:"ok"`
	Response dbstudio.SQLiteHelperResponse `json:"response,omitempty"`
	Error    string                        `json:"error,omitempty"`
}

func main() {
	decoder := json.NewDecoder(os.Stdin)
	var request dbstudio.SQLiteHelperRequest
	if err := decoder.Decode(&request); err != nil {
		write(output{OK: false, Error: "invalid request"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	response, err := dbstudio.ExecuteSQLiteHelper(ctx, request)
	if err != nil {
		write(output{OK: false, Error: err.Error()})
		return
	}
	write(output{OK: true, Response: response})
}

func write(value output) {
	if err := json.NewEncoder(os.Stdout).Encode(value); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}
