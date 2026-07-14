package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"database/sql"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

var createTableQuery string = `
		CREATE TABLE IF NOT EXISTS entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			is_dir BOOLEAN NOT NULL,
			ext TEXT NOT NULL
		)
	` 

type DirEntry struct {
	Dir string 
	Entry os.DirEntry	
}


func init() {

	baseDir, err := os.UserHomeDir()
	if err != nil {
		baseDir = ""
	}

	dsnURI := filepath.Join(baseDir, "columbus.db")
	DB, err = sql.Open("sqlite", dsnURI)
	if err != nil {
		panic(err)
	}

	// some claude shit for speed
	// Big speed win: WAL mode + relaxed sync for bulk writes
	if _, err := DB.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		panic(err)
	}
	if _, err := DB.Exec(`PRAGMA synchronous=NORMAL;`); err != nil {
		panic(err)
	}
	// claude shit ends

	err = DB.Ping()
	if err != nil {
		panic(fmt.Errorf("Could not create or connect to database: %v",err))
	}

	fmt.Println("Database file verified/created successfully!")
	res, err := DB.Exec(createTableQuery)
	if err != nil {
		panic(err)
	}else {
		fmt.Println("Created table entries ", res)
	}

}


// walk the dir and place it in a buffer 
// write to db; a go routine takes from the buffer

func writeToDB(tx *sql.Tx, de DirEntry) {

	ext := "FOLDER"
	if de.Entry.IsDir() == false {
		ext = filepath.Ext(de.Entry.Name())
	}

	path := filepath.Join(de.Dir, de.Entry.Name())

	q := `
		INSERT OR IGNORE INTO entries (name, path, is_dir, ext) 
		VALUES (?,?,?,?)
	`
	_, err := tx.Exec(q,  de.Entry.Name(), path, de.Entry.IsDir(), ext)
	if err != nil {
		fmt.Println("rolling back")
		tx.Rollback()
		panic(err)
	}
}


func CacheFS() {
	var err error  
	buffer := make(chan DirEntry, 1000)

	tx,err := DB.Begin()
	if err != nil {
		panic(err)
	}

	quit := make(chan struct{})
	errch := make(chan error)
	count := 0
	batch := 0

	wg := sync.WaitGroup{}
	wg.Add(1)
	go Walk("/", buffer, errch, &wg)

	go func() {
		wg.Wait()
		close(quit)
	}()

	mainLoop:
	for {
		select {
			case entry := <- buffer :
				writeToDB(tx, entry)
				count += 1
				batch += 1	
			case err := <- errch :
				if !os.IsPermission(err) && !os.IsNotExist(err) {
					panic(err)
				}
			case <- quit :
				break mainLoop	
		}

		if batch >= 5000 {
			err = tx.Commit()
			if err != nil {
				fmt.Println("couldnot commit")
				panic(err)
			}

			tx,err = DB.Begin()
			if err != nil {
				fmt.Println("couldnot create new tx")
				panic(err)
			}	 
		}
	}

	// extra committing
	err = tx.Commit()
	if err != nil {
		fmt.Println("couldnot commit")
		panic(err)
	}
	
	
	fmt.Printf("Added %v entries to the database\n", count)
}



func Walk(dir string, buffer chan DirEntry, errch chan error, wg *sync.WaitGroup) {

	defer func() {
		wg.Done()
	}()

	entries, err := os.ReadDir(dir)
	if err != nil {
		errch <- err 
		return 
	}
	for _, entry := range entries {
		buffer <- DirEntry{
			Dir : dir,
			Entry : entry,
		}
		if entry.IsDir() == true {
			wg.Add(1)
			go Walk(filepath.Join(dir,entry.Name()), buffer, errch, wg)
		}

	}
}