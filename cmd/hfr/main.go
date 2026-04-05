package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/XaaT/hfr-mcp/internal/config"
	"github.com/XaaT/hfr-mcp/internal/hfr"
	hfrmcp "github.com/XaaT/hfr-mcp/internal/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const usage = `Usage: hfr [--auth] <command> [args...]

Commands:
  serve                                    Start MCP server (stdio)
  read   <cat> <post> [page|last|from:to]  Read a topic
  print  <cat> <post> [page] [--last N]    Read in print mode (~1000 posts/page, no signatures)
  reply  <cat> <post> <content>            Post a reply
  edit   <cat> <post> <numreponse> <content>  Edit a post
  quote  <cat> <post> <numreponse>         Get quote BBCode
  mp     <dest> <subject> <content>        Send a private message

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

	needsAuth := cmd != "read" && cmd != "print" && cmd != "serve"
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
	case "serve":
		cmdServe()
		return
	case "read":
		cmdRead(client, args)
	case "print":
		cmdPrint(client, args)
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
		die("usage: hfr read <cat> <post> [page|last|from-to|last-N:last]")
	}
	cat := mustInt(args[0], "cat")
	post := mustInt(args[1], "post")

	pageArg := "1"
	if len(args) >= 3 {
		pageArg = args[2]
	}

	var topic *hfr.Topic
	var err error

	switch {
	case pageArg == "last":
		topic, err = client.ReadTopic(cat, post, 0)
	case strings.Contains(pageArg, ":"):
		// Range: "340:350" or "last-10:last"
		from, to := parseRange(pageArg)
		topic, err = client.ReadTopicRange(cat, post, from, to)
	default:
		page := mustInt(pageArg, "page")
		topic, err = client.ReadTopic(cat, post, page)
	}

	if err != nil {
		die("read failed: %v", err)
	}

	fmt.Printf("Topic cat=%d post=%d page=%d/%d (%d posts)\n\n", topic.Cat, topic.Post, topic.Page, topic.TotalPages, len(topic.Posts))
	for _, p := range topic.Posts {
		fmt.Printf("--- #%d | %s | %s ---\n%s\n\n", p.Numreponse, p.Author, p.Date, strings.TrimSpace(p.Content))
	}
}

func cmdPrint(client *hfr.Client, args []string) {
	if len(args) < 2 {
		die("usage: hfr print <cat> <post> [page] [--last N]")
	}
	cat := mustInt(args[0], "cat")
	post := mustInt(args[1], "post")
	page := 0 // default: last print page
	last := 0 // default: all posts
	i := 2
	for i < len(args) {
		if args[i] == "--last" && i+1 < len(args) {
			last = mustInt(args[i+1], "last")
			i += 2
		} else {
			page = mustInt(args[i], "page")
			i++
		}
	}

	topic, err := client.ReadTopicPrint(cat, post, page, last)
	if err != nil {
		die("print read failed: %v", err)
	}

	fmt.Printf("Topic cat=%d post=%d print_page=%d/%d (%d posts)\n\n", topic.Cat, topic.Post, topic.Page, topic.TotalPages, len(topic.Posts))
	for _, p := range topic.Posts {
		fmt.Printf("--- #%d | %s | %s ---\n%s\n\n", p.Numreponse, p.Author, p.Date, strings.TrimSpace(p.Content))
	}
}

func parseRange(s string) (from, to int) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		die("invalid range: %q (expected from:to)", s)
	}
	return parsePageRef(parts[0]), parsePageRef(parts[1])
}

func parsePageRef(s string) int {
	s = strings.TrimSpace(s)
	if s == "last" {
		return 0
	}
	if strings.HasPrefix(s, "last-") {
		n := mustInt(strings.TrimPrefix(s, "last-"), "offset")
		return -n
	}
	return mustInt(s, "page")
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
		die("usage: hfr quote <cat> <post> <numreponse> [numreponse2 ...]")
	}
	cat := mustInt(args[0], "cat")
	post := mustInt(args[1], "post")
	var nums []int
	for _, a := range args[2:] {
		nums = append(nums, mustInt(a, "numreponse"))
	}

	bbcode, err := client.FetchQuote(cat, post, nums...)
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

func cmdServe() {
	cfg := config.Load()
	if cfg.Login == "" || cfg.Passwd == "" {
		die("credentials required: set login/passwd in hfr.conf or ~/.config/hfr/config, or HFR_LOGIN/HFR_PASSWD env vars")
	}

	client := hfr.NewClient()

	var loginOnce sync.Once
	var loginErr error
	lazyLogin := func() error {
		loginOnce.Do(func() {
			fmt.Fprintln(os.Stderr, "HFR: logging in as", cfg.Login)
			loginErr = client.Login(cfg.Login, cfg.Passwd)
			if loginErr != nil {
				fmt.Fprintln(os.Stderr, "HFR: login failed:", loginErr)
			} else {
				fmt.Fprintln(os.Stderr, "HFR: logged in as", cfg.Login)
			}
		})
		return loginErr
	}

	srv := mcp.NewServer(
		&mcp.Implementation{Name: "hfr", Version: "1.0.0"},
		nil,
	)

	hfrmcp.RegisterTools(srv, client, lazyLogin)

	fmt.Fprintln(os.Stderr, "HFR MCP server starting (stdio)")
	if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
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
