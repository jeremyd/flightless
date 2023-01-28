package main

import (
	"context"
	"log"

	"github.com/awesome-gocui/gocui"
)

func main() {
	GetGormConnection()

	migrateErr := DB.AutoMigrate(&Metadata{})
	migrateErr2 := DB.AutoMigrate(&RelayStatus{})
	migrateErr3 := DB.AutoMigrate(&RecommendServer{})
	if migrateErr != nil || migrateErr2 != nil || migrateErr3 != nil {
		panic("one or more migrations failed, aborting")
	}

	ctx, _ := context.WithCancel(context.Background())

	// connect to relay(s)
	DB.Exec("delete from relay_statuses")
	relayUrls := []string{
		//"wss://relay.snort.social",
		//"wss://relay.damus.io",
		//"wss://nostr.zebedee.cloud",
		//"wss://eden.nostr.land",
		//"wss://nostr-pub.wellorder.net",
		"wss://nostr-dev.wellorder.net",
		//"wss://relay.nostr.info",
	}

	for _, url := range relayUrls {
		doRelay(ctx, url)
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

	// relay status messages
	/*
		go func() {
			for {
				var RelayStatuses []RelayStatus
				DB.Find(&RelayStatuses)
				g.Update(func(g *gocui.Gui) error {
					v, err := g.View("v4")
					if err != nil {
						// handle error
						fmt.Println("error getting view")
					}
					v.Clear()
					for _, relayStatus := range RelayStatuses {

						var shortStatus string
						if relayStatus.Status == "connection established" {
							shortStatus = "✅"
						} else {
							shortStatus = "❌"
						}

						fmt.Fprintf(v, "%s %s\n", shortStatus, relayStatus.Url)
					}
					return nil
				})
				time.Sleep(5 * time.Second)
			}
		}()
	*/

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}
