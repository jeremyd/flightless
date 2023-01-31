package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/awesome-gocui/gocui"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func keybindings(g *gocui.Gui) error {
	/* global for all Views */
	// q key (quit)
	if err := g.SetKeybinding("", rune(0x71), gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	// s key (search)
	if err := g.SetKeybinding("", rune(0x73), gocui.ModNone, search); err != nil {
		log.Panicln(err)
	}
	// tab key (next window)
	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, next); err != nil {
		log.Panicln(err)
	}
	// r key (refresh)
	if err := g.SetKeybinding("", rune(0x72), gocui.ModNone, refreshAll); err != nil {
		log.Panicln(err)
	}
	// c key (Config)
	if err := g.SetKeybinding("", rune(0x63), gocui.ModNone, config); err != nil {
		log.Panicln(err)
	}
	// f key (Follow)
	if err := g.SetKeybinding("", rune(0x66), gocui.ModNone, follow); err != nil {
		log.Panicln(err)
	}

	/* v2 View (main) */
	// cursor
	if err := g.SetKeybinding("v2", gocui.KeyArrowDown, gocui.ModNone, cursorDownV2); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v2", gocui.KeyArrowUp, gocui.ModNone, cursorUpV2); err != nil {
		log.Panicln(err)
	}
	// vim cursor
	// j key (down)
	if err := g.SetKeybinding("v2", rune(0x6a), gocui.ModNone, cursorDownV2); err != nil {
		log.Panicln(err)
	}
	// k key (up)
	if err := g.SetKeybinding("v2", rune(0x6b), gocui.ModNone, cursorUpV2); err != nil {
		log.Panicln(err)
	}
	// pageup and pagedown
	if err := g.SetKeybinding("v2", gocui.KeyPgup, gocui.ModNone, pageUp); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v2", gocui.KeyPgdn, gocui.ModNone, pageDown); err != nil {
		log.Panicln(err)
	}
	// a key (add recommend relay)
	if err := g.SetKeybinding("v2", rune(0x61), gocui.ModNone, addRelay); err != nil {
		log.Panicln(err)
	}
	// spacebar key (select)
	if err := g.SetKeybinding("v2", gocui.KeySpace, gocui.ModNone, selectBar); err != nil {
		log.Panicln(err)
	}
	// enter key (ask)
	if err := g.SetKeybinding("v2", gocui.KeyEnter, gocui.ModNone, askExpand); err != nil {
		log.Panicln(err)
	}

	/* v4 View (Relay List) */
	// d key (delete)
	if err := g.SetKeybinding("v4", rune(0x64), gocui.ModNone, delRelay); err != nil {
		log.Panicln(err)
	}
	// cursor
	if err := g.SetKeybinding("v4", gocui.KeyArrowDown, gocui.ModNone, cursorDownV4); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v4", gocui.KeyArrowUp, gocui.ModNone, cursorUpV4); err != nil {
		log.Panicln(err)
	}
	// a key (add new relay)
	if err := g.SetKeybinding("v4", rune(0x61), gocui.ModNone, addRelay); err != nil {
		log.Panicln(err)
	}

	/* v3 view (expanded metadata) */
	// cursor
	if err := g.SetKeybinding("v3", gocui.KeyArrowDown, gocui.ModNone, cursorDownV3); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v3", gocui.KeyArrowUp, gocui.ModNone, cursorUpV3); err != nil {
		log.Panicln(err)
	}

	/* search view */
	if err := g.SetKeybinding("msg", gocui.KeyEnter, gocui.ModNone, doSearch); err != nil {
		log.Panicln(err)
	}

	/* addrelay view */
	if err := g.SetKeybinding("addrelay", gocui.KeyEnter, gocui.ModNone, doAddRelay); err != nil {
		log.Panicln(err)
	}
	//cancel key
	if err := g.SetKeybinding("addrelay", gocui.KeyEsc, gocui.ModNone, cancelAddRelay); err != nil {
		log.Panicln(err)
	}

	/* config view for accounts */
	//cancel key
	if err := g.SetKeybinding("config", gocui.KeyEsc, gocui.ModNone, cancelConfig); err != nil {
		log.Panicln(err)
	}
	// unsupported: edit
	//if err := g.SetKeybinding("config", gocui.KeyEnter, gocui.ModNone, configEdit); err != nil {
	//	log.Panicln(err)
	//}
	// n key (new config)
	if err := g.SetKeybinding("config", rune(0x6e), gocui.ModNone, configNew); err != nil {
		log.Panicln(err)
	}
	// d key (delete config)
	if err := g.SetKeybinding("config", rune(0x64), gocui.ModNone, doConfigDel); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("config", gocui.KeyArrowDown, gocui.ModNone, cursorDownConfig); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("config", gocui.KeyArrowUp, gocui.ModNone, cursorUpConfig); err != nil {
		log.Panicln(err)
	}
	/* config submenu (new/edit) */
	//cancel key
	if err := g.SetKeybinding("confignew", gocui.KeyEsc, gocui.ModNone, cancelConfigNew); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("confignew", gocui.KeyEnter, gocui.ModNone, doConfigNew); err != nil {
		log.Panicln(err)
	}

	/* follow view */
	// n key (for NO)
	if err := g.SetKeybinding("follow", rune(0x6e), gocui.ModNone, cancelFollow); err != nil {
		log.Panicln(err)
	}
	// y key for (YES)
	if err := g.SetKeybinding("follow", rune(0x79), gocui.ModNone, doFollow); err != nil {
		log.Panicln(err)
	}

	return nil
}

var selectableViews = []string{"v2", "v3", "v4"}
var curViewNum = 0

func next(g *gocui.Gui, v *gocui.View) error {
	for _, view := range selectableViews {
		t, _ := g.View(view)
		//v.FrameColor = gocui.NewRGBColor(255, 255, 255)
		t.Highlight = false
	}
	if curViewNum == len(selectableViews)-1 {
		curViewNum = 0
	} else {
		curViewNum += 1
	}
	newV, err := g.SetCurrentView(selectableViews[curViewNum])
	if err != nil {
		TheLog.Println("ERROR selecting view")
		return nil
	}
	//v.FrameColor = gocui.NewRGBColor(200, 100, 100)
	newV.Highlight = true
	newV.SelBgColor = gocui.ColorCyan
	newV.SelFgColor = gocui.ColorBlack
	return nil
}

func layout(g *gocui.Gui) error {
	//useBg := gocui.Attribute(tcell.ColorSlateBlue)

	useBg := gocui.NewRGBColor(0, 0, 200)
	useFg := gocui.Attribute(tcell.ColorWhite)
	useFrame := gocui.NewRGBColor(200, 200, 200)
	maxX, maxY := g.Size()
	if v, err := g.SetView("v1", -1, -1, maxX, 1, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = false
		v.Wrap = false
		v.Frame = false
		v.BgColor = useBg
		v.FgColor = useFg
		fmt.Fprint(v, "FlexTree Pro Gold v0.0.1")
	}

	if v, err := g.SetView("v2", 0, 1, maxX-20, maxY-20, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		//myTitle := fmt.Sprintf("%-30s %-30s \n", "Name", "Nip05")
		//v.Title = "Profiles"
		v.Wrap = false
		v.Autoscroll = false
		v.BgColor = useBg
		v.FgColor = useFg
		v.FrameColor = useFrame
		v.Editable = false
		refresh(g, v)
		g.SetCurrentView("v2")
	}

	if v, err := g.SetView("v3", 0, maxY-21, maxX-20, maxY-6, 1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Details"
		v.Wrap = true
		v.Autoscroll = true
		v.BgColor = useBg
		v.FgColor = useFg
		v.FrameColor = useFrame
		v.Editable = false
		refreshV3(g, v)
	}

	if v, err := g.SetView("v4", maxX-29, 1, maxX-1, maxY-6, 4); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Relays"
		v.Editable = false
		v.Wrap = false
		v.Autoscroll = true
		v.BgColor = useBg
		v.FgColor = useFg
		v.FrameColor = useFrame
		refreshRelays(g, v)
	}

	if v, err := g.SetView("v5", 0, maxY-6, maxX-1, maxY-1, 1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Keybinds"
		v.Editable = false
		v.Frame = true
		v.BgColor = useBg
		v.FgColor = useFg
		v.FrameColor = useFrame
		// HELP BUTTONS
		NoticeColor := "\033[1;36m%s\033[0m"
		s := fmt.Sprintf("(%s)earch", fmt.Sprintf(NoticeColor, "s"))
		q := fmt.Sprintf("(%s)uit", fmt.Sprintf(NoticeColor, "q"))
		f := fmt.Sprintf("(%s)efresh", fmt.Sprintf(NoticeColor, "r"))
		t := fmt.Sprintf("(%s)next window", fmt.Sprintf(NoticeColor, "tab"))
		a := fmt.Sprintf("(%s)dd relay", fmt.Sprintf(NoticeColor, "a"))

		fmt.Fprintf(v, "%-30s%-30s%-30s%-30s%-30s", s, q, f, t, a)
		ff := fmt.Sprintf("(%s)ollow", fmt.Sprintf(NoticeColor, "f"))
		u := fmt.Sprintf("(%s)n-follow", fmt.Sprintf(NoticeColor, "u"))
		m := fmt.Sprintf("(%s)ute", fmt.Sprintf(NoticeColor, "m"))
		fmt.Fprintf(v, "%-30s%-30s%-30s\n", ff, u, m)
	}

	return nil
}

func search(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("msg", maxX/2-30, maxY/2, maxX/2+30, maxY/2+2, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		if _, err := g.SetCurrentView("msg"); err != nil {
			return err
		}
		v.Title = "Search"
		v.Editable = true
		v.KeybindOnEdit = true
	}
	return nil
}

func doSearch(g *gocui.Gui, v *gocui.View) error {
	followSearch = false
	CurrOffset = 0
	msg, eM := g.View("msg")
	if eM != nil {
		return nil
	}
	searchTerm = "%" + msg.Buffer() + "%"
	g.DeleteView("msg")
	g.SetCurrentView("v2")
	v2, _ := g.View("v2")
	v2.Title = "Search: " + msg.Buffer()
	refresh(g, v)
	refreshV3(g, v)
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	// give the relays time to close their connections
	// kick off a UI update while they do this
	go func() {
		g.Update(func(g *gocui.Gui) error {
			v, err := g.View("v5")
			if err != nil {
				//fmt.Println("error getting view")
			}
			v.Clear()
			fmt.Fprintln(v, "Closing connections and exiting..")
			return nil
		})
		time.Sleep(time.Second * 2)

		g.Update(func(g *gocui.Gui) error {
			return gocui.ErrQuit
		})
	}()
	return nil
}

var v2Meta []Metadata
var searchTerm = ""
var followSearch = false

func refresh(g *gocui.Gui, v *gocui.View) error {
	//g.SetCurrentView("v2")
	v, err := g.View("v2")
	if err != nil {
		//fmt.Println("error getting view")
	}

	_, vY := v.Size()

	v.Clear()
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
			ViewDB.Offset(CurrOffset).Limit(vY-1).Find(&v2Meta, "name like ? or nip05 like ? or pubkey_hex like ?", searchTerm, searchTerm, searchTerm)
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

	v.Highlight = true
	v.SelBgColor = gocui.ColorCyan
	v.SelFgColor = gocui.ColorBlack
	return nil
}

var CurrOffset = 0

func pageUp(g *gocui.Gui, v *gocui.View) error {
	_, vSizeY := v.Size()
	if CurrOffset <= 0 {
		CurrOffset = 0
		return nil
	}
	CurrOffset -= (vSizeY - 1)
	refresh(g, v)
	refreshV3(g, v)
	return nil
}

func pageDown(g *gocui.Gui, v *gocui.View) error {
	_, vSizeY := v.Size()
	// end of results
	if !followSearch && len(v2Meta) < vSizeY-1 {
		return nil
	}
	if followSearch && len(followPages) <= CurrOffset+vSizeY-1 {
		return nil
	}
	CurrOffset += (vSizeY - 1)
	refresh(g, v)
	refreshV3(g, v)
	return nil
}
func cursorDownV2(g *gocui.Gui, v *gocui.View) error {
	if v != nil {

		cx, cy := v.Cursor()
		_, vSizeY := v.Size()

		TheLog.Printf("len was %d, curr offset %d, vSizeY %d, cy %d", len(followPages), CurrOffset, vSizeY, cy)
		if followSearch && (cy+CurrOffset+1) >= len(followPages) {
			// end of list
			return nil
		}

		TheLog.Printf("len was %d, curr offset %d, vSizeY %d, cy %d", len(v2Meta), CurrOffset, vSizeY, cy)
		if !followSearch && len(v2Meta) != vSizeY-1 && (cy+1) >= len(v2Meta) {
			// end of list
			return nil
		}

		if (cy + 1) >= (vSizeY - 1) {
			// end of page / next page
			if err := v.SetCursor(0, 0); err != nil {
				if err := v.SetOrigin(0, 0); err != nil {
					return err
				}
			}

			TheLog.Println("len was", len(followPages), "curr offset", CurrOffset)

			CurrOffset += (vSizeY - 1)
			refresh(g, v)
			refreshV3(g, v)
			return nil
		}

		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
		refreshV3(g, v)
	}
	return nil
}

func cursorDownV3(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		//if (cy + 1) >= (vSizeY - 1) {
		//	return nil
		//}
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func cursorDownV4(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		var count int64
		ViewDB.Model(&RelayStatus{}).Count(&count)
		if int64(cy) >= count-1 {
			return nil
		}
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
	}
	return nil
}
func cursorUpV2(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		_, vSizeY := v.Size()
		if cy == 0 && CurrOffset == 0 {
			return nil
		}
		// page up
		if cy == 0 {
			if CurrOffset >= (vSizeY - 1) {
				CurrOffset -= (vSizeY - 1)
			} else {
				CurrOffset = 0
			}
			refresh(g, v)
			ox, oy := v.Origin()
			if err := v.SetCursor(cx, vSizeY-2); err != nil && oy > 0 {
				if err := v.SetOrigin(ox, oy-1); err != nil {
					return err
				}
			}
			// just up
		} else {
			ox, oy := v.Origin()
			if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
				if err := v.SetOrigin(ox, oy-1); err != nil {
					return err
				}
			}
		}
		refreshV3(g, v)
	}
	return nil
}

func cursorUpV3(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		if cy == 0 {
			return nil
		}
		ox, oy := v.Origin()
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
	return nil
}

func cursorUpV4(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		if cy == 0 && v.Name() == "v4" {
			return nil
		}
		ox, oy := v.Origin()
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
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

func displayMetadataAsText(m Metadata) string {
	// Use GORM API build SQL
	var followersCount int64
	var followsCount int64
	ViewDB.Table("metadata_follows").Where("follow_pubkey_hex = ?", m.PubkeyHex).Count(&followersCount)
	ViewDB.Table("metadata_follows").Where("metadata_pubkey_hex = ?", m.PubkeyHex).Count(&followsCount)

	var servers []RecommendServer
	ViewDB.Model(&m).Association("Servers").Find(&servers, "recommended_by = ?", m.PubkeyHex)
	var useserver string
	if len(servers) == 0 {
		useserver = ""
	} else {
		useserver = servers[0].Url
	}

	x := fmt.Sprintf("%-20sFollowers: %4d, Follows: %4d [%4s]\ndisplay_name: %20s\npubkey: %20s\nnip05: %20s\nwebsite: %20s\nPicture: %20s\nlud06: %20s\nlud16: %20s\n\nabout:\n%s\n",
		m.Name,
		followersCount,
		followsCount,
		useserver,
		m.DisplayName,
		m.PubkeyHex,
		m.Nip05,
		m.Website,
		m.Picture,
		m.Lud06,
		m.Lud16,
		m.About,
	)
	return x
}

func addRelay(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := g.Size()
	prevViewName := v.Name()
	if v, err := g.SetView("addrelay", maxX/2-30, maxY/2, maxX/2+30, maxY/2+2, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		if _, err := g.SetCurrentView("addrelay"); err != nil {
			return err
		}
		v.Title = "Add Relay? [enter] to save / [ESC] to cancel"
		v.Editable = true
		v.KeybindOnEdit = true
		v2, _ := g.View("v2")
		_, cy := v2.Cursor()
		if prevViewName == "v2" {
			curM := v2Meta[cy]
			var curServer RecommendServer
			ViewDB.Model(&curM).Association("Servers").Find(&curServer, "recommended_by = ?", curM.PubkeyHex)
			if curServer.Url == "" {
				/*
					g.SetCurrentView("v2")
					g.DeleteView("addrelay")
				*/
			} else {
				fmt.Fprintf(v, "%s", curServer.Url)
			}
		}
	}
	return nil
}

func doAddRelay(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		line := v.Buffer()
		if line == "" {
			g.SetCurrentView("v2")
			g.DeleteView("addrelay")
			refreshRelays(g, v)
			return nil
		}
		err := ViewDB.Create(&RelayStatus{Url: line, Status: "waiting"}).Error
		if err != nil {
			TheLog.Println("error adding relay")
		}
		g.DeleteView("addrelay")
		refreshRelays(g, v)
		g.SetCurrentView("v2")
	}
	return nil
}

func delRelay(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		_, cy := v.Cursor()
		var relayStatuses []RelayStatus
		ViewDB.Find(&relayStatuses)
		if len(relayStatuses) >= cy {
			err := ViewDB.Model(&relayStatuses[cy]).Update("status", "deleting").Error
			if err != nil {
				//fmt.Println("error deleting relay")
				//fmt.Println(err)
			} else {
				time.Sleep(1 * time.Second)
				refreshRelays(g, v)
			}
		}
	}
	return nil
}

func refreshRelays(g *gocui.Gui, v *gocui.View) error {
	for {
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
}

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

func cancelAddRelay(g *gocui.Gui, v *gocui.View) error {
	g.DeleteView("addrelay")
	g.SetCurrentView("v2")
	return nil
}

func config(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := g.Size()
	accounts := []Account{}
	aerr := ViewDB.Find(&accounts).Error
	if aerr != nil {
		TheLog.Printf("error getting accounts: %s", aerr)
	}
	if v, err := g.SetView("config", maxX/2-50, maxY/2-len(accounts), maxX/2+50, maxY/2+1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}

		theKey := ""
		for _, acct := range accounts {
			theKey = Decrypt(string(Password), acct.Privatekey)
			if len(theKey) != 64 {
				fmt.Fprintf(v, "invalid key.. delete please: %s", theKey)
			} else {
				fmt.Fprintf(v, "[%s ... ] for %s\n", theKey[0:5], acct.PubkeyNpub)
				// full priv key printing
				//fmt.Fprintf(v, "[%s] for %s\n", theKey, acct.Pubkey)
			}
		}

		v.Title = "Config Private Keys - [ESC]Cancel - [n]ew - [d]elete -"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		v.Editable = false
		v.KeybindOnEdit = true
		if _, err := g.SetCurrentView("config"); err != nil {
			TheLog.Println("error setting current view to config")
			return nil
		}
	}
	return nil
}

func configNew(
	g *gocui.Gui,
	v *gocui.View,
) error {
	maxX, maxY := g.Size()
	g.DeleteView("config")
	if v, err := g.SetView("confignew", maxX/2-50, maxY/2-1, maxX/2+50, maxY/2+1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}

		v.Title = "New/Edit Private Key - [Enter]Save - [ESC]Cancel -"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		v.Editable = true
		v.KeybindOnEdit = true
		if _, err := g.SetCurrentView("confignew"); err != nil {
			return err
		}
	}
	return nil
}

func configEdit(
	g *gocui.Gui,
	v *gocui.View,
) error {
	maxX, maxY := g.Size()
	cView, _ := g.View("config")
	_, cy := cView.Cursor()
	accounts := []Account{}
	aerr := ViewDB.Find(&accounts).Error
	if aerr != nil {
		TheLog.Printf("error getting accounts: %s", aerr)
	}
	editThis := Decrypt(string(Password), accounts[cy].Privatekey)
	g.DeleteView("config")
	if v, err := g.SetView("confignew", maxX/2-50, maxY/2-1, maxX/2+50, maxY/2+1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}

		v.Title = "New/Edit Private Key - [Enter]Save - [ESC]Cancel -"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		v.Editable = true
		v.KeybindOnEdit = true
		if _, err := g.SetCurrentView("confignew"); err != nil {
			return err
		}
		fmt.Fprint(v, editThis)
	}
	return nil
}

func doConfigNew(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		line := v.Buffer()
		if line == "" {
			TheLog.Println("no private key entered")
			g.SetCurrentView("v2")
			g.DeleteView("confignew")
			return nil
		}
		//fmt.Println(line)
		//fmt.Println("saving config")
		encKey := Encrypt(string(Password), line)
		pk, ep := nostr.GetPublicKey(line)
		npub, ep2 := nip19.EncodePublicKey(pk)
		if ep != nil || ep2 != nil {
			TheLog.Printf("error getting public key: %s", ep)
		}
		account := Account{Privatekey: encKey, Pubkey: pk, PubkeyNpub: npub}
		e2 := ViewDB.Save(&account).Error
		if e2 != nil {
			TheLog.Printf("error saving private key: %s", e2)
		}

		g.SetCurrentView("v2")
		g.DeleteView("confignew")
		//refresh(g, v)
	}
	return nil
}

func cancelConfig(g *gocui.Gui, v *gocui.View) error {
	g.DeleteView("config")
	g.SetCurrentView("v2")
	return nil
}

func doConfigDel(g *gocui.Gui, v *gocui.View) error {
	cView, _ := g.View("config")
	_, cy := cView.Cursor()
	accounts := []Account{}
	aerr := ViewDB.Find(&accounts).Error
	if aerr != nil {
		TheLog.Printf("error getting accounts: %s", aerr)
	}
	if v != nil {
		line := v.Buffer()
		if line == "" {
			g.SetCurrentView("v2")
			g.DeleteView("config")
			return nil
		}
		e2 := ViewDB.Delete(&accounts[cy]).Error
		if e2 != nil {
			TheLog.Printf("error deleting private key: %s", e2)
		}

		g.DeleteView("config")
	}
	return nil
}

func cursorDownConfig(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		_, cy := v.Cursor()
		_, sy := v.Size()
		// end of view
		if cy >= sy-1 {
			return nil
		}
		// move cursor and origin
		if err := v.SetCursor(0, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func cursorUpConfig(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		_, cy := v.Cursor()
		// top of view
		if cy == 0 {
			return nil
		}
		// move cursor and origin
		if err := v.SetCursor(0, cy-1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
	return nil
}

func cancelConfigNew(g *gocui.Gui, v *gocui.View) error {
	g.DeleteView("confignew")
	g.SetCurrentView("v2")
	return nil
}

// show info for the metadata that the account matches
// show proposed follower changes
// accept input for y/n to confirm
func follow(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := g.Size()

	// use account 0 for now
	accounts := []Account{}
	aerr := ViewDB.Find(&accounts).Error
	if aerr != nil {
		TheLog.Printf("error getting accounts: %s", aerr)
	}

	cView, _ := g.View("v2")
	_, cy := cView.Cursor()

	var m Metadata
	if !followSearch {
		if len(v2Meta) > 0 && len(v2Meta) >= cy {
			// do nothing?
		} else {
			return nil
		}
		m = v2Meta[cy]
	} else {
		if len(followPages) <= cy+CurrOffset {
			return nil
		}
		m = followPages[cy+CurrOffset]
	}

	var numFollows int64
	ViewDB.Table("metadata_follows").Where("metadata_pubkey_hex = ?", accounts[0].Pubkey).Count(&numFollows)

	//lenWindow := len(highlighted) + 2
	if v, err := g.SetView("follow", maxX/2-50, maxY/2-5, maxX/2+50, maxY/2+2, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "Follow - (y)es - (n)o - (esc) cancel"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		v.Editable = false
		v.KeybindOnEdit = true
		fmt.Fprintf(v, "you have %d existing follows in our known data, check safu?\n\n", numFollows)
		fmt.Fprintf(v, "follow %s %s %s?\n", m.Name, m.Nip05, m.PubkeyHex)
		if len(highlighted) > 0 {
			fmt.Fprintf(v, "+bulk follow: selected additional %d highlighted follows\n", len(highlighted))
			/*
				for _, hi := range highlighted {
					fmt.Fprintf(v, "%s\n", hi)
				}
			*/
		}
		if _, err := g.SetCurrentView("follow"); err != nil {
			return err
		}
	}
	return nil

}

func doFollow(g *gocui.Gui, v *gocui.View) error {
	cView, _ := g.View("v2")
	_, cy := cView.Cursor()

	var m Metadata
	if followSearch {
		m = followPages[cy+CurrOffset]
	} else {
		m = v2Meta[cy]
	}

	// use account 0 for now
	accounts := []Account{}
	aerr := ViewDB.Find(&accounts).Error
	if aerr != nil {
		TheLog.Printf("error getting accounts: %s", aerr)
	}

	// get list of current follows
	var me Metadata
	ViewDB.First(&me, "pubkey_hex = ?", accounts[0].Pubkey)
	var curFollows []Metadata
	ViewDB.Model(&me).Association("Follows").Find(&curFollows)

	TheLog.Println("current follows", len(curFollows))

	var tags nostr.Tags
	// todo: set the relay nicely!
	for _, follow := range curFollows {
		tag := nostr.Tag{"p", follow.PubkeyHex}
		tags = append(tags, tag)
	}

	newtag := nostr.Tag{"p", m.PubkeyHex}
	tags = append(tags, newtag)

	for _, hi := range highlighted {
		newtag := nostr.Tag{"p", hi}
		tags = append(tags, newtag)
	}

	ev := nostr.Event{
		PubKey:    accounts[0].Pubkey,
		CreatedAt: time.Now(),
		Kind:      nostr.KindContactList,
		Tags:      tags,
		Content:   "",
	}

	// calling Sign sets the event ID field and the event Sig field
	ev.Sign(Decrypt(string(Password), accounts[0].Privatekey))
	// create context with deadline and cancel
	go func() {
		ctx, cancel := context.WithTimeout(CTX, 10*time.Second)
		defer cancel()
		TheLog.Println("sending event", ev)
		for _, r := range nostrRelays {
			result := r.Publish(CTX, ev)
			TheLog.Println(result)
			TheLog.Printf("published to %v", r.Publish(ctx, ev))
		}
	}()

	highlighted = []string{}
	g.SetCurrentView("v2")
	g.DeleteView("follow")

	return nil
}

func cancelFollow(g *gocui.Gui, v *gocui.View) error {
	g.SetCurrentView("v2")
	g.DeleteView("follow")
	return nil
}

var highlighted []string

// replace the highlighted slice with the last element and return smaller slice
func removeFromHighlight(s []string, i int) []string {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

// highlight selected line at the cursor
func selectBar(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		_, cy := v.Cursor()
		foundHighlight := false

		if followSearch {
			for i, h := range highlighted {
				if h == followPages[cy+CurrOffset].PubkeyHex {
					foundHighlight = true
					highlighted = removeFromHighlight(highlighted, i)
					v.SetHighlight(cy+CurrOffset, false)
				}
			}
			if !foundHighlight {
				v.SetHighlight(cy, true)
				highlighted = append(highlighted, followPages[cy+CurrOffset].PubkeyHex)
			}
		} else {

			for i, h := range highlighted {
				if h == v2Meta[cy].PubkeyHex {
					foundHighlight = true
					highlighted = removeFromHighlight(highlighted, i)
					v.SetHighlight(cy, false)
				}
			}
			if !foundHighlight {
				v.SetHighlight(cy, true)
				highlighted = append(highlighted, v2Meta[cy].PubkeyHex)
			}
		}
	}
	return nil
}

var followPages []Metadata
var followTarget Metadata

func askExpand(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		v2, _ := g.View("v2")
		_, cy := v2.Cursor()
		if followSearch {
			followTarget := followPages[cy+CurrOffset]
			CurrOffset = 0
			ViewDB.Model(&followTarget).Offset(CurrOffset).Association("Follows").Find(&followPages)
			TheLog.Println("len was", len(followPages), "curr offset", CurrOffset)
			if len(followPages) > 0 {
				TheLog.Println("current follows", len(followPages))
				v2.Title = fmt.Sprintf("%s/follows", followTarget.Name)
				refresh(g, v2)
				refreshV3(g, v2)
			} else {
				followSearch = false
			}
		} else {
			// reload view v2 with v2Meta loaded with follows to start
			CurrOffset = 0
			target := v2Meta[cy]
			ViewDB.Model(&target).Offset(CurrOffset).Association("Follows").Find(&followPages)
			TheLog.Println("current follows", len(followPages))
			v2.Title = fmt.Sprintf("%s/follows", target.Name)
			followSearch = true
			refresh(g, v2)
			refreshV3(g, v2)
		}
	}
	return nil
}
