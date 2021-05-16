package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/browser"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	cmdStartHeadline  = "Starts the vocab web application."
	cmdExportHeadline = "Export vocab to a CSV."
	cmdImportHeadline = "Import vocab from a CSV."
)

func main() {
	if len(os.Args) < 2 {
		cmdNotRecognized()
	}
	cmd := os.Args[1]
	if cmd == "start" {
		cmdStart(os.Args[2:])
	} else if cmd == "export" {
		cmdExport(os.Args[2:])
	} else if cmd == "import" {
		cmdImport(os.Args[2:])
	} else {
		cmdNotRecognized()
	}

}

func cmdNotRecognized() {
	fmt.Fprintf(os.Stderr, "\nUsage: vocab <command>\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  start     "+cmdStartHeadline+"\n")
	fmt.Fprintf(os.Stderr, "  export    "+cmdExportHeadline+"\n")
	fmt.Fprintf(os.Stderr, "  import    "+cmdImportHeadline+"\n\n")
	fmt.Fprintf(os.Stderr, "Run 'vocab <command> -help' for more information about a command.\n\n")
	os.Exit(1)
}

func cmdStart(args []string) {
	flg := flag.NewFlagSet("", flag.ExitOnError)
	flg.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n"+cmdStartHeadline+"\n\n")
		fmt.Fprintf(os.Stderr, "Usage: vocab start [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flg.PrintDefaults()
	}

	var port string
	flg.StringVar(&port, "port", "3000", "Port on which to serve the application")
	var openBrowser bool
	flg.BoolVar(&openBrowser, "open", false, "Automatically open a web browser")

	err := flg.Parse(args)
	if err != nil {
		log.Fatal(err)
	}

	db, err := getDb()
	if err != nil {
		log.Fatal(err)
	}

	server := NewServer(db)

	if openBrowser {
		go func() {
			time.Sleep(time.Millisecond * 500)
			browser.OpenURL("http://localhost:" + port)
		}()
	}

	log.Println("Serving at http://localhost:" + port)
	log.Fatal(http.ListenAndServe(":"+port, server))
}

func cmdExport(args []string) {
	flg := flag.NewFlagSet("", flag.ExitOnError)
	flg.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n"+cmdExportHeadline+"\n\n")
		fmt.Fprintf(os.Stderr, "Usage: vocab export [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flg.PrintDefaults()
	}

	var fileS string
	flg.StringVar(&fileS, "file", "vocab.csv", "File path to export the CSV")

	err := flg.Parse(args)
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Create(fileS)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	db, err := getDb()
	if err != nil {
		log.Fatal(err)
	}

	csv := NewCsv(db)
	err = csv.Export(file)
	if err != nil {
		log.Fatal(err)
	}
}

func cmdImport(args []string) {
	flg := flag.NewFlagSet("", flag.ExitOnError)
	flg.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n"+cmdImportHeadline+"\n\n")
		fmt.Fprintf(os.Stderr, "Usage: vocab import [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flg.PrintDefaults()
	}

	var fileS string
	flg.StringVar(&fileS, "file", "", "File path to the import CSV")
	var clean bool
	flg.BoolVar(&clean, "clean", false, "Clean import will delete all existing vocab")

	err := flg.Parse(args)
	if err != nil {
		log.Fatal(err)
	}

	if fileS == "" {
		log.Fatal("flag -file is required")
	}

	file, err := os.Open(fileS)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	db, err := getDb()
	if err != nil {
		log.Fatal(err)
	}

	csv := NewCsv(db)

	if clean {
		err = csv.ImportClean(file)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err = csv.Import(file)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getDb() (*gorm.DB, error) {
	dir, err := appdir()
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "vocab.db")))
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&Vocab{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func appdir() (string, error) {
	hd, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(hd, ".vocab")
	_, err = os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.Mkdir(dir, 0700); err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	return dir, nil
}

type Vocab struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"-"`
	Term           string    `json:"term"`
	Translation    string    `json:"translation"`
	KnowledgeLevel uint      `json:"knowledgeLevel"`
	PracticeAt     time.Time `json:"practiceAt"`
}
