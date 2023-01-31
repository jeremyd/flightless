package main

import (
	"context"
	"encoding/json"
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
		TheLog.Printf("failed initial connection to relay: %s, %s; skipping relay", url, err)
		UpdateOrCreateRelayStatus(db, url, "failed initial connection")
		return false
	}
	nostrRelays = append(nostrRelays, relay)

	UpdateOrCreateRelayStatus(db, url, "connection established")

	pubkey, foundPub := os.LookupEnv("PUBKEY")

	var activeAccount Account
	foundAcct := true
	ferr := db.First(&activeAccount, "active = ?", true).Error
	if ferr != nil {
		TheLog.Printf("failed to find an active account: %s", ferr)
		foundAcct = false
	} else {
		pubkey = activeAccount.Pubkey
	}

	var filters []nostr.Filter
	// create filters
	if foundPub || foundAcct {
		// if the pubkey starts with npub, decode with nip19
		if pubkey[0:4] == "npub" {
			if _, v, err := nip19.Decode(pubkey); err == nil {
				pubkey = v.(string)
			}
		}

		var followers []string
		db.Table("metadata_follows").Select("metadata_pubkey_hex").Where("follow_pubkey_hex = ?", pubkey).Scan(&followers)

		person := Metadata{PubkeyHex: pubkey}
		var follows []Metadata
		db.Model(&person).Association("Follows").Find(&follows)

		var allFollow []string

		for _, f := range follows {
			allFollow = append(allFollow, f.PubkeyHex)
		}

		allFollow = append(allFollow, followers...)

		filters = []nostr.Filter{
			{
				Kinds:   []int{0, 2, 3},
				Limit:   100,
				Authors: []string{pubkey},
			},
			{
				Kinds: []int{0, 2},
				Limit: 100,
			},
			{
				Kinds: []int{3},
				Limit: 100,
			},
			{
				Kinds:   []int{0, 2},
				Limit:   len(allFollow),
				Authors: allFollow,
			},
			{
				Kinds:   []int{3},
				Limit:   len(allFollow),
				Authors: allFollow,
			},
			/*
				{
					Kinds:   []int{2},
					Limit:   100,
					Authors: allFollow,
				},
				{
					Kinds:   []int{3},
					Limit:   100,
					Authors: allFollow,
				},
				{
					Kinds: []int{0, 2, 3},
					Limit: 100,
				},
			*/
		}
	} else {
		filters = []nostr.Filter{
			{
				Kinds: []int{0, 2, 3},
				//Tags:  t,
				// limit = 3, get the three most recent notes
				Limit: 100,
			},
		}
	}

	// create a subscription and submit to relay
	sub := relay.Subscribe(ctx, filters)
	nostrSubs = append(nostrSubs, sub)

	go func() {
		<-sub.EndOfStoredEvents
		TheLog.Printf("got EOSE from %s\n", relay.URL)
		UpdateOrCreateRelayStatus(db, url, "EOSE")
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		TheLog.Println("exiting gracefully")
		sub.Unsub()
		relay.Close()
		// give other relays time to close
		time.Sleep(1 * time.Second)
		//os.Exit(0)
	}()

	go func() {
		for ev := range sub.Events {
			//TheLog.Printf("got event kind %d from %s", ev.Kind, relay.URL)
			if ev.Kind == 0 {
				// Metadata
				m := Metadata{}
				err := json.Unmarshal([]byte(ev.Content), &m)
				if err != nil {
					TheLog.Println(err)
				}
				m.PubkeyHex = ev.PubKey
				npub, errEncode := nip19.EncodePublicKey(ev.PubKey)
				if errEncode == nil {
					m.PubkeyNpub = npub
				}
				m.UpdatedAt = ev.CreatedAt
				if len(m.Picture) > 65535 {
					TheLog.Println("too big a picture for profile, skipping" + ev.PubKey)
					m.Picture = ""
					//continue
				}
				// check timestamps
				var checkMeta Metadata
				notFoundErr := db.First(&checkMeta, "pubkey_hex = ?", m.PubkeyHex).Error
				if notFoundErr != nil {
					err := db.Save(&m).Error
					if err != nil {
						TheLog.Println(err)
					}
					TheLog.Printf("Created metadata for %s, %s\n", m.Name, m.Nip05)
				} else {
					if checkMeta.UpdatedAt.After(ev.CreatedAt) {
						TheLog.Println("skipping old metadata for " + ev.PubKey)
						continue
					} else {
						rowsUpdated := db.Model(Metadata{}).Where("pubkey_hex = ?", m.PubkeyHex).Updates(&m).RowsAffected
						if rowsUpdated > 0 {
							TheLog.Printf("Updated metadata for %s, %s\n", m.Name, m.Nip05)
						}
					}
				}
			} else if ev.Kind == 2 {
				// recommend relay
				TheLog.Println("FOUND TYPE 2! for " + ev.PubKey + " with content " + ev.Content)
				var server RecommendServer
				notF := db.First(&server, "pubkey_hex = ? and recommended_by = ? and url = ?", ev.PubKey, ev.PubKey, ev.Content).Error
				if notF == nil {
					db.Model(&server).Update("url", ev.Content)
				} else {
					// add to recommended servers
					cErr := db.Create(&RecommendServer{
						PubkeyHex:     ev.PubKey,
						Url:           ev.Content,
						RecommendedBy: ev.PubKey,
					}).Error
					if cErr != nil {
						TheLog.Printf("error updating for kind2: %s", cErr)
					}
					// race condition, try again w/update?
					/*
						if cErr != nil {
							notF := db.First(&server, "pubkey_hex = ? and recommended_by = ?", ev.PubKey, ev.PubKey).Error
							if notF == nil {
								db.Model(&server).Update("url", ev.Content)
							}
						}*/
				}
			} else if ev.Kind == 3 {

				// Contact List
				pTags := []string{"p"}
				allPTags := ev.Tags.GetAll(pTags)
				var person Metadata
				notFoundError := db.First(&person, "pubkey_hex = ?", ev.PubKey).Error
				if notFoundError != nil {
					TheLog.Printf("Creating blank metadata for %s\n", ev.PubKey)
					person = Metadata{
						PubkeyHex:    ev.PubKey,
						TotalFollows: len(allPTags),
						// set time to january 1st 1970
						UpdatedAt:         time.Unix(0, 0),
						ContactsUpdatedAt: ev.CreatedAt,
					}
					db.Create(&person)
				} else {
					if person.ContactsUpdatedAt.After(ev.CreatedAt) {
						// double check the timestamp for this follow list, don't update if older than most recent
						TheLog.Printf("skipping old contact list for " + ev.PubKey)
						continue
					} else {
						db.Model(&person).Update("total_follows", len(allPTags))
						db.Model(&person).Update("contacts_updated_at", ev.CreatedAt)
						TheLog.Printf("updating (%d) follows for %s: %s\n", len(allPTags), person.Name, person.PubkeyHex)
					}
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
							TheLog.Println("Error creating user for follow: ", createNewErr)
						}
						// use gorm insert statement to update the join table
						db.Exec("insert or ignore into metadata_follows (metadata_pubkey_hex, follow_pubkey_hex) values (?, ?)", person.PubkeyHex, newUser.PubkeyHex)
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
						db.Exec("insert or ignore into metadata_follows (metadata_pubkey_hex, follow_pubkey_hex) values (?, ?)", person.PubkeyHex, followPerson.PubkeyHex)
					}
				}
			}
		}
	}()

	go func() {
		for notice := range relay.Notices {
			TheLog.Printf("relay: %s notice: %s\n", relay.URL, notice)
		}
	}()

	go func() {
		for cErr := range relay.ConnectionError {
			if cErr != nil {
				var relayStatus RelayStatus
				err := db.First(&relayStatus, "url = ?", relay.URL)
				// if we don't find the relay in our statuses, don't reconnect
				if err == nil {
					TheLog.Printf("relay: %s connection error: %s\n", relay.URL, cErr)
					UpdateOrCreateRelayStatus(db, relay.URL, "connection error: "+cErr.Error())
					// attempt a re-connection
					time.Sleep(60 * time.Second)
					TheLog.Printf("reconnecting to %s\n", relay.URL)
					UpdateOrCreateRelayStatus(db, relay.URL, "reconnecting")
					doRelay(db, ctx, relay.URL)
				}
			}
		}
	}()
	return true
}
