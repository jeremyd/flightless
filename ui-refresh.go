package main

import (
	"fmt"

	"github.com/awesome-gocui/gocui"
)

func refreshAll(g *gocui.Gui, v *gocui.View) error {
	curView := g.CurrentView().Name()
	if curView == "v3" {
		refreshV3(g, v)
	} else if curView == "v4" {
		refreshRelays(g, v)
	} else { //v2
		refresh(g, v)
		refreshV3(g, v)
		refreshRelays(g, v)
	}
	return nil
}

func refresh(g *gocui.Gui, v *gocui.View) error {
	v2, _ := g.View("v2")
	_, vY := v2.Size()
	v2.Clear()
	if followSearch {
		for _, metadata := range followPages[CurrOffset:] {
			if metadata.Nip05 != "" {
				fmt.Fprintf(v, "%-30s %-30s \n", metadata.Name, metadata.Nip05)
			} else {
				fmt.Fprintf(v, "%-30s\n", metadata.Name)
			}
		}
	} else {
		if searchTerm != "" && searchTerm != "%%" {
			ViewDB.Offset(CurrOffset).Limit(vY-1).Find(&v2Meta, "name like ? or nip05 like ? or pubkey_hex like ? or pubkey_npub like ?", searchTerm, searchTerm, searchTerm, searchTerm)
		} else {
			ViewDB.Offset(CurrOffset).Limit(vY-1).Find(&v2Meta, "name != ?", "")
		}
		for _, metadata := range v2Meta {
			if metadata.Nip05 != "" {
				fmt.Fprintf(v, "%-30s %-30s \n", metadata.Name, metadata.Nip05)
			} else {
				fmt.Fprintf(v, "%-30s\n", metadata.Name)
			}
		}
	}

	v2.Highlight = true
	v2.SelBgColor = gocui.ColorCyan
	v2.SelFgColor = gocui.ColorBlack
	return nil
}

func refreshV3(g *gocui.Gui, v *gocui.View) error {
	v2, _ := g.View("v2")
	_, newCy := v2.Cursor()
	v3, _ := g.View("v3")
	v3.Clear()
	if followSearch {
		if len(followPages) > newCy+CurrOffset {
			fmt.Fprintf(v3, "%s", displayMetadataAsText(followPages[newCy+CurrOffset]))

		}
	} else {
		if len(v2Meta) > newCy {
			fmt.Fprintf(v3, "%s", displayMetadataAsText(v2Meta[newCy]))
		}
	}
	return nil
}

func refreshRelays(g *gocui.Gui, v *gocui.View) error {
	var RelayStatuses []RelayStatus
	ViewDB.Find(&RelayStatuses)
	v4, _ := g.View("v4")
	v4.Clear()
	for _, relayStatus := range RelayStatuses {
		var shortStatus string
		if relayStatus.Status == "connection established" {
			shortStatus = "⌛✅"
		} else if relayStatus.Status == "EOSE" {
			shortStatus = "✅"
		} else if relayStatus.Status == "waiting" {
			shortStatus = "⌛"
		} else {
			shortStatus = "❌"
		}

		fmt.Fprintf(v4, "%s %s\n", shortStatus, relayStatus.Url)
	}
	return nil
}

func refreshV5(g *gocui.Gui, v *gocui.View) error {
	v5, _ := g.View("v5")
	v5.Clear()
	// HELP BUTTONS
	NoticeColor := "\033[1;36m%s\033[0m"
	s := fmt.Sprintf("(%s)earch", fmt.Sprintf(NoticeColor, "s"))
	q := fmt.Sprintf("(%s)uit", fmt.Sprintf(NoticeColor, "q"))
	f := fmt.Sprintf("(%s)efresh", fmt.Sprintf(NoticeColor, "r"))
	t := fmt.Sprintf("(%s)next window", fmt.Sprintf(NoticeColor, "tab"))
	a := fmt.Sprintf("(%s)dd relay", fmt.Sprintf(NoticeColor, "a"))

	fmt.Fprintf(v5, "%-30s%-30s%-30s%-30s%-30s\n", s, q, f, t, a)
	ff := fmt.Sprintf("(%s)ollow", fmt.Sprintf(NoticeColor, "f"))
	u := fmt.Sprintf("(%s)n-follow", fmt.Sprintf(NoticeColor, "u"))
	m := fmt.Sprintf("(%s)ute", fmt.Sprintf(NoticeColor, "m"))
	fmt.Fprintf(v5, "%-30s%-30s%-30s\n\n", ff, u, m)

	var ac Account
	var mm Metadata
	ea := ViewDB.First(&ac, "active = ?", true).Error
	if ea == nil {
		em := ViewDB.First(&mm, "pubkey_hex = ?", ac.Pubkey).Error
		usename := "unknown"
		if em == nil && mm.Name != "" {
			usename = mm.Name
		}
		fmt.Fprintf(v5, "account: %s, %s\n", usename, ac.PubkeyNpub)
	} else {
		fmt.Fprintf(v5, "no account active\n")
	}
	return nil
}
