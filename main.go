package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/kryshhzz/columbus/search"
	"github.com/kryshhzz/columbus/open"
	"github.com/kryshhzz/columbus/db"
)

type searchData struct {
	Query string 			  `json:"query"`
	Flags []map[string]string `json:"flags"`
}

func formatSize(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unitIndex := 0
	for size >= 1024 && unitIndex < len(units)-1 {
		size /= 1024
		unitIndex++
	}
	return fmt.Sprintf("%d %s", size, units[unitIndex])
}

func main() {	
	// cache the fs first 
	log.Print("caching the file system ...  ")
 	db.CacheFS()
	log.Println("Done Caching")

	r := gin.Default()

	r.LoadHTMLGlob("templates/*.tmpl")
	r.Static("/assets", "./assets") 

	r.POST("/search", func(c *gin.Context) {

		srchdata := searchData{}
		c.BindJSON(&srchdata)
		
		// as it is compulsory
		params := search.NewSearchParams()
		params.Query = srchdata.Query

		// all others are optional		
		for _, flag := range srchdata.Flags {
			switch flag["key"] {
			case "d" :
				// direcotry 
				params.Query_dir_only = true
			case "f" :
				// fiel only 
				params.Query_file_only = true
			case "dir" :
				// search dir
				params.Dir = flag["value"]
			case "l" :
				// results limit 
				limit_s := flag["value"]
				limit_int, err := strconv.ParseInt(limit_s, 10, 64)
				if err == nil {
					params.Limit = limit_int
				}
			case "e" :
				// extension 
				params.Ext = flag["value"]

			case "eq" :
				// exact match 
				params.ExactMatch = true

			case "default" :
				// nothing
			}
		}
		

		cont := []interface{}{}
		err := search.Search(params, &cont)
		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error" : err.Error(),
				},
			)
			return 
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"Entries" : cont,
			},
		)
	})

	r.GET("/open", func(c * gin.Context){
		file := c.DefaultQuery("file", "")
		err := open.Open(file)
		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error" : err.Error(),
				},
			)
			return 
		}
	})


	r.GET("/", func(c *gin.Context) {

		var err error
		path := c.DefaultQuery("path", "")
		if path == "" {
			path, err = os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
		}

		cont := map[string]interface{}{}

		entries, err := os.ReadDir(path)
		if err != nil {
			c.HTML(
				200, 
				"home.tmpl",
				gin.H{
					"Entries" : cont,
					"Path" : path,
				},
			)
			return
		}

		for _, entry := range entries {

			info, err := entry.Info()
			if err != nil {
				log.Fatal(err)
			}

			tmp := map[string]interface{}{
				"Name" : entry.Name(),
				"Size" : formatSize(info.Size()),
				"Date" : info.ModTime().Format("Monday, Jan _2, 2006 at 03:04PM"),
				"Path" : path + "/" + entry.Name(),
			}

			if entry.IsDir() == true{
				tmp["Type"] = "FOLDER"
				tmp["Extension"] = "FOLDER"
			}else{
				tmp["Type"] = "FILE"
				splitted := strings.Split(entry.Name(), ".")
				tmp["Extension"] = splitted[len(splitted)-1]
			}

			cont[entry.Name()] = tmp;
		}

		c.HTML(
			200, 
			"home.tmpl",
			gin.H{
				"Entries" : cont,
				"Path" : path,
			},
		)
	})

	r.Run(":9061")
}