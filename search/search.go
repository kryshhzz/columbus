package search

import (
	// "os"
	// "bytes"
	// "bufio"
	// "strings"
	"fmt"
	"runtime"

	"github.com/kryshhzz/columbus/db"
)

type SearchParams struct {
	Query string 
	Dir string
	Limit int64
	Query_file_only bool
	Query_dir_only bool
	Ext string
	ExactMatch bool
}

func NewSearchParams() *SearchParams {
	rootDir := "/"

	if runtime.GOOS == "windows" {
		rootDir = "C:\\"
	}

	return &SearchParams {
		Limit : 10,
		Dir : rootDir,
	}
}

func Search(params *SearchParams, ans* []any) (error) {

	fmt.Println(params)

	args := []any{}

	sqlQuery := `SELECT * FROM entries WHERE path LIKE ? `
	args = append(args, params.Dir + "%")

	// adding name, exactmatch
	if params.ExactMatch {
		sqlQuery += `
		AND name = ?
		`
		args = append(args, params.Query)
	}else{
		sqlQuery += `
		AND name LIKE ?
		`
		args = append(args, "%" + params.Query + "%")
	}
	

	// dir only file only
	if params.Query_dir_only {
		sqlQuery += `
			AND is_dir = true
		`
	}else if params.Query_file_only {
		sqlQuery += `
			AND is_dir = false
		`
	}

	// extension
	if params.Ext != "" {
		sqlQuery += `
			AND ext = ?
		`
		args = append(args, params.Ext)
	}

	sqlQuery += `
		LIMIT ?	
	`
	args = append(args, params.Limit)

	//fmt.Println(sqlQuery)

	rows, err := db.DB.Query(sqlQuery, args...)
	if err != nil {
		return err
	}

	for rows.Next() {
		
		var id int64 
		var name string
		var path string 
		var is_dir bool 
		var ext string
		
		err := rows.Scan(&id, &name, &path, &is_dir, &ext, &id)
		if err != nil {
			panic(err)
		}
		
		tans := map[string]string{
					"Name" : name,
					"Type" : "FILE",
					"Path" : path,
					"Extension" : ext,
				}

		if is_dir {
			tans["Type"] = "FOLDER"
			tans["Ext"] = "FOLDER"
		}
		
		// fmt.Println(tans)
		*ans = append(*ans, tans)
	}

	return nil
}

	