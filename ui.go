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
	//if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
	//	log.Panicln(err)
	//}

	// q key (quit)
	if err := g.SetKeybinding("", rune(0x71), gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	// s key (search)
	if err := g.SetKeybinding("v2", rune(0x73), gocui.ModNone, search); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, next); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v2", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("v2", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyF5, gocui.ModNone, refresh); err != nil {
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
	return nil
}

func next(g *gocui.Gui, v *gocui.View) error {
	if g.CurrentView().Name() == "v2" {
		g.SetCurrentView("v3")
		g.Cursor = true
	} else {
		g.Cursor = false
		g.SetCurrentView("v2")
	}
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
		v.Editable = true
		v.Cursor()
		refreshV3(g, v)
	}

	if v, err := g.SetView("v4", maxX-29, 1, maxX-1, maxY-6, 4); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Status"
		v.Editable = false
		v.Wrap = true
		v.Autoscroll = false
		v.BgColor = useBg
		v.FgColor = useFg
		v.FrameColor = useFrame
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
		fmt.Fprint(v, "(s)earch\n")
		fmt.Fprint(v, "(q)uit\n")
		fmt.Fprint(v, "(F5) refresh\n")
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
		time.Sleep(time.Second * 4)

		g.Update(func(g *gocui.Gui) error {
			return gocui.ErrQuit
		})
	}()
	return nil
}

var v2Meta []Metadata
var searchTerm = ""

func refresh(g *gocui.Gui, v *gocui.View) error {
	g.SetCurrentView("v2")
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
		refreshV3(g, v)
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
	x := fmt.Sprintf("%-20sFollowers: %7d, Follows: %7d\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n",
		m.Name,
		followersCount,
		followsCount,
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

func cursorUp(g *gocui.Gui, v *gocui.View) error {
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
		refreshV3(g, v)
	}
	return nil
}
