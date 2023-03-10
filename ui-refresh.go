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
	var resultCount int64
	if followSearch {
		resultCount = int64(len(followPages))
		for _, metadata := range followPages[CurrOffset:] {
			if metadata.Nip05 != "" {
				fmt.Fprintf(v2, "%-30s %-30s \n", metadata.Name, metadata.Nip05)
			} else {
				fmt.Fprintf(v2, "%-30s\n", metadata.Name)
			}
		}
		v2.Title = fmt.Sprintf("%s/follows (%d)", followTarget.Name, resultCount)
	} else {
		if searchTerm != "" && searchTerm != "%%" {

			ViewDB.Model(&Metadata{}).Where("name like ? or nip05 like ? or pubkey_hex like ? or pubkey_npub like ?", searchTerm, searchTerm, searchTerm, searchTerm).Count(&resultCount)
			ViewDB.Offset(CurrOffset).Limit(vY-1).Order("updated_at desc").Find(&v2Meta, "name like ? or nip05 like ? or pubkey_hex like ? or pubkey_npub like ?", searchTerm, searchTerm, searchTerm, searchTerm)
		} else {
			ViewDB.Model(&Metadata{}).Where("name != ?", "").Count(&resultCount)
			ViewDB.Offset(CurrOffset).Limit(vY-1).Order("updated_at desc").Find(&v2Meta, "name != ?", "")
		}
		for _, metadata := range v2Meta {
			if metadata.Nip05 != "" {
				fmt.Fprintf(v2, "%-30s %-30s \n", metadata.Name, metadata.Nip05)
			} else {
				fmt.Fprintf(v2, "%-30s\n", metadata.Name)
			}
			_, cy := v2.Cursor()
			for _, h := range highlighted {
				if h == metadata.PubkeyHex {
					v2.SetHighlight(cy, true)
				} else {
					v2.SetHighlight(cy, false)
				}
			}
		}
		v2.Title = fmt.Sprintf("search: %s (%d results)", searchTerm, resultCount)
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
			shortStatus = "??????"
		} else if relayStatus.Status == "EOSE" {
			shortStatus = "???"
		} else if relayStatus.Status == "waiting" {
			shortStatus = "???"
		} else {
			shortStatus = "???"
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
	u := fmt.Sprintf("<soon>(%s)n-follow", fmt.Sprintf(NoticeColor, "u"))
	m := fmt.Sprintf("<soon>(%s)ute", fmt.Sprintf(NoticeColor, "m"))
	z := fmt.Sprintf("(%s)Select ALL", fmt.Sprintf(NoticeColor, "z"))
	d := fmt.Sprintf("(%s)elete relay", fmt.Sprintf(NoticeColor, "d"))
	c := fmt.Sprintf("(%s)onfigure keys", fmt.Sprintf(NoticeColor, "c"))
	fmt.Fprintf(v5, "%-30s%-30s%-30s%-30s%-30s%-30s\n\n", ff, u, m, z, d, c)

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
