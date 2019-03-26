// vim:ts=4:sw=4
// i3 - IRC bot to greet people and tell them to wait
// © 2012 Michael Stapelberg (see also: LICENSE)
package main

import (
	"flag"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	irc "github.com/fluffle/goirc/client"

	"github.com/stapelberg/greetbot/histogram"
)

var (
	ircChannel = flag.String(
		"channel",
		"#i3-build",
		"In which channel this bot should be in")

	ircPassword = flag.String(
		"password",
		"",
		"Server password, if required. For FreeNode, can be <account>:<password> to identify to services (https://freenode.net/kb/answer/registration)")

	greetings = flag.String(
		"greetings",
		"hello,hallo,hey,hi,yo,good morning,ohai",
		"Greeting words on which the bot should react, comma-separated")

	greetings_re []*regexp.Regexp

	histogram_path = flag.String(
		"histogram_path",
		"./histogram.data",
		"Serialized protobuf file containing the histogram data")

	my_histogram histogram.Histogram

	lastGreetingTime = time.Now()
)

// Returns true if any of the greeting words of the -greetings flag is
// contained in the given message.
func containsGreeting(msg string) bool {
	lcMsg := []byte(strings.ToLower(msg))
	for _, re := range greetings_re {
		if re.Match(lcMsg) {
			log.Printf("Message %s contains greeting %s\n", msg, re)
			return true
		}
	}
	log.Printf("Message %s does NOT contain a greeting\n", msg)
	return false
}

func isBot(nickname string) bool {
	// TODO
	return false
}

// Called when somebody writes some message to the IRC channel.
func handleMessage(conn *irc.Conn, line *irc.Line) {
	msg := line.Args[1]
	if line.Args[0] != *ircChannel {
		log.Printf(`Ignoring private message to me: "%s"`, msg)
		return
	}

	log.Printf("Received: *%s* says: \n", line.Nick)

	// Ignore messages from bots.
	if isBot(line.Nick) {
		log.Printf("Ignoring message, %s is a bot", line.Nick)
		return
	}

	// Count every line as activity.
	my_histogram.LogActivity(line.Nick)

	// Check for greetings and say hello.
	if len(msg) < 10 && containsGreeting(msg) {
		if (time.Now().Unix() - lastGreetingTime.Unix()) < 30 {
			log.Printf("Not replying, 30 seconds have not passed yet")
			return
		}
		log.Printf(`Replying to line "%s"`, msg)
		lastGreetingTime = time.Now()
		go func() {
			// Reply after a random time, at least 1s after the
			// original line, at most 5s after the original line.
			fuzz := rand.Int63n(4000)
			time.Sleep((time.Duration)(1000+fuzz) * time.Millisecond)
			conn.Privmsg(*ircChannel, "Hello! Please be patient, as it may be some time before someone is around who can answer your question. In the meantime, please remember to read the user guide, as well as the current /topic")
		}()
		return
	}

	// Otherwise, see if that was a question.
	if !strings.HasSuffix(strings.TrimSpace(msg), "?") {
		return
	}

	log.Printf(`Got question "%s" from "%s"`, msg, line.Nick)

	// See if we have enough data points about the activity of this channel for
	// this day and hour.
	if my_histogram.IsActive() {
		return
	}

	log.Printf("Telling user to be patient\n")
}

func main() {
	flag.Parse()

	// Compile regular expressions which match the greeting words if they
	// appear as a standalone word.
	for _, greetword := range strings.SplitN(*greetings, ",", -1) {
		re := regexp.MustCompile(`\b` + greetword + `\b`)
		greetings_re = append(greetings_re, re)
	}

	my_histogram = histogram.Load(*histogram_path)

	quit := make(chan bool)

	c := irc.SimpleClient("Eyo", "i3", "http://i3wm.org/")

	c.HandleFunc(irc.CONNECTED,
		func(conn *irc.Conn, line *irc.Line) {
			log.Printf("Connected, joining channel %s\n", *ircChannel)
			conn.Join(*ircChannel)
		})

	c.HandleFunc("disconnected",
		func(conn *irc.Conn, line *irc.Line) { quit <- true })

	c.HandleFunc("PRIVMSG", handleMessage)

	log.Printf("Connecting...\n")

	if err := c.ConnectTo("chat.freenode.net", *ircPassword); err != nil {
		log.Printf("Connection error: %s\n", err.Error())
	}

	// program main loop
	for {
		select {
		case <-quit:
			log.Println("Disconnected. Reconnecting...")
			if err := c.ConnectTo("chat.freenode.net", *ircPassword); err != nil {
				log.Printf("Connection error: %s\n", err.Error())
			}
		}
	}
	log.Fatalln("Fell out of the main loop?!")
}
