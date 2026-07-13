package search

import (
	"os"
	"bytes"
	"bufio"
	"path/filepath"
	"fmt"
	"strings"
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
	return &SearchParams {
		Limit : 10,
		Dir : "/",
	}
}

func init() {
	// make it a tree, 
	// but what is the num of dirs and files are so large that it cant fit in the memory

	file, err := os.OpenFile("columbus_cache.txt",os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	err = Walk("/", w)
	if err != nil {
		panic(err)
	}

	err = w.Flush()
	if err != nil {
		panic(err)
	}
}

func Walk(dir string, w *bufio.Writer) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		str := fmt.Sprintf("%s | %v | %s \n", entry.Name(), entry.IsDir(), filepath.Join(dir, entry.Name()))
		_, err = w.Write([]byte(str))
		if err != nil {
			return err
		}

		if entry.IsDir() == true {
			Walk(filepath.Join(dir,entry.Name()), w)
		}
	}
	return nil
}

func Search(params *SearchParams, ans* []any) (error) {

	dir := params.Dir
	limit := params.Limit
	query := params.Query
	
	file, err := os.Open("columbus_cache.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	added := int64(0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		splitted := bytes.Split(scanner.Bytes(), []byte(" | "))
		for i,_ := range splitted {
			splitted[i] = bytes.TrimSpace(splitted[i])
		}
		name := splitted[0]
		isDir := splitted[1]
		path := splitted[2]

		var tans map[string]interface{}

		if bytes.HasPrefix(path, []byte(dir)) && bytes.Contains(name, []byte(query)) {

			if params.Ext != "" && !bytes.HasSuffix(name, []byte(params.Ext)) {
				continue
			} 

			if params.ExactMatch == true {
				if !bytes.Equal(name, []byte(query)) {
					continue
				}
			}
			
			ssed := strings.Split(string(name), ".")
			tans = map[string]interface{}{
						"Name" : string(name) ,
						"Type" : "FILE",
						"Path" : string(path),
						"Extension" : ssed[len(ssed)-1],
					}

			if bytes.Equal(isDir, []byte("true")){
				tans["Type"] = "FOLDER"
				tans["Extension"] = "FOLDER"
			}

			shouldAppend := true
			if tans["Type"] == "FOLDER" && params.Query_file_only == true {
				shouldAppend = false
			}else if tans["Type"] == "FILE" && params.Query_dir_only == true {
				shouldAppend = false
			}
			if shouldAppend {
				*ans = append(*ans, tans)
			}

			added += 1
			if added >= limit {
				break
			}
		}
	}
	fmt.Println(scanner.Err())
	return scanner.Err()
}


		