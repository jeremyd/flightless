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

	generic_sync "github.com/SaveTheRbtz/generic-sync-map-go"
)

type Metadata struct {
	PubkeyHex   string `gorm:"primaryKey;size:256"`
	Name        string `gorm:"size:1024"`
	About       string `gorm:"size:4096"`
	Nip05       string `gorm:"size:512"`
	Lud06       string `gorm:"size:2048"`
	Lud16       string `gorm:"size:512"`
	Website     string `gorm:"size:512"`
	DisplayName string `gorm:"size:512"`
	Picture     string `gorm:"type:text;size:65535"`
	//UpdatedAt   time.Time   `gorm:"autoUpdateTime"`
	RecommendServer string      `gorm:"size:512"`
	Follows         []*Metadata `gorm:"many2many:metadata_follows"`
}

type Proof struct {
	ID      uint   `gorm:"primaryKey"`
	Relay   string `gorm:"size:512"`
	EventID string `gorm:"size:512"`
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
	if migrateErr != nil {
		panic("migration failed, aborting")
	}
	ctx, cancel := context.WithCancel(context.Background())

	// connect to relay(s)
	relayUrls := []string{
		"wss://relay.snort.social",
		"wss://relay.damus.io",
		"wss://nostr.zebedee.cloud",
		"wss://eden.nostr.land",
		"wss://nostr-pub.wellorder.net",
		"wss://nostr-dev.wellorder.net",
	}

	seenMessages := new(generic_sync.MapOf[string, bool])

	for _, url := range relayUrls {
		relay, err := nostr.RelayConnect(ctx, url)
		if err != nil {
			fmt.Printf("failed initial connection to relay: %s, %s; skipping relay\n", url, err)
			continue
		}

		// create filters
		filters := []nostr.Filter{{
			Kinds: []int{0, 3},
			//Tags:  t,
			// limit = 3, get the three most recent notes
			Limit: 20,
		}}

		// create a subscription and submit to relay
		// results will be returned on the sub.Events channel
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
			cancel()
			sub.Unsub()
			relay.Close()
			// give other relays time to close
			time.Sleep(5 * time.Second)
			os.Exit(0)
		}()

		go func() {
			for ev := range sub.Events {
				seen, _ := seenMessages.Load(ev.ID)
				if seen {
					fmt.Println("seen..")
					continue
				} else {
					seenMessages.Store(ev.ID, true)
				}
				// Metadata
				if ev.Kind == 0 {
					//fmt.Println(ev)
					m := Metadata{}
					err := json.Unmarshal([]byte(ev.Content), &m)
					if err != nil {
						fmt.Println(err)
					}
					//fmt.Printf("name: %s\nabout: %s\n\n", m.Name, m.About)
					m.PubkeyHex = ev.PubKey
					//m.UpdatedAt = ev.CreatedAt
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
					// recommend relay
				} else if ev.Kind == 2 {
					m := Metadata{PubkeyHex: ev.PubKey, RecommendServer: ev.Content}
					rowsUpdated := db.Model(Metadata{}).Where("pubkey_hex = ?", m.PubkeyHex).Updates(&m).RowsAffected
					if rowsUpdated == 0 {
						err := db.Save(&m).Error
						if err != nil {
							fmt.Println(err)
						}
						fmt.Printf("Created metadata for %s, %s, %s", m.Name, m.Nip05, m.RecommendServer)
					} else {
						fmt.Printf("Updated metadata for %s, %s, %s\n", m.Name, m.Nip05, m.RecommendServer)
					}
					// Contact Lists
				} else if ev.Kind == 3 {
					//fmt.Println(ev)
					//fmt.Println(ev.Tags)
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
						//db.Omit("Follows").Create(&person)
					} else {
						fmt.Printf("updating (%d) follows for %s: %s\n", len(allPTags), person.Name, person.PubkeyHex)
					}
					db.Model(&person).Association("Follows").Clear()
					db.Save(&person)
					for _, c := range allPTags {
						var followPerson Metadata
						notFoundFollow := db.First(&followPerson, "pubkey_hex = ?", c[1]).Error
						if notFoundFollow != nil {
							// follow user not found, need to create it
							newUser := Metadata{PubkeyHex: c[1]}
							//createNewErr := db.Create(&newUser).Error
							//createNewErr := db.Omit(clause.Associations).Create(&newUser).Error
							createNewErr := db.Omit("Follows").Create(&newUser).Error
							if createNewErr != nil {
								fmt.Println("Error creating user for follow: ", createNewErr)
							}
							// use gorm insert statement to update the join table
							db.Exec("insert into metadata_follows (metadata_pubkey_hex, follow_pubkey_hex) values (?, ?)", person.PubkeyHex, newUser.PubkeyHex)
						} else {
							// follow user found,
							// use gorm insert statement to update the join table
							db.Exec("insert into metadata_follows (metadata_pubkey_hex, follow_pubkey_hex) values (?, ?)", person.PubkeyHex, followPerson.PubkeyHex)
						}
					}
					/*
						updatedPerson := Metadata{PubkeyHex: ev.PubKey}
						var newFollows []Metadata
						db.Model(&updatedPerson).Association("Follows").Find(&newFollows)
						fmt.Printf("%d newFollows for %s\n", len(newFollows), updatedPerson.PubkeyHex)
					*/
				}
			}
		}()

		go func() {
			for cErr := range relay.ConnectionError {
				if cErr != nil {
					fmt.Printf("relay: %s connection error: %s\n", relay.URL, cErr)
				}
			}
		}()
	}

	for {
		time.Sleep(5 * time.Second)
	}
}
