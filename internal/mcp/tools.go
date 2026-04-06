package mcp

import (
	"context"
	"fmt"

	"github.com/XaaT/hfr-mcp/internal/hfr"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Input structs — le SDK derive le JSON schema automatiquement

type ReadInput struct {
	Cat      int  `json:"cat" jsonschema:"Numero de categorie HFR"`
	Post     int  `json:"post" jsonschema:"Numero du topic"`
	Page     *int `json:"page,omitempty" jsonschema:"Numero de page (defaut 1, 0 pour la derniere)"`
	PageFrom int  `json:"page_from,omitempty" jsonschema:"Debut de range (negatif = relatif a la fin, ex: -9)"`
	PageTo   int  `json:"page_to,omitempty" jsonschema:"Fin de range (0 = derniere page)"`
	Print    bool `json:"print,omitempty" jsonschema:"Mode impression: ~1000 posts/page, sans signatures"`
	Last     int  `json:"last,omitempty" jsonschema:"Garder seulement les N derniers posts (avec print)"`
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

type TopicsInput struct {
	Cat    int `json:"cat" jsonschema:"Numero de categorie HFR"`
	Subcat int `json:"subcat,omitempty" jsonschema:"Numero de sous-categorie (0 = toutes)"`
	Page   int `json:"page,omitempty" jsonschema:"Numero de page (defaut 1)"`
}

type QuoteInput struct {
	Cat         int   `json:"cat" jsonschema:"Numero de categorie HFR"`
	Post        int   `json:"post" jsonschema:"Numero du topic"`
	Numreponse  int   `json:"numreponse,omitempty" jsonschema:"Numero du message a citer (simple quote)"`
	Numreponses []int `json:"numreponses,omitempty" jsonschema:"Numeros des messages a citer (multiquote)"`
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
		Description: "Lire un topic HFR. page defaut 1 si omis, page=0 pour la derniere page. page_from/page_to pour lire plusieurs pages en parallele (valeurs negatives = relatif a la fin, 0 = derniere). print=true pour le mode impression (~1000 posts/page, sans signatures). last=N pour garder les N derniers posts (print uniquement).",
	}, handleRead(client, login))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "hfr_topics",
		Description: "Lister les topics d'une categorie HFR. Retourne titre, auteur, reponses, vues, dernier message. subcat=0 pour toutes les sous-categories.",
	}, handleTopics(client))

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
		Description: "Recuperer le BBCode de citation d'un ou plusieurs messages HFR. Utiliser numreponse pour un seul message, numreponses pour un multiquote.",
	}, handleQuote(client, login))
}

func handleRead(client *hfr.Client, login LoginFunc) mcp.ToolHandlerFor[ReadInput, Result] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ReadInput) (*mcp.CallToolResult, Result, error) {
		if err := login(); err != nil {
			return nil, Result{}, fmt.Errorf("login failed: %w", err)
		}

		var topic *hfr.Topic
		var err error

		page := 1
		if input.Page != nil {
			page = *input.Page
		}

		if input.Print {
			topic, err = client.ReadTopicPrint(input.Cat, input.Post, page, input.Last)
		} else if input.PageFrom != 0 || input.PageTo != 0 {
			topic, err = client.ReadTopicRange(input.Cat, input.Post, input.PageFrom, input.PageTo)
		} else {
			topic, err = client.ReadTopic(input.Cat, input.Post, page)
		}

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
		nums := input.Numreponses
		if len(nums) == 0 && input.Numreponse != 0 {
			nums = []int{input.Numreponse}
		}
		if len(nums) == 0 {
			return nil, Result{}, fmt.Errorf("numreponse or numreponses required")
		}
		bbcode, err := client.FetchQuote(input.Cat, input.Post, nums...)
		if err != nil {
			return nil, Result{}, fmt.Errorf("quote failed: %w", err)
		}
		return nil, Result{Message: bbcode}, nil
	}
}

func handleTopics(client *hfr.Client) mcp.ToolHandlerFor[TopicsInput, Result] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input TopicsInput) (*mcp.CallToolResult, Result, error) {
		page := input.Page
		if page < 1 {
			page = 1
		}
		list, err := client.ListTopics(input.Cat, input.Subcat, page)
		if err != nil {
			return nil, Result{}, fmt.Errorf("list topics failed: %w", err)
		}
		return nil, Result{Message: formatTopicList(list)}, nil
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
