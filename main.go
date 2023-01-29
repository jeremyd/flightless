package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/awesome-gocui/gocui"
	"github.com/joho/godotenv"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Metadata struct {
	PubkeyHex    string `gorm:"primaryKey;size:256"`
	Name         string `gorm:"size:1024"`
	About        string `gorm:"size:4096"`
	Nip05        string `gorm:"size:512"`
	Lud06        string `gorm:"size:2048"`
	Lud16        string `gorm:"size:512"`
	Website      string `gorm:"size:512"`
	DisplayName  string `gorm:"size:512"`
	Picture      string `gorm:"type:text;size:65535"`
	TotalFollows int
	UpdatedAt    time.Time         `gorm:"autoUpdateTime"`
	Follows      []*Metadata       `gorm:"many2many:metadata_follows"`
	Servers      []RecommendServer `gorm:"foreignKey:PubkeyHex;references:PubkeyHex"`
}

type RecommendServer struct {
	PubkeyHex     string    `gorm:"primaryKey;size:256"`
	Url           string    `gorm:"size:512"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
	CreatedAt     time.Time `gorm:"autoUpdateTime"`
	RecommendedBy string    `gorm:"size:256"`
}

type RelayStatus struct {
	Url       string    `gorm:"primaryKey;size:512"`
	Status    string    `gorm:"size:512"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type GormErr struct {
	Number  int    `json:"Number"`
	Message string `json:"Message"`
}

func CheckAndReportGormError(err error, allowErrors []string) bool {
	if err != nil {
		for _, e := range allowErrors {
			if err.Error() == e {
				//fmt.Println("known error: " + e)
				return true
			}
		}
		byteErr, _ := json.Marshal(err)
		var newError GormErr
		json.Unmarshal((byteErr), &newError)
		//fmt.Println(newError)
		return false
	}
	return true
}

func UpdateOrCreateRelayStatus(db *gorm.DB, url string, status string) {
	rowsUpdated := db.Model(RelayStatus{}).Where("url = ?", url).Updates(&RelayStatus{Url: url, Status: status}).RowsAffected
	if rowsUpdated == 0 {
		db.Create(&RelayStatus{Url: url, Status: status})
	}
}

func GetGormConnection() *gorm.DB {
	file, err := os.OpenFile("nono.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		// Handle error
		panic(err)
	}
	newLogger := logger.New(
		log.New(file, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Error, // Log level
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,        // Disable color
		},
	)

	dsn, foundDsn := os.LookupEnv("DB")
	if !foundDsn {
		dsn = "nono.db?cache=shared&mode=rwc"
	}

	db, dberr := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: newLogger})
	if dberr != nil {
		panic("failed to connect database")
	}
	db.Logger.LogMode(logger.Silent)
	sql, _ := db.DB()
	sql.SetMaxOpenConns(1)

	return db
}

var ViewDB *gorm.DB

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx := context.Background()

	DB := GetGormConnection()
	ViewDB = DB

	migrateErr := DB.AutoMigrate(&Metadata{})
	migrateErr2 := DB.AutoMigrate(&RelayStatus{})
	migrateErr3 := DB.AutoMigrate(&RecommendServer{})
	if migrateErr != nil || migrateErr2 != nil || migrateErr3 != nil {
		panic("one or more migrations failed, aborting")
	}

	// connect to relay(s)
	//DB.Exec("delete from relay_statuses")
	var relayUrls []string
	var relayStatuses []RelayStatus
	DB.Find(&relayStatuses)
	if len(relayStatuses) == 0 {
		fmt.Println("error finding relay urls")
		relayUrls = []string{
			//"wss://relay.snort.social",
			//"wss://relay.damus.io",
			//"wss://nostr.zebedee.cloud",
			//"wss://eden.nostr.land",
			//"wss://nostr-pub.wellorder.net",
			"wss://nostr-dev.wellorder.net",
			//"wss://relay.nostr.info",
		}
	} else {
		for _, relayStatus := range relayStatuses {
			relayUrls = append(relayUrls, relayStatus.Url)
		}
	}

	for _, url := range relayUrls {
		doRelay(DB, ctx, url)
	}

	g, err := gocui.NewGui(gocui.OutputTrue, true)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)
	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}

	// relay status manager
	go func() {
		for {
			var RelayStatuses []RelayStatus
			DB.Find(&RelayStatuses)
			for _, relayStatus := range RelayStatuses {

				if relayStatus.Status == "waiting" {
					doRelay(DB, ctx, relayStatus.Url)
				} else if relayStatus.Status == "deleting" {
					foundit := false
					for _, r := range nostrRelays {
						if r.URL == relayStatus.Url {
							err := DB.Delete(&relayStatus).Error
							if err != nil {
								fmt.Println(err)
							}
							foundit = true
							r.Close()
						}
					}
					// if we didn't find it, delete the record anyway
					if !foundit {
						err := DB.Delete(&relayStatus).Error
						if err != nil {
							fmt.Println(err)
						}
					}
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}
