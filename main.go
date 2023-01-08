package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	sqlite3 "github.com/mattn/go-sqlite3"
)

// addr is the bind address for the web server.
const addr = ":8080"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()

	// Parse command line flags.
	dsn := flag.String("dsn", "", "datasource name")
	flag.Parse()
	if *dsn == "" {
		flag.Usage()
		return fmt.Errorf("required: -dsn")
	}

	// Open database file.
	db, err := sql.Open("sqlite3", *dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	isRO := func(query string) (bool, error) {
		c, err := db.Conn(context.Background())
		defer c.Close()
		if err != nil {
			return false, err
		}

		var ro bool
		err = c.Raw(func(dc interface{}) error {
			stmt, err := dc.(*sqlite3.SQLiteConn).Prepare(query)
			if err != nil {
				return err
			}
			if stmt == nil {
				return errors.New("stmt is nil")
			}
			ro = stmt.(*sqlite3.SQLiteStmt).Readonly()
			return nil
		})
		if err != nil {
			return false, err
		}
		return ro, nil // On errors ro will remain false.
	}

	// Run web server.
	fmt.Printf("listening on %s\n", addr)
	go http.ListenAndServe(addr,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/query" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}

			runInput := func() error {
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					return err
				}
				queries := strings.Split(string(body), ";")

				for _, query := range queries {

					ro, err := isRO(query)
					if err != nil {
						return err
					}
					if ro {
						rows, err := db.Query(query)
						defer rows.Close()
						if err != nil {
							return err
						}
						cols, err := rows.Columns()
						if err != nil {
							return err
						}
						numCols := len(cols)
						if numCols == 0 {
							continue
						}

						fmt.Fprintf(w, "%s\n", cols)

						// Result is your slice string.
						rawResult := make([][]byte, numCols)
						result := make([]string, numCols)

						dest := make([]interface{}, len(cols)) // A temporary interface{} slice
						for i := range rawResult {
							dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
						}
						for rows.Next() {
							// Scan the row into the slice of sql.RawBytes pointers
							err = rows.Scan(dest...)
							if err != nil {
								return err
							}

							for i, raw := range rawResult {
								if raw == nil {
									result[i] = "\\N"
								} else {
									result[i] = string(raw)
								}
							}
							fmt.Fprintf(w, "%s\n", result)
						}

					} else {
						result, err := db.Exec(query)
						if err != nil {
							return err
						}
						fmt.Fprintf(w, "%s\n", result)
					}
				}
				return nil
			}

			err := runInput()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}),
	)

	// Wait for signal.
	<-ctx.Done()
	log.Print("personaldb received signal, shutting down")

	return nil
}
