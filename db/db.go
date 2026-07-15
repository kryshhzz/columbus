package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

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
			ext TEXT NOT NULL,
			sync_id INT NOT NULL
		)
	` 

type DirEntry struct {
	Dir string 
	Entry os.DirEntry	
}

var curSyncID int64


func init() {

	curSyncID = time.Time{}.Unix()

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
		fmt.Println("Created table 'entries' ", res)
	}
}


func delOldRows() {
	res, err := DB.Exec(`
		DELETE FROM entries WHERE sync_id != ?
	`, curSyncID)
	if err != nil {
		panic(err)
	}
	
	affrows, err := res.RowsAffected()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Deleted %v old rows\n", affrows)
}


// walk the dir and place it in a buffer 
// write to db; a go routine takes from the buffer

func writeToDB(buffer chan DirEntry) { 

	tx, err := DB.Begin()
	if err != nil {
		panic(err)
	}

	count := 0
	batch := 0
	batchSize := 10000

	for de := range buffer {
		ext := "FOLDER"
		if de.Entry.IsDir() == false {
			ext = filepath.Ext(de.Entry.Name())
		}
		
		path := filepath.Join(de.Dir, de.Entry.Name())
		
		q := `
		INSERT INTO entries (name, path, is_dir, ext, sync_id) 
		VALUES (?,?,?,?,?)
		ON CONFLICT(path) DO UPDATE SET sync_id = excluded.sync_id;
		`
		_, err := tx.Exec(q,  de.Entry.Name(), path, de.Entry.IsDir(), ext, curSyncID)
		if err != nil {
			fmt.Println("rolling back")
			tx.Rollback()
			panic(err)
		}
		count += 1 
		batch += 1

		if batch >= batchSize {
			batch = 0
			// 
			err = tx.Commit()
			if err != nil {
				panic(err)
			}
			fmt.Printf("\r\033[K[DB] Committed items: %d...", count)

			// creating new tx
			tx, err = DB.Begin()
			if err != nil {
				panic(err)
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("\nAdded %v entries to the database\n", count)

	// delete the prev rows 
	delOldRows()
}


func CacheFS() {

	start := time.Now()
	buffer := make(chan DirEntry, 10000)
	
	// db side
	wg2 := sync.WaitGroup{}
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		writeToDB(buffer)
	}()

	// file walk side
	quit := make(chan struct{})
	errch := make(chan error)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go Walk("/", buffer, errch, &wg)

	go func() {
		wg.Wait()
		close(quit) 
		close(buffer)
	}()
		
	mainLoop:
	for {
		select {
		case err := <- errch :
			if !os.IsPermission(err) && !os.IsNotExist(err) {
				panic(err)
			}
			case <- quit : 
			break mainLoop	
		}
	}

	wg2.Wait()
	fmt.Printf("Time taken for caching : %v \n",time.Since(start))

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