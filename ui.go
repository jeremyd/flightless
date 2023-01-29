package main

import (
	"errors"
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/awesome-gocui/gocui"
	tcell "github.com/gdamore/tcell/v2"
)

func keybindings(g *gocui.Gui) error {
	// q key (quit)
	if err := g.SetKeybinding("", rune(0x71), gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	// s key (search)
	if err := g.SetKeybinding("v2", rune(0x73), gocui.ModNone, search); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v2", gocui.KeyTab, gocui.ModNone, next); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v3", gocui.KeyTab, gocui.ModNone, next); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v4", gocui.KeyTab, gocui.ModNone, next); err != nil {
		log.Panicln(err)
	}
	// d key (delete)
	if err := g.SetKeybinding("v4", rune(0x64), gocui.ModNone, delRelay); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v2", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v2", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v4", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v4", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v3", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v3", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		log.Panicln(err)
	}
	// r key (refresh)
	if err := g.SetKeybinding("", rune(0x72), gocui.ModNone, refresh); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyPgup, gocui.ModNone, pageUp); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyPgdn, gocui.ModNone, pageDown); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("msg", gocui.KeyEnter, gocui.ModNone, doSearch); err != nil {
		log.Panicln(err)
	}
	// a key (add recommend relay)
	if err := g.SetKeybinding("v2", rune(0x61), gocui.ModNone, addRelay); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v2", gocui.KeyEnter, gocui.ModNone, next); err != nil {
		log.Panicln(err)
	}
	//y key
	if err := g.SetKeybinding("addrelay", rune(0x79), gocui.ModNone, doAddRelay); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("addrelay", gocui.KeyEnter, gocui.ModNone, doAddRelay); err != nil {
		log.Panicln(err)
	}
	//n key
	if err := g.SetKeybinding("addrelay", rune(0x6e), gocui.ModNone, cancelAddRelay); err != nil {
		log.Panicln(err)
	}
	return nil
}

var selectableViews = []string{"v2", "v3", "v4"}
var curView = 0

func next(g *gocui.Gui, v *gocui.View) error {
	for _, view := range selectableViews {
		v, _ := g.View(view)
		//v.FrameColor = gocui.NewRGBColor(255, 255, 255)
		v.Highlight = false
	}
	if curView == len(selectableViews)-1 {
		curView = 0
	} else {
		curView += 1
	}
	v, err := g.SetCurrentView(selectableViews[curView])
	if err != nil {
		fmt.Println("ERROR selecting view")
		return nil
	}
	if v.Name() == "v4" {
		refreshRelays(g, v)
	}
	//v.FrameColor = gocui.NewRGBColor(200, 100, 100)
	v.Highlight = true
	v.SelBgColor = gocui.ColorCyan
	v.SelFgColor = gocui.ColorBlack
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
		fmt.Fprint(v, "NoGo v0.0.1")
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
	}

	if v, err := g.SetView("v3", 0, maxY-21, maxX-20, maxY-6, 1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Details"
		v.Wrap = true
		v.Autoscroll = false
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

		fmt.Fprintf(v, "%-30s%-30s%-30s%-30s%-30s\n", s, q, f, t, a)
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
	msg, _ := g.View("msg")
	searchTerm = "%" + msg.Buffer() + "%"
	g.DeleteView("msg")
	CurrOffset = 0
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

func refresh(g *gocui.Gui, v *gocui.View) error {
	//g.SetCurrentView("v2")
	v, err := g.View("v2")
	if err != nil {
		//fmt.Println("error getting view")
	}

	_, vY := v.Size()

	if searchTerm != "" {
		ViewDB.Offset(CurrOffset).Limit(vY-1).Find(&v2Meta, "name like ? or nip05 like ?", searchTerm, searchTerm)
	} else {
		ViewDB.Offset(CurrOffset).Limit(vY-1).Find(&v2Meta, "name != ?", "")
	}
	v.Clear()
	for _, metadata := range v2Meta {
		if metadata.Nip05 != "" {
			fmt.Fprintf(v, "%-30s %-30s \n", metadata.Name, metadata.Nip05)
		} else {
			fmt.Fprintf(v, "%-30s\n", metadata.Name)
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
	CurrOffset -= vSizeY
	refresh(g, v)
	refreshV3(g, v)
	return nil
}

func pageDown(g *gocui.Gui, v *gocui.View) error {
	_, vSizeY := v.Size()
	// end of results
	if len(v2Meta) < vSizeY-1 {
		return nil
	}
	CurrOffset += vSizeY
	refresh(g, v)
	refreshV3(g, v)
	return nil
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		_, vSizeY := v.Size()
		var count int64
		if v.Name() == "v4" {
			ViewDB.Model(&RelayStatus{}).Count(&count)
			if int64(cy) >= count-1 {
				return nil
			}
		}
		// v2 pagination
		if (cy + 1) >= (vSizeY - 1) {
			// end of page
			CurrOffset += vSizeY
			refresh(g, v)
			return nil
		}
		if (cy + 1) >= len(v2Meta) {
			// end of list
			return nil
		}
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
		if v.Name() == "v2" {
			refreshV3(g, v)
		}
	}
	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		_, vSizeY := v.Size()
		if cy == 0 && v.Name() == "v4" {
			return nil
		}
		if cy == 0 {
			if CurrOffset >= vSizeY {
				CurrOffset -= vSizeY
			} else {
				CurrOffset = 0
			}
			refresh(g, v)
		} else {
			ox, oy := v.Origin()
			if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
				if err := v.SetOrigin(ox, oy-1); err != nil {
					return err
				}
			}
		}
		if v.Name() == "v2" {
			refreshV3(g, v)
		}
	}
	return nil
}

func refreshV3(g *gocui.Gui, v *gocui.View) error {
	_, newCy := v.Cursor()
	v, err := g.View("v3")
	if err != nil {
		// handle error
		//fmt.Println("error getting view")
		return nil
	}
	v.Clear()
	//v.Title = v2Meta[0].Name
	if len(v2Meta) > 0 {
		fmt.Fprintf(v, "%s", displayMetadataAsText(v2Meta[newCy]))
	}
	g.SetCurrentView("v2")
	return nil
}

func displayMetadataAsText(m Metadata) string {
	// Use GORM API build SQL
	var followersCount int64
	var followsCount int64
	ViewDB.Table("metadata_follows").Where("follow_pubkey_hex = ?", m.PubkeyHex).Count(&followersCount)
	ViewDB.Table("metadata_follows").Where("metadata_pubkey_hex = ?", m.PubkeyHex).Count(&followsCount)

	var servers []RecommendServer
	ViewDB.Model(&m).Association("Servers").Find(&servers)
	var useserver string
	if len(servers) == 0 {
		useserver = ""
	} else {
		useserver = servers[0].Url
	}

	x := fmt.Sprintf("%-20sFollowers: %4d, Follows: %4d [%4s]\n%s\npubkey: %s\nnip05: %s\nabout:\n%s\n%s\n%s\n%s\n",
		m.Name,
		followersCount,
		followsCount,
		useserver,
		m.DisplayName,
		m.PubkeyHex,
		m.Nip05,
		m.About,
		m.Website,
		m.Lud06,
		m.Lud16,
	)
	return x
}

func addRelay(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("addrelay", maxX/2-30, maxY/2, maxX/2+30, maxY/2+2, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		if _, err := g.SetCurrentView("addrelay"); err != nil {
			return err
		}
		v.Title = "Add Relay? (y/n)"
		v.Editable = false
		v.KeybindOnEdit = true
		v2, _ := g.View("v2")
		_, cy := v2.Cursor()
		curM := v2Meta[cy]
		var curServer RecommendServer
		ViewDB.Model(&curM).Association("Servers").Find(&curServer)
		if curServer.Url == "" {
			fmt.Fprintf(v, "%s", "not found")
			time.Sleep(2 * time.Second)
			g.SetCurrentView("v2")
			g.DeleteView("addrelay")
		} else {
			fmt.Fprintf(v, "%s", curServer.Url)
		}
	}
	return nil
}

func doAddRelay(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		v2, _ := g.View("v2")
		_, cy := v2.Cursor()
		curM := v2Meta[cy]
		var curServer RecommendServer
		ViewDB.Model(&curM).Association("Servers").Find(&curServer)
		err := ViewDB.Create(&RelayStatus{Url: curServer.Url, Status: "waiting"}).Error
		if err != nil {
			v.Title = "error adding relay"
		}
		g.SetCurrentView("v2")
		g.DeleteView("addrelay")
		refreshRelays(g, v)
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
		v, err := g.View("v4")
		if err != nil {
			// handle error
			//fmt.Println("error getting view")
		}
		v.Clear()
		for _, relayStatus := range RelayStatuses {
			var shortStatus string
			if relayStatus.Status == "connection established" {
				shortStatus = "✅"
			} else if relayStatus.Status == "waiting" {
				shortStatus = "⌛"
			} else {
				shortStatus = "❌"
			}

			fmt.Fprintf(v, "%s %s\n", shortStatus, relayStatus.Url)
		}
		return nil
	}
}

func cancelAddRelay(g *gocui.Gui, v *gocui.View) error {
	g.DeleteView("addrelay")
	g.SetCurrentView("v2")
	return nil
}
