package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/awesome-gocui/gocui"
	//"gorm.io/driver/sqlite"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var AppInfo = "flightless v0.0.4"

type Metadata struct {
	PubkeyHex         string `gorm:"primaryKey;size:65"`
	PubkeyNpub        string `gorm:"size:65"`
	Name              string `gorm:"size:1024"`
	About             string `gorm:"size:4096"`
	Nip05             string `gorm:"size:512"`
	Lud06             string `gorm:"size:2048"`
	Lud16             string `gorm:"size:512"`
	Website           string `gorm:"size:512"`
	DisplayName       string `gorm:"size:512"`
	Picture           string `gorm:"type:text;size:65535"`
	TotalFollows      int
	UpdatedAt         time.Time `gorm:"autoUpdateTime"`
	ContactsUpdatedAt time.Time
	MetadataUpdatedAt time.Time
	Follows           []*Metadata       `gorm:"many2many:metadata_follows"`
	Servers           []RecommendServer `gorm:"foreignKey:PubkeyHex;references:PubkeyHex"`
}

type RecommendServer struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	PubkeyHex     string    `gorm:"size:65"`
	Url           string    `gorm:"size:512"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
	CreatedAt     time.Time `gorm:"autoUpdateTime"`
	RecommendedBy string    `gorm:"size:256"`
}

type RelayStatus struct {
	Url       string    `gorm:"primaryKey;size:512"`
	Status    string    `gorm:"size:512"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	LastEOSE  time.Time
	LastDisco time.Time
}

type Account struct {
	Pubkey     string `gorm:"primaryKey;size:65"`
	PubkeyNpub string `gorm:"size:65"`
	Privatekey string `gorm:"primaryKey;size:65"` // encrypted
	Active     bool
}

type Login struct {
	PasswordHash string `gorm:"size:43"` //salted and hashed
}

var TheLog *log.Logger

func GetGormConnection() *gorm.DB {
	file, err := os.OpenFile("flightless.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		// Handle error
		panic(err)
	}

	TheLog = log.New(file, "", log.LstdFlags) // io writer
	newLogger := logger.New(
		TheLog,
		logger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Error, // Log level
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,        // Disable color
		},
	)

	dsn, foundDsn := os.LookupEnv("DB")
	if !foundDsn {
		dsn = "flightless.db?cache=shared&mode=rwc"
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

var Password []byte

var CTX context.Context

func main() {

	/*
		err := godotenv.Load()
		if err != nil {
			log.Info("Error loading .env file, using defaults")
		}
	*/

	CTX = context.Background()

	DB := GetGormConnection()
	ViewDB = DB

	migrateErr := DB.AutoMigrate(&Metadata{})
	migrateErr2 := DB.AutoMigrate(&RelayStatus{})
	migrateErr3 := DB.AutoMigrate(&RecommendServer{})
	migrateErr4 := DB.AutoMigrate(&Login{})
	migrateErr5 := DB.AutoMigrate(&Account{})

	migrateErrs := []error{
		migrateErr,
		migrateErr2,
		migrateErr3,
		migrateErr4,
		migrateErr5,
	}
	for i, err := range migrateErrs {
		if err != nil {
			fmt.Println("Error running a migration (%d) %s\nexiting.", i, err)
			os.Exit(1)
		}
	}

	// Login

	var login Login
	loginDbErr := DB.First(&login).Error

	if loginDbErr != nil || login.PasswordHash == "" {
		fmt.Println("no login found, create a new password")
		Password = GetNewPwd()
		login.PasswordHash = HashAndSalt(Password)
		DB.Create(&login)
		fmt.Println("login created, loading...")
	} else {
		Password = GetPwd()
		success := ComparePasswords(login.PasswordHash, Password)
		if success {
			fmt.Println("login success, loading...")
		} else {
			fmt.Println("login failed")
			os.Exit(1)
		}
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
		doRelay(DB, CTX, url)
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
					doRelay(DB, CTX, relayStatus.Url)
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
