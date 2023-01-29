package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"gorm.io/gorm"
)

var nostrSubs []*nostr.Subscription
var nostrRelays []*nostr.Relay

func doRelay(db *gorm.DB, ctx context.Context, url string) bool {
	relay, err := nostr.RelayConnect(ctx, url)
	if err != nil {
		//fmt.Printf("failed initial connection to relay: %s, %s; skipping relay\n", url, err)
		UpdateOrCreateRelayStatus(db, url, "failed initial connection")
		return false
	}
	nostrRelays = append(nostrRelays, relay)

	UpdateOrCreateRelayStatus(db, url, "connection established")

	pubkey, foundPub := os.LookupEnv("PUBKEY")
	var filters []nostr.Filter
	// create filters
	if foundPub {
		// if the pubkey starts with npub, decode with nip19
		if pubkey[0:4] == "npub" {
			if _, v, err := nip19.Decode(pubkey); err == nil {
				pubkey = v.(string)
			}
		}
		filters = []nostr.Filter{{
			Kinds: []int{0, 2, 3},
			//Tags:  t,
			// limit = 3, get the three most recent notes
			Limit: 10,
		},
			{Kinds: []int{0, 2, 3},
				Authors: []string{pubkey},
				Limit:   10,
			}}
	} else {
		filters = []nostr.Filter{{
			Kinds: []int{0, 2, 3},
			//Tags:  t,
			// limit = 3, get the three most recent notes
			Limit: 10,
		}}
	}

	// create a subscription and submit to relay
	sub := relay.Subscribe(ctx, filters)
	nostrSubs = append(nostrSubs, sub)

	go func() {
		<-sub.EndOfStoredEvents
		//fmt.Printf("got EOSE from %s\n", relay.URL)
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		//fmt.Println("exiting gracefully")
		sub.Unsub()
		relay.Close()
		// give other relays time to close
		time.Sleep(1 * time.Second)
		//os.Exit(0)
	}()

	go func() {
		for ev := range sub.Events {
			if ev.Kind == 0 {
				// Metadata
				m := Metadata{}
				err := json.Unmarshal([]byte(ev.Content), &m)
				if err != nil {
					//fmt.Println(err)
				}
				m.PubkeyHex = ev.PubKey
				if len(m.Picture) > 65535 {
					//fmt.Println("dumbass put too big a picture, skipping")
					continue
				}
				rowsUpdated := db.Model(Metadata{}).Where("pubkey_hex = ?", m.PubkeyHex).Updates(&m).RowsAffected
				if rowsUpdated == 0 {
					err := db.Save(&m).Error
					if err != nil {
						//fmt.Println(err)
					}
					//fmt.Printf("Created metadata for %s, %s\n", m.Name, m.Nip05)
				} else {
					//fmt.Printf("Updated metadata for %s, %s\n", m.Name, m.Nip05)
				}
			} else if ev.Kind == 2 {
				// recommend relay
				//fmt.Println("FOUND TYPE 2!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
				var server RecommendServer
				notF := db.First(&server, "pubkey_hex = ? and recommended_by = ?", ev.PubKey, ev.PubKey).Error
				if notF == nil {
					db.Model(&server).Update("url", ev.Content)
				} else {
					// add to recommended servers
					cErr := db.Create(&RecommendServer{
						PubkeyHex:     ev.PubKey,
						Url:           ev.Content,
						RecommendedBy: ev.PubKey,
					}).Error
					// race condition, try again w/update?
					if cErr != nil {
						notF := db.First(&server, "pubkey_hex = ? and recommended_by = ?", ev.PubKey, ev.PubKey).Error
						if notF == nil {
							db.Model(&server).Update("url", ev.Content)
						}
					}
				}
			} else if ev.Kind == 3 {
				// Contact List
				pTags := []string{"p"}
				allPTags := ev.Tags.GetAll(pTags)
				var person Metadata
				notFoundError := db.First(&person, "pubkey_hex = ?", ev.PubKey).Error
				if notFoundError != nil {
					//fmt.Printf("Creating blank metadata for %s\n", ev.PubKey)
					person = Metadata{
						PubkeyHex:    ev.PubKey,
						TotalFollows: len(allPTags),
					}
					db.Create(&person)
				} else {
					db.Model(&person).Update("total_follows", len(allPTags))
					//fmt.Printf("updating (%d) follows for %s: %s\n", len(allPTags), person.Name, person.PubkeyHex)
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
							//fmt.Println("Error creating user for follow: ", createNewErr)
						}
						// use gorm insert statement to update the join table
						errExec := db.Exec("insert or ignore into metadata_follows (metadata_pubkey_hex, follow_pubkey_hex) values (?, ?)", person.PubkeyHex, newUser.PubkeyHex).Error
						CheckAndReportGormError(errExec, []string{"1062"})
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
						errExec := db.Exec("insert or ignore into metadata_follows (metadata_pubkey_hex, follow_pubkey_hex) values (?, ?)", person.PubkeyHex, followPerson.PubkeyHex).Error
						CheckAndReportGormError(errExec, []string{"1062"})
					}
				}
			}
		}
	}()

	go func() {
		for cErr := range relay.ConnectionError {
			if cErr != nil {
				var relayStatus RelayStatus
				err := db.First(&relayStatus, "url = ?", relay.URL)
				// if we don't find the relay in our statuses, don't reconnect
				if err == nil {
					//fmt.Printf("relay: %s connection error: %s\n", relay.URL, cErr)
					UpdateOrCreateRelayStatus(db, relay.URL, "connection error: "+cErr.Error())
					// attempt a re-connection
					time.Sleep(60 * time.Second)
					fmt.Printf("reconnecting to %s\n", relay.URL)
					UpdateOrCreateRelayStatus(db, relay.URL, "reconnecting")
					doRelay(db, ctx, relay.URL)
				}
			}
		}
	}()
	return true
}
