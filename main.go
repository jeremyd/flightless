package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Metadata struct {
	PubkeyHex   string            `gorm:"primaryKey;size:256"`
	Name        string            `gorm:"size:1024"`
	About       string            `gorm:"size:4096"`
	Nip05       string            `gorm:"size:512"`
	Lud06       string            `gorm:"size:2048"`
	Lud16       string            `gorm:"size:512"`
	Website     string            `gorm:"size:512"`
	DisplayName string            `gorm:"size:512"`
	Picture     string            `gorm:"type:text;size:65535"`
	UpdatedAt   time.Time         `gorm:"autoUpdateTime"`
	Follows     []*Metadata       `gorm:"many2many:metadata_follows"`
	Servers     []RecommendServer `gorm:"foreignKey:PubkeyHex;references:PubkeyHex"`
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

func checkAndReportGormError(err error, allowErrors []string) bool {
	if err != nil {
		for _, e := range allowErrors {
			if err.Error() == e {
				fmt.Println("known error: " + e)
				return true
			}
		}
		byteErr, _ := json.Marshal(err)
		var newError GormErr
		json.Unmarshal((byteErr), &newError)
		fmt.Println(newError)
		return false
	}
	return true
}

func updateOrCreateRelayStatus(db *gorm.DB, url string, status string) {
	rowsUpdated := db.Model(RelayStatus{}).Where("url = ?", url).Updates(&RelayStatus{Url: url, Status: status}).RowsAffected
	if rowsUpdated == 0 {
		db.Create(&RelayStatus{Url: url, Status: status})
	}
}

func doRelay(db *gorm.DB, ctx context.Context, url string) bool {
	relay, err := nostr.RelayConnect(ctx, url)
	if err != nil {
		fmt.Printf("failed initial connection to relay: %s, %s; skipping relay\n", url, err)
		updateOrCreateRelayStatus(db, url, "failed initial connection")
		return false
	}

	updateOrCreateRelayStatus(db, url, "connection established")

	// create filters
	filters := []nostr.Filter{{
		Kinds: []int{0, 2, 3},
		//Tags:  t,
		// limit = 3, get the three most recent notes
		Limit: 10,
	}}

	// create a subscription and submit to relay
	sub := relay.Subscribe(ctx, filters)

	go func() {
		<-sub.EndOfStoredEvents
		fmt.Printf("got EOSE from %s\n", relay.URL)
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("exiting gracefully")
		sub.Unsub()
		relay.Close()
		// give other relays time to close
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}()

	go func() {
		for ev := range sub.Events {
			if ev.Kind == 0 {
				// Metadata
				m := Metadata{}
				err := json.Unmarshal([]byte(ev.Content), &m)
				if err != nil {
					fmt.Println(err)
				}
				m.PubkeyHex = ev.PubKey
				if len(m.Picture) > 65535 {
					fmt.Println("dumbass put too big a picture, skipping")
					continue
				}
				rowsUpdated := db.Model(Metadata{}).Where("pubkey_hex = ?", m.PubkeyHex).Updates(&m).RowsAffected
				if rowsUpdated == 0 {
					err := db.Save(&m).Error
					if err != nil {
						fmt.Println(err)
					}
					fmt.Printf("Created metadata for %s, %s\n", m.Name, m.Nip05)
				} else {
					fmt.Printf("Updated metadata for %s, %s\n", m.Name, m.Nip05)
				}
			} else if ev.Kind == 2 {
				// recommend relay
				fmt.Println("FOUND TYPE 2!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
				var servers []RecommendServer
				db.Find(&servers, "pubkey_hex = ? and url = ? and recommended_by = ?", ev.PubKey, ev.Content, ev.PubKey)
				if len(servers) > 0 {
					// already recommended, update time fields?
					fmt.Println("already recommended, skip")
				} else {
					// add to recommended servers
					db.Create(&RecommendServer{
						PubkeyHex:     ev.PubKey,
						Url:           ev.Content,
						RecommendedBy: ev.PubKey,
					})
				}
			} else if ev.Kind == 3 {
				// Contact List
				pTags := []string{"p"}
				allPTags := ev.Tags.GetAll(pTags)
				var person Metadata
				notFoundError := db.First(&person, "pubkey_hex = ?", ev.PubKey).Error
				if notFoundError != nil {
					fmt.Printf("Creating blank metadata for %s\n", ev.PubKey)
					person = Metadata{
						PubkeyHex: ev.PubKey,
					}
					db.Create(&person)
				} else {
					fmt.Printf("updating (%d) follows for %s: %s\n", len(allPTags), person.Name, person.PubkeyHex)
				}

				// purge followers that have been 'unfollowed'
				var oldFollows []Metadata
				db.Model(&person).Association("Follows").Find(&oldFollows)
				for _, oldFollow := range oldFollows {
					found := false
					for _, n := range allPTags {
						if n[1] == oldFollow.PubkeyHex {
							found = true
						}
					}
					if !found {
						db.Exec("delete from metadata_follows where metadata_pubkey_hex = ? and follow_pubkey_hex = ?", person.PubkeyHex, oldFollow.PubkeyHex)
					}
				}

				for _, c := range allPTags {
					var followPerson Metadata
					notFoundFollow := db.First(&followPerson, "pubkey_hex = ?", c[1]).Error
					if notFoundFollow != nil {
						// follow user not found, need to create it
						var newUser Metadata
						// follow user recommend server suggestion if it exists
						if len(c) >= 3 && c[2] != "" {
							newUser = Metadata{
								PubkeyHex: c[1],
								Servers:   []RecommendServer{{Url: c[2], RecommendedBy: person.PubkeyHex}},
							}
						} else {
							newUser = Metadata{PubkeyHex: c[1]}
						}
						createNewErr := db.Omit("Follows").Create(&newUser).Error
						if createNewErr != nil {
							fmt.Println("Error creating user for follow: ", createNewErr)
						}
						// use gorm insert statement to update the join table
						errExec := db.Exec("insert ignore into metadata_follows (metadata_pubkey_hex, follow_pubkey_hex) values (?, ?)", person.PubkeyHex, newUser.PubkeyHex).Error
						checkAndReportGormError(errExec, []string{"1062"})
					} else {
						// follow user found,
						// update the follow user's recommend server suggestion
						if len(c) >= 3 && c[2] != "" {
							var servers []RecommendServer
							db.Find(&servers, "pubkey_hex = ? and url = ? and recommended_by = ?", followPerson.PubkeyHex, c[2], person.PubkeyHex)
							if len(servers) > 0 {
								// already recommended, update time fields?
							} else {
								// add to recommended servers
								db.Model(&followPerson).Association("Servers").Append(&RecommendServer{
									Url:           c[2],
									RecommendedBy: person.PubkeyHex,
								})
							}
						}
						// use gorm insert statement to update the join table
						errExec := db.Exec("insert ignore into metadata_follows (metadata_pubkey_hex, follow_pubkey_hex) values (?, ?)", person.PubkeyHex, followPerson.PubkeyHex).Error
						checkAndReportGormError(errExec, []string{"1062"})
					}
				}
			}
		}
	}()

	go func() {
		for cErr := range relay.ConnectionError {
			if cErr != nil {
				fmt.Printf("relay: %s connection error: %s\n", relay.URL, cErr)
				updateOrCreateRelayStatus(db, relay.URL, "connection error: "+cErr.Error())
				// attempt a re-connection
				time.Sleep(60 * time.Second)
				fmt.Printf("reconnecting to %s\n", relay.URL)
				updateOrCreateRelayStatus(db, relay.URL, "reconnecting")
				doRelay(db, ctx, relay.URL)
			}
		}
	}()
	return true
}

func main() {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Error, // Log level
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,        // Disable color
		},
	)

	dsn := "jeremy:jeremy@tcp(127.0.0.1:3306)/nono?charset=utf8mb4&parseTime=True&loc=UTC"
	db, dberr := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: newLogger})
	db.Logger.LogMode(logger.Silent)
	if dberr != nil {
		panic("failed to connect database")
	}

	migrateErr := db.AutoMigrate(&Metadata{})
	migrateErr2 := db.AutoMigrate(&RelayStatus{})
	migrateErr3 := db.AutoMigrate(&RecommendServer{})
	if migrateErr != nil || migrateErr2 != nil || migrateErr3 != nil {
		panic("one or more migrations failed, aborting")
	}

	ctx, _ := context.WithCancel(context.Background())

	// connect to relay(s)
	relayUrls := []string{
		"wss://relay.snort.social",
		"wss://relay.damus.io",
		"wss://nostr.zebedee.cloud",
		"wss://eden.nostr.land",
		"wss://nostr-pub.wellorder.net",
		"wss://nostr-dev.wellorder.net",
		"wss://relay.nostr.info",
	}

	for _, url := range relayUrls {
		doRelay(db, ctx, url)
	}

	for {
		time.Sleep(5 * time.Second)
	}

}
