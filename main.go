package main

import (
	"flag"
	"github.com/godbus/dbus"
	"log"
	"time"
)

var defaultName = "org.mpris.MediaPlayer2.fake"
var objectPath = dbus.ObjectPath("/org/mpris/MediaPlayer2")
var objectInterface = "org.mpris.MediaPlayer2.Player"

type Player struct {
	referenceTime time.Time
	duration      time.Duration
	position      time.Duration
	refresh       chan bool
}

func (p *Player) SetPosition(path dbus.ObjectPath, position int64) (int64, *dbus.Error) {
	p.position = time.Duration(position) * 1000
	log.Printf("SetPosition %v\n", p.position)
	p.refresh <- true
	return position, nil
}

func (p *Player) Get(iface string, property string) (int64, *dbus.Error) {
	if property == "Position" {
		return p.getCurrentPosition(), nil
	}
	return 0, dbus.NewError("property not found", nil)
}

func (p *Player) getCurrentPosition() int64 {
	return int64((p.position + time.Since(p.referenceTime)) / 1000)
}

func main() {
	name := flag.String("name", defaultName, "dbus name")
	duration := flag.Int("duration", 3*60, "duration (in seconds)")
	position := flag.Int("position", 0, "duration (in seconds)")
	flag.Parse()

	conn, err := dbus.SessionBus()
	if err != nil {
		panic(err)
	}
	reply, err := conn.RequestName(*name, dbus.NameFlagDoNotQueue)
	if err != nil {
		panic(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Fatalf("Name already taken")
	}
	player := Player{
		duration:      time.Duration(*duration) * time.Second,
		referenceTime: time.Now(),
		position:      time.Duration(*position) * time.Second,
		refresh:       make(chan bool),
	}
	err = conn.Export(&player, objectPath, objectInterface)
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = conn.Export(&player, objectPath, "org.freedesktop.DBus.Properties")
	if err != nil {
		log.Fatalf("%v", err)
	}

outer:
	for {
		if player.position >= player.duration {
			log.Printf("Song finished (1)")
			break
		}
		player.referenceTime = time.Now()
		remaining := player.duration - player.position
		log.Printf("Waiting for event (or end of song in %v)\n", remaining)
		select {
		case <-player.refresh:
			log.Printf("Refresh")
			break
		case <-time.After(remaining):
			log.Printf("Song finished (2)")
			break outer
		}
	}
}
