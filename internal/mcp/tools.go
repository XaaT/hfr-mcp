package mcp

import (
	"context"
	"fmt"

	"github.com/XaaT/hfr-mcp/internal/hfr"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Input structs — le SDK derive le JSON schema automatiquement

type ReadInput struct {
	Cat  int `json:"cat" jsonschema:"Numero de categorie HFR"`
	Post int `json:"post" jsonschema:"Numero du topic"`
	Page int `json:"page,omitempty" jsonschema:"Numero de page (defaut 1)"`
}

type ReplyInput struct {
	Cat     int    `json:"cat" jsonschema:"Numero de categorie HFR"`
	Post    int    `json:"post" jsonschema:"Numero du topic"`
	Content string `json:"content" jsonschema:"Contenu du message en BBCode HFR"`
}

type EditInput struct {
	Cat        int    `json:"cat" jsonschema:"Numero de categorie HFR"`
	Post       int    `json:"post" jsonschema:"Numero du topic"`
	Numreponse int    `json:"numreponse" jsonschema:"Numero du message a editer"`
	Content    string `json:"content" jsonschema:"Nouveau contenu en BBCode HFR"`
}

type MPInput struct {
	Dest    string `json:"dest" jsonschema:"Pseudo du destinataire"`
	Subject string `json:"subject" jsonschema:"Sujet du MP"`
	Content string `json:"content" jsonschema:"Contenu du message en BBCode HFR"`
}

type QuoteInput struct {
	Cat        int `json:"cat" jsonschema:"Numero de categorie HFR"`
	Post       int `json:"post" jsonschema:"Numero du topic"`
	Numreponse int `json:"numreponse" jsonschema:"Numero du message a citer"`
}

// Output struct
type Result struct {
	Message string `json:"message"`
}

// LoginFunc is called before each tool to ensure the client is logged in
type LoginFunc func() error

// RegisterTools adds all HFR tools to the MCP server
func RegisterTools(srv *mcp.Server, client *hfr.Client, login LoginFunc) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "hfr_read",
		Description: "Lire un topic HFR. Retourne les posts de la page demandee.",
	}, handleRead(client, login))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "hfr_reply",
		Description: "Poster une reponse sur un topic HFR.",
	}, handleReply(client, login))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "hfr_edit",
		Description: "Editer un post existant sur HFR.",
	}, handleEdit(client, login))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "hfr_mp",
		Description: "Envoyer un message prive sur HFR.",
	}, handleMP(client, login))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "hfr_quote",
		Description: "Recuperer le BBCode de citation d'un message HFR. A utiliser avant hfr_reply pour citer correctement.",
	}, handleQuote(client, login))
}

func handleRead(client *hfr.Client, login LoginFunc) mcp.ToolHandlerFor[ReadInput, Result] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ReadInput) (*mcp.CallToolResult, Result, error) {
		if err := login(); err != nil {
			return nil, Result{}, fmt.Errorf("login failed: %w", err)
		}
		page := input.Page
		if page == 0 {
			page = 1
		}
		topic, err := client.ReadTopic(input.Cat, input.Post, page)
		if err != nil {
			return nil, Result{}, fmt.Errorf("read failed: %w", err)
		}
		return nil, Result{Message: formatTopic(topic)}, nil
	}
}

func handleReply(client *hfr.Client, login LoginFunc) mcp.ToolHandlerFor[ReplyInput, Result] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ReplyInput) (*mcp.CallToolResult, Result, error) {
		if err := login(); err != nil {
			return nil, Result{}, fmt.Errorf("login failed: %w", err)
		}
		if err := client.Reply(input.Cat, input.Post, input.Content); err != nil {
			return nil, Result{}, fmt.Errorf("reply failed: %w", err)
		}
		return nil, Result{Message: "Message poste avec succes."}, nil
	}
}

func handleEdit(client *hfr.Client, login LoginFunc) mcp.ToolHandlerFor[EditInput, Result] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input EditInput) (*mcp.CallToolResult, Result, error) {
		if err := login(); err != nil {
			return nil, Result{}, fmt.Errorf("login failed: %w", err)
		}
		if err := client.Edit(input.Cat, input.Post, input.Numreponse, input.Content); err != nil {
			return nil, Result{}, fmt.Errorf("edit failed: %w", err)
		}
		return nil, Result{Message: "Message edite avec succes."}, nil
	}
}

func handleQuote(client *hfr.Client, login LoginFunc) mcp.ToolHandlerFor[QuoteInput, Result] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input QuoteInput) (*mcp.CallToolResult, Result, error) {
		if err := login(); err != nil {
			return nil, Result{}, fmt.Errorf("login failed: %w", err)
		}
		bbcode, err := client.FetchQuote(input.Cat, input.Post, input.Numreponse)
		if err != nil {
			return nil, Result{}, fmt.Errorf("quote failed: %w", err)
		}
		return nil, Result{Message: bbcode}, nil
	}
}

func handleMP(client *hfr.Client, login LoginFunc) mcp.ToolHandlerFor[MPInput, Result] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input MPInput) (*mcp.CallToolResult, Result, error) {
		if err := login(); err != nil {
			return nil, Result{}, fmt.Errorf("login failed: %w", err)
		}
		if err := client.SendMP(input.Dest, input.Subject, input.Content); err != nil {
			return nil, Result{}, fmt.Errorf("mp failed: %w", err)
		}
		return nil, Result{Message: "MP envoye avec succes."}, nil
	}
}
