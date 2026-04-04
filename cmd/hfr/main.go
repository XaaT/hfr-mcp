package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/XaaT/hfr-mcp/internal/config"
	"github.com/XaaT/hfr-mcp/internal/hfr"
)

const usage = `Usage: hfr [--auth] <command> [args...]

Commands:
  read   <cat> <post> [page]              Read a topic
  reply  <cat> <post> <content>           Post a reply
  edit   <cat> <post> <numreponse> <content>  Edit a post
  quote  <cat> <post> <numreponse>        Get quote BBCode
  mp     <dest> <subject> <content>       Send a private message

Options:
  --auth    Login before executing. Required for reply, edit, quote, mp.

Config: ./hfr.conf or ~/.config/hfr/config (login=, passwd=)
Env vars HFR_LOGIN/HFR_PASSWD override config file.`

func main() {
	if len(os.Args) < 2 {
		die(usage)
	}

	args := os.Args[1:]
	auth := false
	if args[0] == "--auth" {
		auth = true
		args = args[1:]
	}

	if len(args) < 1 {
		die(usage)
	}

	cmd := args[0]
	args = args[1:]

	needsAuth := cmd != "read"
	if needsAuth {
		auth = true
	}

	client := hfr.NewClient()

	if auth {
		cfg := config.Load()
		if cfg.Login == "" || cfg.Passwd == "" {
			die("credentials required: set login/passwd in hfr.conf or ~/.config/hfr/config, or HFR_LOGIN/HFR_PASSWD env vars")
		}
		if err := client.Login(cfg.Login, cfg.Passwd); err != nil {
			die("login failed: %v", err)
		}
	}

	switch cmd {
	case "read":
		cmdRead(client, args)
	case "reply":
		cmdReply(client, args)
	case "edit":
		cmdEdit(client, args)
	case "quote":
		cmdQuote(client, args)
	case "mp":
		cmdMP(client, args)
	default:
		die("unknown command: %s\n\n%s", cmd, usage)
	}
}

func cmdRead(client *hfr.Client, args []string) {
	if len(args) < 2 {
		die("usage: hfr read <cat> <post> [page]")
	}
	cat := mustInt(args[0], "cat")
	post := mustInt(args[1], "post")
	page := 1
	if len(args) >= 3 {
		page = mustInt(args[2], "page")
	}

	topic, err := client.ReadTopic(cat, post, page)
	if err != nil {
		die("read failed: %v", err)
	}

	fmt.Printf("Topic cat=%d post=%d page=%d (%d posts)\n\n", topic.Cat, topic.Post, topic.Page, len(topic.Posts))
	for _, p := range topic.Posts {
		fmt.Printf("--- #%d | %s | %s ---\n%s\n\n", p.Numreponse, p.Author, p.Date, strings.TrimSpace(p.Content))
	}
}

func cmdReply(client *hfr.Client, args []string) {
	if len(args) < 3 {
		die("usage: hfr reply <cat> <post> <content>")
	}
	cat := mustInt(args[0], "cat")
	post := mustInt(args[1], "post")
	content := strings.Join(args[2:], " ")

	if err := client.Reply(cat, post, content); err != nil {
		die("reply failed: %v", err)
	}
	fmt.Println("Reply posted.")
}

func cmdEdit(client *hfr.Client, args []string) {
	if len(args) < 4 {
		die("usage: hfr edit <cat> <post> <numreponse> <content>")
	}
	cat := mustInt(args[0], "cat")
	post := mustInt(args[1], "post")
	numreponse := mustInt(args[2], "numreponse")
	content := strings.Join(args[3:], " ")

	if err := client.Edit(cat, post, numreponse, content); err != nil {
		die("edit failed: %v", err)
	}
	fmt.Println("Post edited.")
}

func cmdQuote(client *hfr.Client, args []string) {
	if len(args) < 3 {
		die("usage: hfr quote <cat> <post> <numreponse>")
	}
	cat := mustInt(args[0], "cat")
	post := mustInt(args[1], "post")
	numreponse := mustInt(args[2], "numreponse")

	bbcode, err := client.FetchQuote(cat, post, numreponse)
	if err != nil {
		die("quote failed: %v", err)
	}
	fmt.Println(bbcode)
}

func cmdMP(client *hfr.Client, args []string) {
	if len(args) < 3 {
		die("usage: hfr mp <dest> <subject> <content>")
	}
	dest := args[0]
	subject := args[1]
	content := strings.Join(args[2:], " ")

	if err := client.SendMP(dest, subject, content); err != nil {
		die("mp failed: %v", err)
	}
	fmt.Println("MP sent.")
}

func mustInt(s, name string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		die("invalid %s: %q (expected integer)", name, s)
	}
	return n
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
